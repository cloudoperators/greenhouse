// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const supportGroupClaimPrefix = "support-group:"

func handleAuthorize(w http.ResponseWriter, r *http.Request, c client.Client, mapper meta.RESTMapper) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	var review authv1.SubjectAccessReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		authzDeniedTotal.WithLabelValues(reasonDecodeError).Inc()
		http.Error(w, fmt.Sprintf("decode: %v", err), http.StatusBadRequest)
		return
	}

	attrs := review.Spec.ResourceAttributes
	if attrs == nil || attrs.Name == "" {
		recordDenied("", "", reasonMissingAttributes, nil)
		respond(w, review, false, "missing resource attributes")
		return
	}

	verb := attrs.Verb

	// Get support-groups from identity (either SA team or user claims)
	supportGroups, reasonCode, err := getSupportGroups(ctx, c, review.Spec.User, review.Spec.Groups, attrs.Namespace)
	if err != nil {
		logger.Info("Authorization denied",
			"user", review.Spec.User,
			"verb", verb,
			"resource", attrs.Resource,
			"namespace", attrs.Namespace,
			"allowed", false,
			"reason", err.Error())
		recordDenied(verb, attrs.Resource, reasonCode, nil)
		respond(w, review, false, err.Error())
		return
	}

	// Authorize access using unified flow
	allowed, reasonCode, ownedByValue, message := authorizeAccess(ctx, c, mapper, attrs, supportGroups)

	// Log authorization decision
	logger.Info("Authorization decision",
		"user", review.Spec.User,
		"verb", verb,
		"resource", attrs.Resource,
		"namespace", attrs.Namespace,
		"allowed", allowed,
		"ownedBy", ownedByValue,
		"supportGroups", supportGroups,
		"reason", message)

	// Record metrics and respond
	if allowed {
		recordAllowed(verb, attrs.Resource, ownedByValue)
		respond(w, review, true, message)
	} else {
		recordDenied(verb, attrs.Resource, reasonCode, supportGroups)
		respond(w, review, false, message)
	}
}

// getSupportGroups gets support-groups from the request identity.
// For ServiceAccounts, it fetches the SA and extracts its team from the owned-by label.
// For users, it extracts support-group claims from the groups.
func getSupportGroups(
	ctx context.Context, c client.Client, username string, groups []string, namespace string,
) (supportGroups []string, reasonCode string, err error) {

	// Check if this is a ServiceAccount
	serviceAccountName := extractServiceAccountName(username, namespace)
	if serviceAccountName != "" {
		// Fetch ServiceAccount
		sa := &corev1.ServiceAccount{}
		saKey := types.NamespacedName{
			Namespace: namespace,
			Name:      serviceAccountName,
		}
		if err := c.Get(ctx, saKey, sa); err != nil {
			return nil, reasonServiceAccountNotFound, fmt.Errorf("ServiceAccount %s not found", serviceAccountName)
		}

		// Extract team name from SA's owned-by label
		teamName, ok := sa.Labels[greenhouseapis.LabelKeyOwnedBy]
		if !ok {
			return nil, reasonNoOwnedByLabel, errors.New("ServiceAccount missing owned-by label")
		}

		return []string{teamName}, "", nil
	}

	// Extract support-group claims from user groups
	var userSupportGroups []string
	for _, group := range groups {
		supportGroupName, found := strings.CutPrefix(group, supportGroupClaimPrefix)
		if found {
			userSupportGroups = append(userSupportGroups, supportGroupName)
		}
	}

	if len(userSupportGroups) == 0 {
		return nil, reasonNoSupportGroupClaims, errors.New("user has no support-group claims and is not an authorized ServiceAccount")
	}

	return userSupportGroups, "", nil
}

// authorizeAccess performs the unified authorization flow for both users and ServiceAccounts.
// It fetches the resource, validates the Team, and checks if the identity is authorized.
func authorizeAccess(
	ctx context.Context, c client.Client, mapper meta.RESTMapper, attrs *authv1.ResourceAttributes, supportGroups []string,
) (allowed bool, reasonCode, ownedByValue, message string) {

	// 1. Fetch resource and extract owned-by label
	ownedByValue, reasonCode, err := fetchResourceWithOwnership(ctx, c, mapper, attrs)
	if err != nil {
		return false, reasonCode, "", fmt.Sprintf("failed to fetch object: %v", err)
	}

	// 2. Validate Team exists and is a support-group
	reasonCode, err = validateTeam(ctx, c, attrs.Namespace, ownedByValue)
	if err != nil {
		return false, reasonCode, "", err.Error()
	}

	// 3. Check if any support-group matches the resource owner
	if slices.Contains(supportGroups, ownedByValue) {
		return true, "", ownedByValue, "authorized to access resource owned by " + ownedByValue
	}

	// No matching support-group found
	return false, reasonSupportGroupMismatch, "", "support-group does not match resource owner"
}

// fetchResourceWithOwnership fetches a resource and extracts its owned-by label.
func fetchResourceWithOwnership(
	ctx context.Context, c client.Client, mapper meta.RESTMapper, attrs *authv1.ResourceAttributes,
) (ownedByValue, reasonCode string, err error) {

	gvr := schema.GroupVersionResource{
		Group:    attrs.Group,
		Resource: attrs.Resource,
	}
	if attrs.Version != "" && attrs.Version != "*" {
		gvr.Version = attrs.Version
	}

	gvk, err := mapper.KindFor(gvr)
	if err != nil {
		authzKindResolutionErrorsTotal.Inc()
		return "", reasonKindResolutionFailed, err
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	key := types.NamespacedName{Namespace: attrs.Namespace, Name: attrs.Name}
	if err := c.Get(ctx, key, obj); err != nil {
		authzKubeFetchErrorsTotal.WithLabelValues(err.Error()).Inc()
		return "", reasonObjectNotFound, err
	}

	labels := obj.GetLabels()
	ownedByValue, ok := labels[greenhouseapis.LabelKeyOwnedBy]
	if !ok {
		return "", reasonNoOwnedByLabel, errors.New("resource has no owned-by label")
	}

	return ownedByValue, "", nil
}

// validateTeam checks that the Team referenced in the owned-by label exists and is a support-group.
func validateTeam(ctx context.Context, c client.Client, namespace, teamName string) (reasonCode string, err error) {
	team := &greenhousev1alpha1.Team{}
	teamKey := types.NamespacedName{Namespace: namespace, Name: teamName}
	if err := c.Get(ctx, teamKey, team); err != nil {
		authzKubeFetchErrorsTotal.WithLabelValues(err.Error()).Inc()
		return reasonObjectNotFound, fmt.Errorf("team %s referenced in owned-by label does not exist: %w", teamName, err)
	}

	// Validate that the Team is marked as a support-group
	supportGroup, ok := team.Labels[greenhouseapis.LabelKeySupportGroup]
	if !ok || supportGroup != "true" {
		return reasonObjectNotFound, fmt.Errorf("team %s is not a support-group", teamName)
	}

	return "", nil
}

// extractServiceAccountName extracts the ServiceAccount name from the username
// if it's in the format system:serviceaccount:{namespace}:{serviceaccount-name}
func extractServiceAccountName(username, namespace string) string {
	// ServiceAccount username format: system:serviceaccount:{namespace}:{serviceaccount-name}
	prefix := "system:serviceaccount:" + namespace + ":"
	serviceAccountName, found := strings.CutPrefix(username, prefix)
	if !found {
		return ""
	}
	return serviceAccountName
}

func respond(w http.ResponseWriter, review authv1.SubjectAccessReview, allowed bool, msg string) {
	review.Status = authv1.SubjectAccessReviewStatus{
		Allowed: allowed,
		Reason:  msg,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(review); err != nil {
		logger.Error(err, "encode SubjectAccessReview failed")
	}
}

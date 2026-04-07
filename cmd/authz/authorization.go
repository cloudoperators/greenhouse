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
	logger.Info("Received request", "user", review.Spec.User, "verb", verb, "resource", attrs.Resource, "ns", attrs.Namespace)

	// Check if the request is from a ServiceAccount
	serviceAccountName := extractServiceAccountName(review.Spec.User, attrs.Namespace)
	if serviceAccountName != "" {
		logger.Info("Request is from ServiceAccount", "serviceAccount", serviceAccountName)
		allowed, reasonCode, ownedByValue := authorizeServiceAccount(ctx, c, mapper, attrs, serviceAccountName)
		if allowed {
			recordAllowed(verb, attrs.Resource, ownedByValue)
			respond(w, review, true, fmt.Sprintf("ServiceAccount %s authorized to access resource owned by %s", serviceAccountName, ownedByValue))
			return
		}
		// If ServiceAccount authorization fails, deny the request
		recordDenied(verb, attrs.Resource, reasonCode, nil)
		var denyMsg string
		switch reasonCode {
		case reasonServiceAccountNotFound:
			denyMsg = fmt.Sprintf("ServiceAccount %s not found", serviceAccountName)
		case reasonNoOwnedByLabel:
			denyMsg = "ServiceAccount or resource missing owned-by label"
		case reasonSupportGroupMismatch:
			denyMsg = "ServiceAccount team does not match resource owner"
		default:
			denyMsg = "authorization failed: " + reasonCode
		}
		respond(w, review, false, denyMsg)
		return
	}

	var userSupportGroups []string
	for _, group := range review.Spec.Groups {
		supportGroupName, found := strings.CutPrefix(group, supportGroupClaimPrefix)
		if found {
			userSupportGroups = append(userSupportGroups, supportGroupName)
		}
	}

	if len(userSupportGroups) == 0 {
		recordDenied(verb, attrs.Resource, reasonNoSupportGroupClaims, nil)
		respond(w, review, false, "user has no support-group claims and is not an authorized ServiceAccount")
		return
	}
	logger.Info("User has the following support-group claims: " + strings.Join(userSupportGroups, ", "))

	// Fetch the resource and extract its owned-by label
	ownedByValue, reasonCode, err := fetchResourceWithOwnership(ctx, c, mapper, attrs)
	if err != nil {
		recordDenied(verb, attrs.Resource, reasonCode, userSupportGroups)
		respond(w, review, false, fmt.Sprintf("failed to fetch object: %v", err))
		return
	}
	logger.Info("Requested resource is owned by: " + ownedByValue)

	// If the support-group matches the greenhouse.sap/owned-by label on the resources the user should get full permissions on the resource.
	if slices.Contains(userSupportGroups, ownedByValue) {
		recordAllowed(verb, attrs.Resource, ownedByValue)
		respond(w, review, true, "user has a support-group claim for the requested resource: "+ownedByValue)
		return
	}

	recordDenied(verb, attrs.Resource, reasonSupportGroupMismatch, userSupportGroups)
	respond(w, review, false, "")
}

// fetchResourceWithOwnership fetches a resource and extracts its owned-by label.
// Returns the ownedByValue on success, or an error with a reason code on failure.
func fetchResourceWithOwnership(ctx context.Context, c client.Client, mapper meta.RESTMapper, attrs *authv1.ResourceAttributes) (ownedByValue, reasonCode string, err error) {
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

// authorizeServiceAccount checks if a ServiceAccount is authorized to access the resource.
// Returns allowed status, reason code (from metrics.go constants), and ownedByValue.
func authorizeServiceAccount(ctx context.Context, c client.Client, mapper meta.RESTMapper, attrs *authv1.ResourceAttributes, serviceAccountName string) (allowed bool, reasonCode, ownedByValue string) {
	// 1. Fetch ServiceAccount and verify its owned-by label
	sa := &corev1.ServiceAccount{}
	saKey := types.NamespacedName{
		Namespace: attrs.Namespace,
		Name:      serviceAccountName,
	}
	if err := c.Get(ctx, saKey, sa); err != nil {
		return false, reasonServiceAccountNotFound, ""
	}

	teamName, ok := sa.Labels[greenhouseapis.LabelKeyOwnedBy]
	if !ok {
		return false, reasonNoOwnedByLabel, ""
	}
	logger.Info("ServiceAccount ownership verified", "serviceAccountName", sa.Name, "team", teamName)

	// 2. Fetch the resource and extract its owned-by label using shared helper
	ownedByValue, reasonCode, err := fetchResourceWithOwnership(ctx, c, mapper, attrs)
	if err != nil {
		return false, reasonCode, ""
	}
	logger.Info("Requested resource is owned by: " + ownedByValue)

	// 3. Check if the ServiceAccount's team matches the resource's owned-by label
	if teamName == ownedByValue {
		return true, "", ownedByValue
	}

	logger.Info("ServiceAccount team does not match resource owner", "serviceAccountTeam", teamName, "resourceOwner", ownedByValue)
	return false, reasonSupportGroupMismatch, ""
}

func respond(w http.ResponseWriter, review authv1.SubjectAccessReview, allowed bool, msg string) {
	if allowed {
		logger.Info("[ALLOWED] " + msg)
	} else {
		logger.Info("[DENIED] " + msg)
	}
	review.Status = authv1.SubjectAccessReviewStatus{
		Allowed: allowed,
		Reason:  msg,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(review); err != nil {
		logger.Error(err, "encode SubjectAccessReview failed")
	}
}

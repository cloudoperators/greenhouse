// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
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
	logger.Info("Received request", "user", review.Spec.User, "verb", verb, "resource", attrs.Resource, "ns", attrs.Namespace)

	var userSupportGroups []string
	for _, group := range review.Spec.Groups {
		supportGroupName, found := strings.CutPrefix(group, supportGroupClaimPrefix)
		if found {
			userSupportGroups = append(userSupportGroups, supportGroupName)
		}
	}

	if len(userSupportGroups) == 0 {
		recordDenied(verb, attrs.Resource, reasonNoSupportGroupClaims, nil)
		respond(w, review, false, "user has no support-group claims")
		return
	}
	logger.Info("User has the following support-group claims: " + strings.Join(userSupportGroups, ", "))

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
		recordDenied(verb, attrs.Resource, reasonKindResolutionFailed, userSupportGroups)
		respond(w, review, false, "failed to get Kind for the requested resource")
		return
	}

	// Try to fetch the resource
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	key := types.NamespacedName{Namespace: attrs.Namespace, Name: attrs.Name}
	if err := c.Get(ctx, key, obj); err != nil {
		authzKubeFetchErrorsTotal.WithLabelValues(err.Error()).Inc()
		recordDenied(verb, attrs.Resource, reasonObjectNotFound, userSupportGroups)
		respond(w, review, false, fmt.Sprintf("failed to fetch object: %v", err))
		return
	}

	labels := obj.GetLabels()
	ownedByValue, ok := labels[greenhouseapis.LabelKeyOwnedBy]
	if !ok {
		recordDenied(verb, attrs.Resource, reasonNoOwnedByLabel, userSupportGroups)
		respond(w, review, false, "requested resource has no owned-by label set")
		return
	}
	logger.Info("Requested resource is owned by: " + ownedByValue)

	// Validate that the Team referenced in the owned-by label actually exists and is a support-group.
	// Optimization: if the resource being accessed is the Team itself, we've already validated its existence.
	if gvk.Kind != "Team" || attrs.Name != ownedByValue {
		team := &greenhousev1alpha1.Team{}
		teamKey := types.NamespacedName{Namespace: attrs.Namespace, Name: ownedByValue}
		if err := c.Get(ctx, teamKey, team); err != nil {
			authzKubeFetchErrorsTotal.WithLabelValues(err.Error()).Inc()
			recordDenied(verb, attrs.Resource, reasonObjectNotFound, userSupportGroups)
			respond(w, review, false, fmt.Sprintf("team %s referenced in owned-by label does not exist: %v", ownedByValue, err))
			return
		}
		// Validate that the Team is marked as a support-group.
		supportGroup, ok := team.Labels[greenhouseapis.LabelKeySupportGroup]
		if !ok || supportGroup != "true" {
			recordDenied(verb, attrs.Resource, reasonObjectNotFound, userSupportGroups)
			respond(w, review, false, fmt.Sprintf("team %s is not a support-group", ownedByValue))
			return
		}
	}

	// If the support-group matches the greenhouse.sap/owned-by label on the resources the user should get full permissions on the resource.
	if slices.Contains(userSupportGroups, ownedByValue) {
		recordAllowed(verb, attrs.Resource, ownedByValue)
		respond(w, review, true, "user has a support-group claim for the requested resource: "+ownedByValue)
		return
	}

	recordDenied(verb, attrs.Resource, reasonSupportGroupMismatch, userSupportGroups)
	respond(w, review, false, "")
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

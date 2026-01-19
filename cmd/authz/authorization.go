// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log"
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
)

func handleAuthorize(w http.ResponseWriter, r *http.Request, c client.Client, mapper meta.RESTMapper) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	var review authv1.SubjectAccessReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, fmt.Sprintf("decode: %v", err), http.StatusBadRequest)
		return
	}

	attrs := review.Spec.ResourceAttributes
	if attrs == nil || attrs.Name == "" {
		respond(w, review, false, "missing resource attributes")
		return
	}

	log.Printf("[AuthZ] Request for user=%s, verb=%s, resource=%s, ns=%s",
		review.Spec.User, attrs.Verb, attrs.Resource, attrs.Namespace)

	var userSupportGroups []string
	supportGroupClaimPrefix := "support-group:"
	for _, group := range review.Spec.Groups {
		supportGroupName, found := strings.CutPrefix(group, supportGroupClaimPrefix)
		if found {
			userSupportGroups = append(userSupportGroups, supportGroupName)
		}
	}

	if len(userSupportGroups) == 0 {
		respond(w, review, false, "user has no support-group claims")
		return
	}
	log.Printf("[AuthZ] User has the following support-group claims: %s", strings.Join(userSupportGroups, ", "))

	gvr := schema.GroupVersionResource{
		Group:    attrs.Group,
		Version:  attrs.Version,
		Resource: attrs.Resource,
	}
	gvk, err := mapper.KindFor(gvr)
	if err != nil {
		respond(w, review, false, "failed to get Kind for the requested resource")
	}

	// Try to fetch the resource
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	key := types.NamespacedName{Namespace: attrs.Namespace, Name: attrs.Name}
	if err := c.Get(ctx, key, obj); err != nil {
		respond(w, review, false, fmt.Sprintf("failed to fetch object: %v", err))
		return
	}

	labels := obj.GetLabels()
	ownedByValue, ok := labels[greenhouseapis.LabelKeyOwnedBy]
	if !ok {
		respond(w, review, false, "requested resource has no owned-by label set")
		return
	}
	log.Printf("[AuthZ] Requested resource is owned by: %s", ownedByValue)

	// If the support-group matches the greenhouse.sap/owned-by label on the resources the user should get full permissions on the resource.
	if slices.Contains(userSupportGroups, ownedByValue) {
		respond(w, review, true, "user has a support-group claim for the requested resource: "+ownedByValue)
		return
	}

	respond(w, review, false, "")
}

func respond(w http.ResponseWriter, review authv1.SubjectAccessReview, allowed bool, msg string) {
	if allowed {
		log.Printf("[AuthZ ALLOWED] %s", msg)
	} else {
		log.Printf("[AuthZ DENIED] %s", msg)
	}
	review.Status = authv1.SubjectAccessReviewStatus{
		Allowed: allowed,
		Reason:  msg,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(review); err != nil {
		log.Printf("encode SubjectAccessReview failed: %v", err)
	}
}

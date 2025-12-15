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

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func handleAuthorize(w http.ResponseWriter, r *http.Request, dyn dynamic.Interface) {
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

	log.Printf("AuthZ request for user=%s, verb=%s, resource=%s, ns=%s",
		review.Spec.User, attrs.Verb, attrs.Resource, attrs.Namespace)

	var userSupportGroups []string
	supportGroupClaimPrefix := "support-group:"
	for _, group := range review.Spec.Groups {
		supportGroupName, found := strings.CutPrefix(group, supportGroupClaimPrefix)
		if !found {
			continue
		}

		log.Printf("AuthZ found support-group claim: %s", supportGroupName)
		userSupportGroups = append(userSupportGroups, supportGroupName)
	}

	if len(userSupportGroups) == 0 {
		respond(w, review, false, "user has no support-group claims")
	}

	gvr := schema.GroupVersionResource{
		Group:    attrs.Group,
		Version:  attrs.Version,
		Resource: attrs.Resource,
	}

	// Try to fetch the resource
	var obj *unstructured.Unstructured
	var err error
	if attrs.Namespace != "" {
		obj, err = dyn.Resource(gvr).Namespace(attrs.Namespace).
			Get(ctx, attrs.Name, metav1.GetOptions{})
	} else {
		obj, err = dyn.Resource(gvr).
			Get(ctx, attrs.Name, metav1.GetOptions{})
	}

	if err != nil {
		respond(w, review, false, fmt.Sprintf("failed to fetch object: %v", err))
		return
	}

	labels := obj.GetLabels()
	ownedByValue, ok := labels[greenhouseapis.LabelKeyOwnedBy]
	if !ok {
		respond(w, review, false, "requested resource has no owned-by label set")
		return
	}

	// If the support-group matches the greenhouse.sap/owned-by label on the resources the user should get full permissions on the resource.
	if slices.Contains(userSupportGroups, ownedByValue) {
		respond(w, review, true, fmt.Sprintf("user has a support-group claim for the requested resource: %s", ownedByValue))
	}

	respond(w, review, false, "")
}

func respond(w http.ResponseWriter, review authv1.SubjectAccessReview, allowed bool, msg string) {
	review.Status = authv1.SubjectAccessReviewStatus{
		Allowed: allowed,
		Reason:  msg,
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(review)
	http.Error(w, fmt.Sprintf("encode: %v", err), http.StatusInternalServerError)
}

func handleAuthorizeDummy(w http.ResponseWriter, r *http.Request) {
	log.Println("[DUMMY] Authz webhook allowing all requests")

	if r.TLS == nil {
		fmt.Println("TLS: no (unexpected for https)")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("TLS: peer certs=%d, verifiedChains=%d\n",
		len(r.TLS.PeerCertificates), len(r.TLS.VerifiedChains))

	if len(r.TLS.PeerCertificates) > 0 {
		c := r.TLS.PeerCertificates[0]
		log.Printf("Client cert subject: %s\n", c.Subject.String())
		log.Printf("Client cert issuer:  %s\n", c.Issuer.String())
	}

	if len(r.TLS.VerifiedChains) == 0 {
		log.Println("Client cert NOT verified (or not required)")
	} else {
		log.Println("Client cert VERIFIED")
	}

	review := authv1.SubjectAccessReview{
		Status: authv1.SubjectAccessReviewStatus{
			Allowed: true,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(review)
}

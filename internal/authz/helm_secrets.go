// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
)

const helmReleaseSecretPrefix = "sh.helm.release.v1." //nolint:gosec // this is a constant prefix used by Helm for release secrets

func (a *authorizer) authorizeHelmSecret(ctx context.Context, w http.ResponseWriter, review authv1.SubjectAccessReview, attrs *authv1.ResourceAttributes) {
	verb := attrs.Verb

	switch verb {
	case "list":
		if !hasHelmOwnerSelector(attrs.LabelSelector) {
			a.logger.Info("Authorization decision",
				"user", review.Spec.User,
				"verb", verb,
				"resource", attrs.Resource,
				"namespace", attrs.Namespace,
				"allowed", false,
				"reason", "listing secrets is only permitted for helm release storage")
			recordDenied(verb, attrs.Resource, reasonSupportGroupMismatch, nil)
			a.respond(w, review, false, "listing secrets is only permitted for helm release storage")
			return
		}
		a.logger.Info("Authorization decision",
			"user", review.Spec.User,
			"verb", verb,
			"resource", attrs.Resource,
			"namespace", attrs.Namespace,
			"allowed", true,
			"reason", "authorized to list helm release secrets")
		recordAllowed(verb, attrs.Resource, "")
		a.respond(w, review, true, "authorized to list helm release secrets")

	case "create":
		supportGroups, reasonCode, err := a.getSupportGroups(ctx, review.Spec.User, review.Spec.Groups, attrs.Namespace)
		if err != nil {
			a.logger.Info("Authorization decision",
				"user", review.Spec.User,
				"verb", verb,
				"resource", attrs.Resource,
				"namespace", attrs.Namespace,
				"allowed", false,
				"reason", err.Error())
			recordDenied(verb, attrs.Resource, reasonCode, nil)
			a.respond(w, review, false, err.Error())
			return
		}
		if len(supportGroups) == 0 {
			a.logger.Info("Authorization decision",
				"user", review.Spec.User,
				"verb", verb,
				"resource", attrs.Resource,
				"namespace", attrs.Namespace,
				"allowed", false,
				"reason", "caller is not a member of any support-group")
			recordDenied(verb, attrs.Resource, reasonSupportGroupMismatch, supportGroups)
			a.respond(w, review, false, "caller is not a member of any support-group")
			return
		}
		a.logger.Info("Authorization decision",
			"user", review.Spec.User,
			"verb", verb,
			"resource", attrs.Resource,
			"namespace", attrs.Namespace,
			"supportGroups", supportGroups,
			"allowed", true,
			"reason", "authorized to create secret as support-group member")
		recordAllowed(verb, attrs.Resource, "")
		a.respond(w, review, true, "authorized to create secret as support-group member")

	case "get", "update":
		if !strings.HasPrefix(attrs.Name, helmReleaseSecretPrefix) {
			a.logger.Info("Authorization decision",
				"user", review.Spec.User,
				"verb", verb,
				"resource", attrs.Resource,
				"namespace", attrs.Namespace,
				"name", attrs.Name,
				"allowed", false,
				"reason", "secret is not helm release storage")
			recordDenied(verb, attrs.Resource, reasonSupportGroupMismatch, nil)
			a.respond(w, review, false, "secret is not helm release storage (expected name prefix "+helmReleaseSecretPrefix+")")
			return
		}

		ownedBy, reasonCode, err := a.secretOwnedBy(ctx, attrs.Namespace, attrs.Name)
		if err != nil {
			a.logger.Info("Authorization decision",
				"user", review.Spec.User,
				"verb", verb,
				"resource", attrs.Resource,
				"namespace", attrs.Namespace,
				"name", attrs.Name,
				"allowed", false,
				"reason", err.Error())
			recordDenied(verb, attrs.Resource, reasonCode, nil)
			a.respond(w, review, false, err.Error())
			return
		}

		supportGroups, reasonCode, err := a.getSupportGroups(ctx, review.Spec.User, review.Spec.Groups, attrs.Namespace)
		if err != nil {
			a.logger.Info("Authorization decision",
				"user", review.Spec.User,
				"verb", verb,
				"resource", attrs.Resource,
				"namespace", attrs.Namespace,
				"ownedBy", ownedBy,
				"allowed", false,
				"reason", err.Error())
			recordDenied(verb, attrs.Resource, reasonCode, nil)
			a.respond(w, review, false, err.Error())
			return
		}

		if !slices.Contains(supportGroups, ownedBy) {
			a.logger.Info("Authorization decision",
				"user", review.Spec.User,
				"verb", verb,
				"resource", attrs.Resource,
				"namespace", attrs.Namespace,
				"ownedBy", ownedBy,
				"supportGroups", supportGroups,
				"allowed", false,
				"reason", "support-group does not match secret owner")
			recordDenied(verb, attrs.Resource, reasonSupportGroupMismatch, supportGroups)
			a.respond(w, review, false, "support-group does not match secret owner "+ownedBy)
			return
		}

		if reasonCode, err := a.validateTeam(ctx, attrs.Namespace, ownedBy); err != nil {
			a.logger.Info("Authorization decision",
				"user", review.Spec.User,
				"verb", verb,
				"resource", attrs.Resource,
				"namespace", attrs.Namespace,
				"ownedBy", ownedBy,
				"supportGroups", supportGroups,
				"allowed", false,
				"reason", err.Error())
			recordDenied(verb, attrs.Resource, reasonCode, supportGroups)
			a.respond(w, review, false, err.Error())
			return
		}

		a.logger.Info("Authorization decision",
			"user", review.Spec.User,
			"verb", verb,
			"resource", attrs.Resource,
			"namespace", attrs.Namespace,
			"ownedBy", ownedBy,
			"supportGroups", supportGroups,
			"allowed", true,
			"reason", "authorized to "+verb+" secret owned by "+ownedBy)
		recordAllowed(verb, attrs.Resource, ownedBy)
		a.respond(w, review, true, "authorized to "+verb+" secret owned by "+ownedBy)

	default:
		a.logger.Info("Authorization decision",
			"user", review.Spec.User,
			"verb", verb,
			"resource", attrs.Resource,
			"namespace", attrs.Namespace,
			"allowed", false,
			"reason", "verb not permitted on secrets")
		recordDenied(verb, attrs.Resource, reasonSupportGroupMismatch, nil)
		a.respond(w, review, false, "verb not permitted on secrets")
	}
}

func hasHelmOwnerSelector(sel *authv1.LabelSelectorAttributes) bool {
	if sel == nil {
		return false
	}
	for _, req := range sel.Requirements {
		if req.Key == "owner" && req.Operator == metav1.LabelSelectorOpIn &&
			len(req.Values) == 1 && req.Values[0] == "helm" {
			return true
		}
	}
	return false
}

func (a *authorizer) secretOwnedBy(ctx context.Context, namespace, name string) (ownedBy, reasonCode string, err error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: namespace, Name: name}
	if err := a.client.Get(ctx, key, secret); err != nil {
		authzKubeFetchErrorsTotal.WithLabelValues(err.Error()).Inc()
		return "", reasonObjectNotFound, err
	}
	ownedBy, ok := secret.Labels[greenhouseapis.LabelKeyOwnedBy]
	if !ok {
		return "", reasonNoOwnedByLabel, errors.New("secret has no owned-by label")
	}
	return ownedBy, "", nil
}

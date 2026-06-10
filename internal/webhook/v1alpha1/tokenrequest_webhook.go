// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

const (
	// maxTokenExpirationSeconds is 90 days in seconds.
	maxTokenExpirationSeconds = int64(90 * 24 * 60 * 60)
)

// SetupTokenRequestWebhookWithManager registers the mutating webhook for TokenRequest.
func SetupTokenRequestWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&authenticationv1.TokenRequest{},
		webhook.WebhookFuncs[*authenticationv1.TokenRequest]{
			DefaultFunc: defaultTokenRequest,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-authentication-k8s-io-v1-tokenrequest,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=serviceaccounts/token,verbs=create,versions=v1,name=mtokenrequest.kb.io,admissionReviewVersions=v1

func defaultTokenRequest(ctx context.Context, c client.Client, tokenRequest *authenticationv1.TokenRequest) error {
	// Do not intercept pod-bound token requests (projected volumes, /var/run/secrets).
	if tokenRequest.Spec.BoundObjectRef != nil {
		return nil
	}

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	// Only cap tokens for greenhouse team SAs (identified by the owned-by label).
	sa := &corev1.ServiceAccount{}
	if err := c.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, sa); err != nil {
		return apierrors.NewInternalError(err)
	}
	if sa.Labels[greenhouseapis.LabelKeyOwnedBy] == "" {
		return nil
	}

	if tokenRequest.Spec.ExpirationSeconds == nil || *tokenRequest.Spec.ExpirationSeconds > maxTokenExpirationSeconds {
		ctrl.LoggerFrom(ctx).Info("requested expiration shortened", "from", tokenRequest.Spec.ExpirationSeconds, "to", maxTokenExpirationSeconds)
		tokenRequest.Spec.ExpirationSeconds = new(int64)
		*tokenRequest.Spec.ExpirationSeconds = maxTokenExpirationSeconds
	}

	return nil
}

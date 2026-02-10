// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// Webhook for the Team custom resource.

func SetupTeamWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha1.Team{},
		webhook.WebhookFuncs[*greenhousev1alpha1.Team]{
			DefaultFunc:        DefaultTeam,
			ValidateCreateFunc: ValidateCreateTeam,
			ValidateUpdateFunc: ValidateUpdateTeam,
			ValidateDeleteFunc: ValidateDeleteTeam,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-team,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teams,verbs=create;update,versions=v1alpha1,name=mteam.kb.io,admissionReviewVersions=v1

func DefaultTeam(_ context.Context, _ client.Client, _ *greenhousev1alpha1.Team) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-team,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teams,verbs=create;update;delete,versions=v1alpha1,name=vteam.kb.io,admissionReviewVersions=v1

func ValidateCreateTeam(ctx context.Context, c client.Client, team *greenhousev1alpha1.Team) (admission.Warnings, error) {
	if err := validateGreenhouseLabels(team, ctx, c); err != nil {
		return nil, err
	}
	return nil, validateJoinURL(team)
}

func ValidateUpdateTeam(ctx context.Context, c client.Client, _, team *greenhousev1alpha1.Team) (admission.Warnings, error) {
	if err := validateGreenhouseLabels(team, ctx, c); err != nil {
		return nil, err
	}
	return nil, validateJoinURL(team)
}

func ValidateDeleteTeam(_ context.Context, _ client.Client, _ *greenhousev1alpha1.Team) (admission.Warnings, error) {
	return nil, nil
}

func validateGreenhouseLabels(team *greenhousev1alpha1.Team, ctx context.Context, c client.Client) error {
	labelsWhiteList := map[string]struct{}{
		"support-group": {},
	}

	pluginDefinitions := greenhousev1alpha1.ClusterPluginDefinitionList{}
	if err := c.List(ctx, &pluginDefinitions); !apierrors.IsNotFound(err) && err != nil {
		return err
	}
	for _, pluginDefinition := range pluginDefinitions.Items {
		labelsWhiteList[pluginDefinition.GetName()] = struct{}{}
	}

	labels := team.GetLabels()
	for labelKey := range labels {
		if strings.HasPrefix(labelKey, greenhouseapis.GroupName) {
			labelSuffix := strings.TrimPrefix(labelKey, greenhouseapis.GroupName+"/")
			_, ok := labelsWhiteList[labelSuffix]

			if !ok {
				return apierrors.NewInvalid(team.GroupVersionKind().GroupKind(), team.GetName(), field.ErrorList{
					field.Forbidden(field.NewPath("metadata").Child("labels").Child(labelKey),
						"Only pluginDefinition names as greenhouse labels allowed."),
				})
			}
		}
	}
	return nil
}

func validateJoinURL(team *greenhousev1alpha1.Team) error {
	if team.Spec.JoinURL == "" {
		return nil
	}
	if !webhook.ValidateURL(team.Spec.JoinURL) {
		return apierrors.NewInvalid(team.GroupVersionKind().GroupKind(), team.GetName(), field.ErrorList{
			field.Invalid(field.NewPath("spec").Child("joinUrl"), team.Spec.JoinURL,
				"JoinURL must be a valid 'http:' or 'https:' URL, like 'https://example.com'."),
		})
	}
	return nil
}

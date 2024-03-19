// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

var (
	labelsWhiteList = map[string]struct{}{
		"support-group": {},
	}
)

// Webhook for the Team custom resource.

func SetupTeamWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&greenhousev1alpha1.Team{},
		webhookFuncs{
			defaultFunc:        DefaultTeam,
			validateCreateFunc: ValidateCreateTeam,
			validateUpdateFunc: ValidateUpdateTeam,
			validateDeleteFunc: ValidateDeleteTeam,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-team,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teams,verbs=create;update,versions=v1alpha1,name=mteam.kb.io,admissionReviewVersions=v1

func DefaultTeam(_ context.Context, _ client.Client, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-team,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teams,verbs=create;update,versions=v1alpha1,name=vteam.kb.io,admissionReviewVersions=v1

func ValidateCreateTeam(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	team, ok := o.(*greenhousev1alpha1.Team)
	if !ok {
		return nil, nil
	}
	return nil, validateGreenhouseLabels(team, ctx, c)
}

func ValidateUpdateTeam(ctx context.Context, c client.Client, _, o runtime.Object) (admission.Warnings, error) {
	team, ok := o.(*greenhousev1alpha1.Team)
	if !ok {
		return nil, nil
	}
	return nil, validateGreenhouseLabels(team, ctx, c)
}

func ValidateDeleteTeam(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateGreenhouseLabels(team *greenhousev1alpha1.Team, ctx context.Context, c client.Client) error {
	plugins := greenhousev1alpha1.PluginList{}
	if err := c.List(ctx, &plugins); !apierrors.IsNotFound(err) && err != nil {
		return err
	}
	for _, plugin := range plugins.Items {
		labelsWhiteList[plugin.GetName()] = struct{}{}
	}

	labels := team.GetLabels()
	for labelKey := range labels {
		if strings.HasPrefix(labelKey, greenhouseapis.GroupName) {
			labelSuffix := strings.TrimPrefix(labelKey, greenhouseapis.GroupName+"/")
			_, ok := labelsWhiteList[labelSuffix]

			if !ok {
				return apierrors.NewInvalid(team.GroupVersionKind().GroupKind(), team.GetName(), field.ErrorList{
					field.Forbidden(field.NewPath("metadata").Child("labels").Child(labelKey),
						"Only plugin names as greenhouse labels allowed."),
				})
			}
		}
	}
	return nil
}

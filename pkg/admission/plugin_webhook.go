// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package admission

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// Webhook for the Plugin custom resource.

func SetupPluginWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&greenhousev1alpha1.Plugin{},
		webhookFuncs{
			defaultFunc:        DefaultPlugin,
			validateCreateFunc: ValidateCreatePlugin,
			validateUpdateFunc: ValidateUpdatePlugin,
			validateDeleteFunc: ValidateDeletePlugin,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-plugin,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=plugins,verbs=create;update,versions=v1alpha1,name=mplugin.kb.io,admissionReviewVersions=v1

func DefaultPlugin(_ context.Context, _ client.Client, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-plugin,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=plugins,verbs=create;update,versions=v1alpha1,name=vplugin.kb.io,admissionReviewVersions=v1

func ValidateCreatePlugin(_ context.Context, _ client.Client, o runtime.Object) (admission.Warnings, error) {
	plugin, ok := o.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}
	if err := validatePluginMustSpecifyHelmChartOrUIApplication(plugin); err != nil {
		return nil, err
	}
	return nil, validatePluginOptionValueAndType(plugin)
}

func ValidateUpdatePlugin(_ context.Context, _ client.Client, _, o runtime.Object) (admission.Warnings, error) {
	plugin, ok := o.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}
	if err := validatePluginMustSpecifyHelmChartOrUIApplication(plugin); err != nil {
		return nil, err
	}
	return nil, validatePluginOptionValueAndType(plugin)
}

func ValidateDeletePlugin(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validatePluginMustSpecifyHelmChartOrUIApplication(plugin *greenhousev1alpha1.Plugin) error {
	if plugin.Spec.HelmChart == nil && plugin.Spec.UIApplication == nil {
		return apierrors.NewInvalid(plugin.GroupVersionKind().GroupKind(), plugin.GetName(), field.ErrorList{
			field.Required(field.NewPath("spec").Child("helmChart", "uiApplication"),
				"A Plugin without both spec.helmChart and spec.uiApplication is invalid."),
		})
	}
	return nil
}

// validatePluginOptionValueAndType validates that the type and value of each PluginOption matches.
func validatePluginOptionValueAndType(plugin *greenhousev1alpha1.Plugin) error {
	for _, option := range plugin.Spec.Options {
		if err := option.IsValid(); err != nil {
			return apierrors.NewInvalid(plugin.GroupVersionKind().GroupKind(), plugin.GetName(), field.ErrorList{
				field.Invalid(field.NewPath("spec").Child("options").Child("name"), option.Name,
					"A PluginOption Default must match the specified Type."),
			})
		}
	}
	return nil
}

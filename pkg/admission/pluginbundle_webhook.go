// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// Webhook for the PluginBundle custom resource.

func SetupPluginBundleWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&greenhousev1alpha1.PluginBundle{},
		webhookFuncs{
			defaultFunc:        DefaultPluginBundle,
			validateCreateFunc: ValidateCreatePluginBundle,
			validateUpdateFunc: ValidateUpdatePluginBundle,
			validateDeleteFunc: ValidateDeletePluginBundle,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-pluginbundle,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginbundles,verbs=create;update,versions=v1alpha1,name=mpluginbundle.kb.io,admissionReviewVersions=v1

func DefaultPluginBundle(_ context.Context, _ client.Client, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-pluginbundle,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginbundles,verbs=create;update,versions=v1alpha1,name=vpluginbundle.kb.io,admissionReviewVersions=v1

func ValidateCreatePluginBundle(_ context.Context, _ client.Client, o runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func ValidateUpdatePluginBundle(_ context.Context, _ client.Client, oldObj, curObj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func ValidateDeletePluginBundle(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

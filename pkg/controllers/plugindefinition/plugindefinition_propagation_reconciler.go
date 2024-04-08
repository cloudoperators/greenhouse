// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/controllers"
)

//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch

type PluginDefinitionPropagationReconciler struct {
	controllers.PropagationReconciler
}

func (r *PluginDefinitionPropagationReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.EmptyObj = &greenhousev1alpha1.PluginDefinition{}
	r.EmptyObjList = &greenhousev1alpha1.PluginDefinitionList{}
	r.CRDName = "plugindefinitions.greenhouse.sap"
	r.StripObjectWrapper = r.StripObject
	r.HandlerFunc = r.ListObjectsAsReconcileRequests

	return r.BaseSetupWithManager(name, mgr)
}

func (r *PluginDefinitionPropagationReconciler) ListObjectsAsReconcileRequests(ctx context.Context, _ client.Object) []ctrl.Request {
	res := []ctrl.Request{}

	objList, ok := r.ListObjects(ctx).(*greenhousev1alpha1.PluginDefinitionList)
	if !ok {
		log.FromContext(ctx).Error(fmt.Errorf("object %T is not a greenhousev1alpha1.PluginDefinitionList", objList), "failed to list objects")
		return res
	}

	for _, pluginDefinition := range objList.Items {
		res = append(res, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(pluginDefinition.DeepCopy())})
	}

	return res
}

func (r *PluginDefinitionPropagationReconciler) StripObject(in client.Object) (client.Object, error) {
	obj, ok := in.(*greenhousev1alpha1.PluginDefinition)
	if !ok {
		return nil, fmt.Errorf("error: %T is not a pluginDefinition", in)
	}

	typeMeta := metav1.TypeMeta{
		Kind:       in.GetObjectKind().GroupVersionKind().Kind,
		APIVersion: in.GetObjectKind().GroupVersionKind().GroupVersion().String(),
	}
	objectMeta := metav1.ObjectMeta{
		Name:        in.GetName(),
		Namespace:   in.GetNamespace(),
		Labels:      in.GetLabels(),
		Annotations: in.GetAnnotations(),
	}

	return &greenhousev1alpha1.PluginDefinition{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec:       obj.Spec,
	}, nil
}

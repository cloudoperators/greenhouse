// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

// PropagationReconciler implements the basic functionality every resource propagation reconciler needs.
type PropagationReconciler struct {
	client.Client
	Recorder           record.EventRecorder
	EmptyObj           client.Object
	EmptyObjList       client.ObjectList
	CRDName            string
	HandlerFunc        func(context.Context, client.Object) []ctrl.Request
	StripObjectWrapper func(client.Object) (client.Object, error)
}

func (r *PropagationReconciler) BaseSetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(r.EmptyObj).
		// Watch the respective CRD and enqueue all objects.
		Watches(&apiextensionsv1.CustomResourceDefinition{},
			handler.EnqueueRequestsFromMapFunc(r.HandlerFunc),
			builder.WithPredicates(
				clientutil.PredicateByName(r.CRDName),
				clientutil.PredicateHasFinalizer(greenhouseapis.FinalizerCleanupPropagatedResource)),
		).
		Complete(r)
}

func (r *PropagationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj, ok := r.EmptyObj.DeepCopyObject().(client.Object)
	if !ok {
		return ctrl.Result{}, fmt.Errorf("object %T is not a client.Object", obj)
	}

	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !controllerutil.ContainsFinalizer(obj, greenhouseapis.FinalizerCleanupPropagatedResource) {
		fmt.Printf("Skip resource because it does not contain the cleanup finalizer")
		return ctrl.Result{}, nil
	}

	clusterList := new(greenhousev1alpha1.ClusterList)
	if err := r.List(ctx, clusterList, &client.ListOptions{Namespace: obj.GetNamespace()}); err != nil {
		return ctrl.Result{}, err
	}

	// TODO parallelize
	for _, cluster := range clusterList.Items {
		// skip clusters that are no longer accessible
		if !cluster.Status.StatusConditions.IsReadyTrue() {
			continue
		}

		// get corresponding secret to access the cluster
		var secret = new(corev1.Secret)
		if err := r.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, secret); err != nil {
			return ctrl.Result{}, err
		}

		restClientGetter, err := clientutil.NewRestClientGetterFromSecret(secret, req.Namespace, clientutil.WithPersistentConfig())
		if err != nil {
			return ctrl.Result{}, err
		}

		remoteRestClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
		if err != nil {
			return ctrl.Result{}, err
		}

		if err = r.reconcileObject(ctx, remoteRestClient, obj, cluster.GetName()); err != nil {
			return ctrl.Result{}, err
		}
	}

	_, err := clientutil.Patch(ctx, r.Client, obj, func() error {
		controllerutil.RemoveFinalizer(obj, greenhouseapis.FinalizerCleanupPropagatedResource)
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PropagationReconciler) reconcileObject(ctx context.Context, restClient client.Client, obj client.Object, clusterName string) error {
	remoteObject := obj.DeepCopyObject().(client.Object) //nolint:errcheck
	remoteObjectExists := true
	if err := restClient.Get(ctx, client.ObjectKeyFromObject(remoteObject), remoteObject); err != nil {
		if apierrors.IsNotFound(err) {
			remoteObjectExists = false
		} else {
			return err
		}
	}

	// cleanup
	if remoteObjectExists {
		if err := restClient.Delete(ctx, remoteObject); err != nil {
			// might have been deleted by now
			if apierrors.IsNotFound(err) {
				log.FromContext(ctx).Info("object does not exist on target cluster", "object", obj, "cluster", clusterName)
				return nil
			} else {
				return err
			}
		}
		log.FromContext(ctx).Info("deleted object on target cluster", "object", obj, "cluster", clusterName)
	}
	return nil
}

func (r *PropagationReconciler) ListObjects(ctx context.Context) client.ObjectList {
	objList, ok := r.EmptyObjList.DeepCopyObject().(client.ObjectList)
	if !ok {
		log.FromContext(ctx).Error(fmt.Errorf("object %T is not a client.ObjectList", objList), "failed to list objects")
		return r.EmptyObjList
	}
	if err := r.List(ctx, objList, &client.ListOptions{Namespace: ""}); err != nil {
		log.FromContext(ctx).Error(err, "failed to list objects")
		return r.EmptyObjList
	}

	return objList
}

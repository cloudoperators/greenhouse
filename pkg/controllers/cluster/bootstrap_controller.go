// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type BootstrapReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *BootstrapReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&corev1.Secret{}, builder.WithPredicates(
			clientutil.PredicateFilterBySecretType(greenhouseapis.SecretTypeKubeConfig),
		)).
		// Watch clusters and enqueue its secret.
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(enqueueSecretForCluster)).
		Complete(r)
}

func (r *BootstrapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var kubeConfigSecret = new(corev1.Secret)
	if err := r.Get(ctx, req.NamespacedName, kubeConfigSecret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.reconcileCluster(ctx, kubeConfigSecret); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.ensureOwnerReferenceOnSecret(ctx, kubeConfigSecret); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
}

func (r *BootstrapReconciler) reconcileCluster(ctx context.Context, kubeConfigSecret *corev1.Secret) error {
	cluster, isFound, err := r.getClusterAndIgnoreNotFoundError(ctx, kubeConfigSecret)
	// Anything other than an IsNotFound error is reflected in the status to ensure the cluster resource is created in any case.
	if err != nil {
		return r.createOrPatchCluster(ctx, cluster, kubeConfigSecret, err)
	}

	// This cluster has already been bootstrapped
	// How does a customer provide a new KubeConfig ?
	// TODO: The below is a short-term fix to avoid flapping accessModes and should be considered again.
	// A new/updated KubeConfig should be handled and we shouldn't break here though
	// avoiding flapping of the accessMode, e.g. due to apiserver downtime, network interruption, etc.
	if isFound && cluster.Spec.AccessMode != "" {
		return nil
	}

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(kubeConfigSecret, kubeConfigSecret.Namespace, clientutil.WithPersistentConfig())
	if err != nil {
		return r.createOrPatchCluster(ctx, cluster, kubeConfigSecret, errors.Wrap(err, "failed creating restClientGetter"))
	}

	if _, err := clientutil.GetKubernetesVersion(restClientGetter); err != nil {
		return r.createOrPatchCluster(ctx, cluster, kubeConfigSecret, errors.Wrap(err, "failed fetching kubernetes Version from cluster"))
	}

	return r.createOrPatchCluster(ctx, cluster, kubeConfigSecret, nil)
}

// ensureOwnerReferenceOnSecret adds the ownerReference to the secret containing the kubeconfig, so that it is garbage collected on cluster deletion.
func (r *BootstrapReconciler) ensureOwnerReferenceOnSecret(ctx context.Context, kubeConfigSecret *corev1.Secret) error {
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := r.Get(ctx, types.NamespacedName{Namespace: kubeConfigSecret.GetNamespace(), Name: kubeConfigSecret.GetName()}, cluster); err != nil {
		return err
	}
	_, err := clientutil.CreateOrPatch(ctx, r.Client, kubeConfigSecret, func() error {
		return controllerutil.SetOwnerReference(cluster, kubeConfigSecret, r.Scheme())
	})
	return err
}

func (r *BootstrapReconciler) getClusterAndIgnoreNotFoundError(ctx context.Context, kubeConfigSecret *corev1.Secret) (cluster *greenhousev1alpha1.Cluster, isFound bool, err error) {
	cluster = new(greenhousev1alpha1.Cluster)
	err = r.Get(ctx, client.ObjectKeyFromObject(kubeConfigSecret), cluster)
	return cluster, !apierrors.IsNotFound(err), client.IgnoreNotFound(err)
}

// createOrPatchCluster creates or patches the cluster resource and persists input err in the cluster.status.message.
func (r *BootstrapReconciler) createOrPatchCluster(
	ctx context.Context,
	cluster *greenhousev1alpha1.Cluster,
	kubeConfigSecret *corev1.Secret,
	err error,
) error {
	// Ignore clusters about to be deleted.
	if cluster.DeletionTimestamp != nil {
		return nil
	}

	// if createOrPatch was called without an error the previous clientutil.GetKubernetesVersion() call was successful
	readyCondition := greenhousev1alpha1.TrueCondition(greenhousev1alpha1.ReadyCondition, "", "")

	accessMode := greenhousev1alpha1.ClusterAccessModeDirect

	if err != nil {
		readyCondition.Message = "cluster not ready: " + err.Error()
		readyCondition.Status = metav1.ConditionFalse
	}

	cluster.Name = kubeConfigSecret.Name
	cluster.Namespace = kubeConfigSecret.Namespace
	result, err := clientutil.CreateOrPatch(ctx, r.Client, cluster, func() error {
		cluster.Spec.AccessMode = accessMode
		return nil
	})
	if err != nil {
		return err
	}
	if result != clientutil.OperationResultNone {
		logMessage := fmt.Sprintf("%s cluster", result)
		log.FromContext(ctx).Info(logMessage, "namespace", cluster.Namespace, "name", cluster.Name)
	}

	// patch message and condition
	result, err = clientutil.PatchStatus(ctx, r.Client, cluster, func() error {
		cluster.Status.SetConditions(readyCondition)
		return nil
	})
	if err != nil {
		return err
	}
	if result != clientutil.OperationResultNone {
		logMessage := fmt.Sprintf("%s cluster.status", result)
		log.FromContext(ctx).Info(logMessage, "namespace", cluster.Namespace, "name", cluster.Name, "status", cluster.Status)
	}
	return nil
}

func enqueueSecretForCluster(_ context.Context, o client.Object) []ctrl.Request {
	cluster, ok := o.(*greenhousev1alpha1.Cluster)
	if !ok {
		return nil
	}
	// Ignore clusters being deleted currently.
	if cluster.DeletionTimestamp != nil {
		return nil
	}
	return []ctrl.Request{{NamespacedName: types.NamespacedName{Namespace: cluster.GetNamespace(), Name: cluster.GetSecretName()}}}
}

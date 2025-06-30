// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pkg/errors"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
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
			clientutil.PredicateFilterBySecretTypes(greenhouseapis.SecretTypeKubeConfig, greenhouseapis.SecretTypeOIDCConfig),
		)).
		// Watch clusters and enqueue its secret.
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(enqueueSecretForCluster)).
		Complete(r)
}

// BootstrapController is not refactored to use the lifecycle package, because the Secret resource does not implement lifecycle.RuntimeObject.

func (r *BootstrapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var kubeConfigSecret = new(corev1.Secret)
	if err := r.Get(ctx, req.NamespacedName, kubeConfigSecret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if kubeConfigSecret.Type == greenhouseapis.SecretTypeOIDCConfig {
		// if secret type is oidc we check if a kubeconfig was already generated,
		// and we also check if the greenhousekubeconfig key is present and the value is not empty
		genTime, genTimeAvail := kubeConfigSecret.Annotations[greenhouseapis.SecretOIDCConfigGeneratedOnAnnotation]
		if !genTimeAvail || !clientutil.IsSecretContainsKey(kubeConfigSecret, greenhouseapis.GreenHouseKubeConfigKey) {
			sa := utils.NewServiceAccount(kubeConfigSecret.GetName(), kubeConfigSecret.GetNamespace())
			_, err := clientutil.CreateOrPatch(ctx, r.Client, sa, func() error {
				return controllerutil.SetOwnerReference(kubeConfigSecret, sa, r.Scheme())
			})
			if err != nil {
				return ctrl.Result{}, errors.Wrap(err, "failed creating service account for OIDC config")
			}
			log.FromContext(ctx).Info("OIDC config generated", "date", genTime, "namespace", kubeConfigSecret.GetNamespace(), "name", kubeConfigSecret.GetName())
			return ctrl.Result{}, r.createKubeConfigKey(ctx, kubeConfigSecret)
		}
	}

	if err := r.reconcileCluster(ctx, kubeConfigSecret); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.ensureOwnerReferences(ctx, kubeConfigSecret); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: utils.DefaultRequeueInterval}, nil
}

func (r *BootstrapReconciler) createKubeConfigKey(ctx context.Context, secret *corev1.Secret) error {
	// get the api-server-url from annotation
	// get the certificate from the secret
	annotations := secret.GetAnnotations()
	remoteAPIServerURL := annotations[greenhouseapis.SecretAPIServerURLAnnotation]
	certData := secret.Data[greenhouseapis.SecretAPIServerCAKey]
	certDecoded, err := base64.StdEncoding.DecodeString(string(certData))
	if err != nil {
		return errors.Wrap(err, "failed decoding certificate data")
	}

	// create token request from SA with audience
	clusterResourceSA := utils.NewServiceAccount(secret.GetName(), secret.GetNamespace())
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{greenhouseapis.OIDCAudience},
			ExpirationSeconds: ptr.To[int64](600),
		},
	}
	if err := r.Client.SubResource("token").Create(ctx, clusterResourceSA, tokenRequest); err != nil {
		return errors.Wrap(err, "failed creating token request for OIDC config")
	}

	// generate kubeconfig with oidc token
	generator := &utils.KubeConfigHelper{
		Host:        remoteAPIServerURL,
		CAData:      certDecoded,
		BearerToken: tokenRequest.Status.Token,
		Username:    fmt.Sprintf("system:serviceaccount:%s:%s", clusterResourceSA.GetNamespace(), clusterResourceSA.GetName()),
		Namespace:   clusterResourceSA.GetNamespace(),
	}
	kubeconfigByte, err := clientcmd.Write(generator.RestConfigToAPIConfig(secret.GetName()))
	if err != nil {
		return errors.Wrapf(err, "failed to generate kubeconfig for cluster %s", secret.GetName())
	}
	// update secret with kubeconfig directly on greenhousekubeconfig key and update oidc generated on annotation
	secret.Data[greenhouseapis.GreenHouseKubeConfigKey] = kubeconfigByte
	annotations[greenhouseapis.SecretOIDCConfigGeneratedOnAnnotation] = metav1.Now().Format(time.DateTime)
	secret.Annotations = annotations
	return r.Update(ctx, secret)
}

func (r *BootstrapReconciler) reconcileCluster(ctx context.Context, kubeConfigSecret *corev1.Secret) error {
	cluster, isFound, err := r.getClusterAndIgnoreNotFoundError(ctx, kubeConfigSecret)
	// Anything other than an IsNotFound error is reflected in the status to ensure the cluster resource is created in any case.
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to get cluster", "namespace", kubeConfigSecret.GetNamespace(), "name", kubeConfigSecret.GetName())
		return err
	}

	// This cluster has already been bootstrapped
	// How does a customer provide a new KubeConfig ?
	// TODO: The below is a short-term fix to avoid flapping accessModes and should be considered again.
	// A new/updated KubeConfig should be handled and we shouldn't break here though
	// avoiding flapping of the accessMode, e.g. due to apiserver downtime, network interruption, etc.
	if isFound && cluster.Spec.AccessMode != "" {
		return nil
	}
	return r.createOrUpdateCluster(ctx, cluster, kubeConfigSecret)
}

// createOrUpdateCluster creates or updates the cluster resource and persists input err in the cluster.status.message.
func (r *BootstrapReconciler) createOrUpdateCluster(
	ctx context.Context,
	cluster *greenhousev1alpha1.Cluster,
	kubeConfigSecret *corev1.Secret,
) error {
	// Ignore clusters about to be deleted.
	if !cluster.DeletionTimestamp.IsZero() {
		return nil
	}
	accessMode := greenhousev1alpha1.ClusterAccessModeDirect

	cluster.SetName(kubeConfigSecret.Name)
	cluster.SetNamespace(kubeConfigSecret.Namespace)

	// Propagate the owned-by label value from kubeConfigSecret for new Clusters.
	if _, ok := cluster.Labels[greenhouseapis.LabelKeyOwnedBy]; !ok {
		if cluster.Labels == nil {
			cluster.Labels = make(map[string]string)
		}
		if supportGroupTeam, ok := kubeConfigSecret.Labels[greenhouseapis.LabelKeyOwnedBy]; ok {
			cluster.Labels[greenhouseapis.LabelKeyOwnedBy] = supportGroupTeam
		} else {
			return fmt.Errorf("failed to propagate %s label from Secret %s", greenhouseapis.LabelKeyOwnedBy, kubeConfigSecret.Name)
		}
	}

	annotations := make(map[string]string)
	if cluster.GetAnnotations() != nil {
		annotations = cluster.GetAnnotations()
	}
	if kubeConfigSecret.Type == greenhouseapis.SecretTypeKubeConfig {
		annotations[greenhouseapis.ClusterConnectivityAnnotation] = greenhouseapis.ClusterConnectivityKubeconfig
	}
	if kubeConfigSecret.Type == greenhouseapis.SecretTypeOIDCConfig {
		annotations[greenhouseapis.ClusterConnectivityAnnotation] = greenhouseapis.ClusterConnectivityOIDC
	}
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, cluster, func() error {
		cluster.SetAnnotations(annotations)
		cluster.Spec.AccessMode = accessMode
		// Transport KubeConfigSecret labels to Cluster
		cluster = (lifecycle.NewPropagator(kubeConfigSecret, cluster).ApplyLabels()).(*greenhousev1alpha1.Cluster) //nolint:errcheck
		return nil
	})
	if err != nil {
		return err
	}
	if result != controllerutil.OperationResultNone {
		logMessage := fmt.Sprintf("%s cluster", result)
		log.FromContext(ctx).Info(logMessage, "namespace", cluster.Namespace, "name", cluster.Name)
	}
	return nil
}

// ensureOwnerReferences adds the ownerReference to the secret containing the kubeconfig, so that it is garbage collected on cluster deletion.
func (r *BootstrapReconciler) ensureOwnerReferences(ctx context.Context, kubeConfigSecret *corev1.Secret) error {
	cluster := &greenhousev1alpha1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: kubeConfigSecret.GetNamespace(), Name: kubeConfigSecret.GetName()}, cluster); err != nil {
		return err
	}
	if cluster.DeletionTimestamp != nil {
		return nil
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

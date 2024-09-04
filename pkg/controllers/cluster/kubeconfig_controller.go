// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"

	"github.com/rxwycdh/rxhash"
)

type KubeconfigReconciler struct {
	client.Client
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=cluster-kubeconfigs,verbs=get;list;watch;create;update;patch

// SetupWithManager sets up the controller with the Manager
func (r *KubeconfigReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		Watches(&v1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(enqueueSameNameResource)).
		For(&v1alpha1.ClusterKubeconfig{}).
		Complete(r)
}

func (r *KubeconfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("cluster", req.Name, "namespace", req.Namespace)

	var cluster v1alpha1.Cluster
	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		l.Error(err, "unable to fetch cluster")
		return ctrl.Result{}, err
	}

	if cluster.GetDeletionTimestamp() != nil {
		l.Info("skip reconcile, cluster is being deleted")
		return ctrl.Result{}, nil
	}

	var kubeconfig v1alpha1.ClusterKubeconfig
	updateRequired := false
	failed := false

	if err := r.Get(ctx, req.NamespacedName, &kubeconfig); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}

		l.Info("kubeconfig not found, will be created")
		kubeconfig.Name = cluster.Name
		kubeconfig.Namespace = cluster.Namespace
		kubeconfig.Spec.Kubeconfig = v1alpha1.ClusterKubeconfigData{
			Kind:       "Config",
			APIVersion: "v1",
		}
		kubeconfig.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: cluster.APIVersion,
				Kind:       cluster.Kind,
				Name:       cluster.Name,
				UID:        cluster.UID,
			},
		}

		kubeconfig.Status.Conditions.SetConditions(v1alpha1.TrueCondition(v1alpha1.KubeconfigCreatedCondition, "NewCreation", ""))

		err := r.Client.Create(ctx, &kubeconfig)
		if err != nil {
			l.Error(err, "unable to create kubeconfig")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if kubeconfig.GetDeletionTimestamp() != nil {
		l.Info("skip reconcile, kubeconfig is being deleted")
		return ctrl.Result{}, nil
	}

	condition := kubeconfig.Status.Conditions.GetConditionByType(v1alpha1.KubeconfigReconcileFailedCondition)
	if condition != nil && condition.IsTrue() {
		l.Info("skip reconcile, reconcile failed already")
		return ctrl.Result{}, nil
	}

	// get cluster connection data from cluster secret
	var secret corev1.Secret
	err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &secret)
	if err != nil {
		kubeconfig.Status.Conditions.SetConditions(v1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "SecretFetch", err.Error()))
		updateRequired = true
		failed = true
		l.Error(err, "unable to fetch secret")
	} else if secret.ObjectMeta.ResourceVersion != kubeconfig.ObjectMeta.Annotations[ClusterSecretResourceVersionAnnotation] { // check if cluster secret has been updated
		updateRequired = true
		if kubeconfig.ObjectMeta.Annotations == nil {
			kubeconfig.ObjectMeta.Annotations = make(map[string]string)
		}
		kubeconfig.ObjectMeta.Annotations[ClusterSecretResourceVersionAnnotation] = secret.ObjectMeta.ResourceVersion

		rootKubeCfg := secret.Data["kubeconfig"]
		kubeCfg, err := clientcmd.Load(rootKubeCfg)
		if err != nil {
			l.Error(err, "unable to load kubeconfig")
			return ctrl.Result{}, err
		}

		var clusterCfg *clientcmdapi.Cluster
		for _, v := range kubeCfg.Clusters {
			clusterCfg = v
			break
		}

		kubeconfig.Spec.Kubeconfig.Clusters = []v1alpha1.ClusterKubeconfigClusterItem{
			{
				Name: cluster.Name,
				Cluster: v1alpha1.ClusterKubeconfigCluster{
					Server:                   clusterCfg.Server,
					CertificateAuthorityData: clusterCfg.CertificateAuthorityData,
				},
			}}
	}

	// get oidc info from organization
	oidc, err := r.getOIDCInfo(ctx, cluster.Namespace)
	if err != nil {
		kubeconfig.Status.Conditions.SetConditions(v1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "OIDCFetch", err.Error()))
		updateRequired = true
		failed = true
		l.Error(err, "unable to fetch oidc data")
	} else {
		oidcHash, err := rxhash.HashStruct(oidc)
		if err != nil {
			l.Error(err, "unable to hash oidc info")
			return ctrl.Result{}, err
		}
		if kubeconfig.ObjectMeta.Annotations[OIDCHashAnnotation] != oidcHash {
			updateRequired = true
			if kubeconfig.ObjectMeta.Annotations == nil {
				kubeconfig.ObjectMeta.Annotations = make(map[string]string)
			}
			kubeconfig.ObjectMeta.Annotations[OIDCHashAnnotation] = oidcHash

			kubeconfig.Spec.Kubeconfig.AuthInfo = []v1alpha1.ClusterKubeconfigAuthInfoItem{
				{
					Name: "oidc@" + cluster.Name,
					AuthInfo: v1alpha1.ClusterKubeconfigAuthInfo{
						AuthProvider: clientcmdapi.AuthProviderConfig{
							Name: "oidc",
							Config: map[string]string{
								"client-id":      oidc.ClientID,
								"client-secret":  oidc.ClientSecret,
								"idp-issuer-url": oidc.IssuerURL,
							},
						},
					},
				},
			}
		}
	}

	if updateRequired {
		if !failed {
			kubeconfig.Status.Conditions.SetConditions(v1alpha1.TrueCondition(v1alpha1.KubeconfigReadyCondition, "Complete", ""))
		}
		// check for the context settings
		kubeconfig.Spec.Kubeconfig.Contexts = []v1alpha1.ClusterKubeconfigContextItem{
			{
				Name: cluster.Name,
				Context: v1alpha1.ClusterKubeconfigContext{
					Cluster:   cluster.Name,
					AuthInfo:  "oidc@" + cluster.Name,
					Namespace: "default",
				},
			}}
		kubeconfig.Spec.Kubeconfig.CurrentContext = cluster.Name

		err := r.Client.Update(ctx, &kubeconfig)
		if err != nil {
			l.Error(err, "unable to update kubeconfig")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

type OIDCInfo struct {
	ClientID     string
	ClientSecret string
	IssuerURL    string
}

func (r *KubeconfigReconciler) getOIDCInfo(ctx context.Context, orgName string) (OIDCInfo, error) {
	l := log.FromContext(ctx).WithValues("org", orgName)

	var org v1alpha1.Organization
	if err := r.Get(ctx, client.ObjectKey{Name: orgName}, &org); err != nil {
		l.Error(err, "unable to fetch organization", "organization", orgName)
		return OIDCInfo{}, err
	}

	clientIDRef := org.Spec.Authentication.OIDCConfig.ClientIDReference
	clientID, err := clientutil.GetSecretKeyFromSecretKeyReference(
		ctx,
		r.Client,
		orgName,
		clientIDRef,
	)
	if err != nil {
		l.Error(err, "unable to fetch client id", "organization", orgName)
		return OIDCInfo{}, err
	}

	clientSecretRef := org.Spec.Authentication.OIDCConfig.ClientSecretReference
	clientSecret, err := clientutil.GetSecretKeyFromSecretKeyReference(
		ctx,
		r.Client,
		orgName,
		clientSecretRef,
	)
	if err != nil {
		l.Error(err, "unable to fetch client secret", "organization", orgName)
		return OIDCInfo{}, err
	}
	oidc := OIDCInfo{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		IssuerURL:    org.Spec.Authentication.OIDCConfig.Issuer,
	}
	return oidc, nil
}

const (
	ClusterSecretResourceVersionAnnotation = "greenhouse.sap/cluster-secret-resource-version"
	OIDCHashAnnotation                     = "greenhouse.sap/oidc-hash"
)

func enqueueSameNameResource(_ context.Context, o client.Object) []ctrl.Request {
	return []ctrl.Request{{NamespacedName: types.NamespacedName{Namespace: o.GetNamespace(), Name: o.GetName()}}}
}

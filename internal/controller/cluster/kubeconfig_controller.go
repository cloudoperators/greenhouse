// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	"github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
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
		Watches(&v1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(sameNameResource),
			builder.WithPredicates(clientutil.PredicateClusterIsReady())).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(clusterSecretToCluster)).
		Watches(&v1alpha1.Organization{}, handler.EnqueueRequestsFromMapFunc(r.organizationToClusters)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.organizationSecretToClusters)).
		For(&v1alpha1.ClusterKubeconfig{}).
		Complete(r)
}

func (r *KubeconfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("cluster", req.Name, "namespace", req.Namespace)

	var cluster v1alpha1.Cluster
	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		l.Info("skip reconcile, cluster is not found")
		return ctrl.Result{}, nil
	}

	if cluster.GetDeletionTimestamp() != nil {
		l.Info("skip reconcile, cluster is being deleted")
		return ctrl.Result{}, nil
	}

	var kubeconfig v1alpha1.ClusterKubeconfig

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
	}

	defer func() {
		result, err := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
			kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
			return nil
		})
		if err != nil {
			log.FromContext(ctx).Error(err, "error setting status")
		}
		l.Info("status updated", "result", result)
	}()

	// get oidc info from organization
	oidc, err := r.getOIDCInfo(ctx, cluster.Namespace)
	if err != nil {
		kubeconfig.Status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "OIDCInfoError", err.Error()))
		return ctrl.Result{}, nil
	}

	// get cluster connection data from cluster secret
	var secret corev1.Secret
	err = r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &secret)
	if err != nil {
		kubeconfig.Status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "SecretDataError", err.Error()))
		return ctrl.Result{}, nil
	}

	// collect cluster connection data and update kubeconfig
	rootKubeCfg := secret.Data[greenhouseapis.GreenHouseKubeConfigKey]
	kubeCfg, err := clientcmd.Load(rootKubeCfg)
	if err != nil {
		kubeconfig.Status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "KubeconfigLoadError", err.Error()))
		return ctrl.Result{}, nil
	}

	var clusterCfg *clientcmdapi.Cluster
	for _, v := range kubeCfg.Clusters {
		clusterCfg = v
		break
	}

	// collect oidc data and update kubeconfig
	result, err := clientutil.CreateOrPatch(ctx, r.Client, &kubeconfig, func() error {
		// Mirror the cluster's labels
		kubeconfig.Labels = cluster.GetLabels()

		kubeconfig.Spec.Kubeconfig.Clusters = []v1alpha1.ClusterKubeconfigClusterItem{
			{
				Name: cluster.Name,
				Cluster: v1alpha1.ClusterKubeconfigCluster{
					Server:                   clusterCfg.Server,
					CertificateAuthorityData: clusterCfg.CertificateAuthorityData,
				},
			}}
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
		return nil
	})

	if err != nil {
		return ctrl.Result{}, err
	}
	l.Info("kubeconfig updated", "result", result)
	return ctrl.Result{}, nil
}

type OIDCInfo struct {
	ClientID     string
	ClientSecret string
	IssuerURL    string
}

func (r *KubeconfigReconciler) getOIDCInfo(ctx context.Context, orgName string) (OIDCInfo, error) {
	var org v1alpha1.Organization
	if err := r.Get(ctx, client.ObjectKey{Name: orgName}, &org); err != nil {
		return OIDCInfo{}, err
	}

	if org.Spec.Authentication == nil || org.Spec.Authentication.OIDCConfig == nil {
		return OIDCInfo{}, errors.New("no oidc config found")
	}

	clientIDRef := org.Spec.Authentication.OIDCConfig.ClientIDReference
	clientID, err := clientutil.GetSecretKeyFromSecretKeyReference(
		ctx,
		r.Client,
		orgName,
		clientIDRef,
	)
	if err != nil {
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
		return OIDCInfo{}, err
	}
	oidc := OIDCInfo{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		IssuerURL:    org.Spec.Authentication.OIDCConfig.Issuer,
	}
	return oidc, nil
}

func sameNameResource(_ context.Context, o client.Object) []ctrl.Request {
	return []ctrl.Request{{NamespacedName: types.NamespacedName{Namespace: o.GetNamespace(), Name: o.GetName()}}}
}

func clusterSecretToCluster(_ context.Context, o client.Object) []ctrl.Request {
	secret, ok := o.(*corev1.Secret)
	if ok && secret.Type == greenhouseapis.SecretTypeKubeConfig {
		return []ctrl.Request{{NamespacedName: types.NamespacedName{Namespace: o.GetNamespace(), Name: o.GetName()}}}
	}
	return nil
}

func (r *KubeconfigReconciler) organizationToClusters(ctx context.Context, o client.Object) []ctrl.Request {
	// get namespace for this org
	ns := &corev1.Namespace{}
	err := r.Get(ctx, client.ObjectKey{Name: o.GetName()}, ns)

	// if namespace exists
	if err == nil {
		// get clusters in this namespace
		clusters := &v1alpha1.ClusterList{}
		err = r.List(ctx, clusters, client.InNamespace(ns.GetName()))
		if err == nil {
			var requests []ctrl.Request
			for _, cluster := range clusters.Items {
				requests = append(requests, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}})
			}
			return requests
		}
	}
	return nil
}

func (r *KubeconfigReconciler) organizationSecretToClusters(ctx context.Context, o client.Object) []ctrl.Request {
	// get clusters in this namespace
	clusters := &v1alpha1.ClusterList{}
	err := r.List(ctx, clusters, client.InNamespace(o.GetNamespace()))
	if err == nil {
		var requests []ctrl.Request
		for _, cluster := range clusters.Items {
			requests = append(requests, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}})
		}
		return requests
	}
	return nil
}

func calculateKubeconfigStatus(ck *v1alpha1.ClusterKubeconfig) v1alpha1.ClusterKubeconfigStatus {
	// new creation
	status := ck.Status.DeepCopy()
	if len(status.Conditions.Conditions) == 0 {
		status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigCreatedCondition, "NewCreation", ""))
	}

	for _, ct := range ExposedKubeconfigConditions {
		if status.Conditions.GetConditionByType(ct) == nil {
			status.Conditions.SetConditions(greenhousemetav1alpha1.UnknownCondition(ct, "", ""))
		}
	}
	// check for failure
	reconcileFailedStatus := status.Conditions.GetConditionByType(v1alpha1.KubeconfigReconcileFailedCondition)
	if reconcileFailedStatus != nil && reconcileFailedStatus.IsTrue() {
		status.Conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(v1alpha1.KubeconfigReadyCondition, "ReconcileFailed", ""))
		status.Conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(v1alpha1.KubeconfigCreatedCondition, "ReconcileFailed", ""))
	} else {
		status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReadyCondition, "Complete", ""))
		status.Conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(v1alpha1.KubeconfigReconcileFailedCondition, "ReadyState", ""))
		status.Conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(v1alpha1.KubeconfigCreatedCondition, "ReadyState", ""))
	}
	return *status
}

var ExposedKubeconfigConditions = []greenhousemetav1alpha1.ConditionType{
	v1alpha1.KubeconfigCreatedCondition,
	v1alpha1.KubeconfigReconcileFailedCondition,
	v1alpha1.KubeconfigReadyCondition,
}

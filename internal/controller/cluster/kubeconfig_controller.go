// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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
	MaxConcurrentReconciles int
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
		Watches(&v1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(sameNameResource)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(clusterSecretToCluster)).
		Watches(&v1alpha1.Organization{}, handler.EnqueueRequestsFromMapFunc(r.organizationToClusters)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.organizationSecretToClusters)).
		For(&v1alpha1.ClusterKubeconfig{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
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

		// ensure the ClusterKubeconfig resource exists before any status patches happen later
		if err := r.Create(ctx, &kubeconfig); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				l.Error(err, "failed to create initial ClusterKubeconfig resource")
				return ctrl.Result{}, err
			}
		} else {
			l.Info("created initial ClusterKubeconfig resource")
		}
	}

	// Sync labels from cluster on every reconcile, independent of OIDC/secret availability.
	if _, err := clientutil.CreateOrPatch(ctx, r.Client, &kubeconfig, func() error {
		kubeconfig.Labels = cluster.GetLabels()
		// Always delete the version key first so a stale value from the Cluster's own
		// metadata labels cannot survive when Status.KubernetesVersion is empty.
		delete(kubeconfig.Labels, greenhouseapis.LabelKeyKubernetesVersion)
		if v := cluster.Status.KubernetesVersion; v != "" {
			if kubeconfig.Labels == nil {
				kubeconfig.Labels = make(map[string]string)
			}
			if safe, ok := labelSafeVersion(v); ok {
				kubeconfig.Labels[greenhouseapis.LabelKeyKubernetesVersion] = safe
			} else {
				l.Info("kubernetes version cannot be represented as a label value, skipping", "version", v)
			}
		}
		return nil
	}); err != nil {
		l.Error(err, "failed to sync labels on ClusterKubeconfig")
		return ctrl.Result{}, err
	}

	// get oidc info from organization
	oidc, err := r.getOIDCInfo(ctx, cluster.Namespace, &cluster)
	if err != nil {
		kubeconfig.Status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "OIDCInfoError", err.Error()))
		// patch status before returning on error
		result, perr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
			kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
			return nil
		})
		if perr != nil {
			log.FromContext(ctx).Error(perr, "error setting status")
		}
		l.Info("status updated", "result", result)
		return ctrl.Result{}, nil
	}

	// get cluster connection data from cluster secret
	var secret corev1.Secret
	err = r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &secret)
	if err != nil {
		kubeconfig.Status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "SecretDataError", err.Error()))
		// patch status before returning on error
		result, perr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
			kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
			return nil
		})
		if perr != nil {
			log.FromContext(ctx).Error(perr, "error setting status")
		}
		l.Info("status updated", "result", result)
		return ctrl.Result{}, nil
	}

	// collect cluster connection data and update kubeconfig
	var clusterCfg *clientcmdapi.Cluster
	if cluster.Annotations[greenhouseapis.ClusterWorkloadIdentityAnnotation] == greenhouseapis.ClusterWorkloadIdentityEnabled {
		caData, err := base64.StdEncoding.DecodeString(string(secret.Data[greenhouseapis.SecretAPIServerCAKey]))
		if err != nil {
			result, perr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
				kubeconfig.Status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "SecretDataError", err.Error()))
				kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
				return nil
			})
			if perr != nil {
				log.FromContext(ctx).Error(perr, "error setting status")
			}
			l.Info("status updated", "result", result)
			return ctrl.Result{}, nil
		}
		clusterCfg = &clientcmdapi.Cluster{
			Server:                   secret.Annotations[greenhouseapis.SecretAPIServerURLAnnotation],
			CertificateAuthorityData: caData,
		}
	} else {
		rootKubeCfg, hasKey := secret.Data[greenhouseapis.GreenHouseKubeConfigKey]
		if !hasKey || len(rootKubeCfg) == 0 {
			kubeconfig.Status.Conditions.SetConditions(
				greenhousemetav1alpha1.TrueCondition(
					v1alpha1.KubeconfigReconcileFailedCondition,
					"KubeconfigMissing",
					"secret data key greenhousekubeconfig missing or empty",
				),
			)
			// patch status before returning on error
			result, perr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
				kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
				return nil
			})
			if perr != nil {
				log.FromContext(ctx).Error(perr, "error setting status")
			}
			l.Info("status updated", "result", result)
			return ctrl.Result{}, nil
		}

		kubeCfg, err := clientcmd.Load(rootKubeCfg)
		if err != nil {
			kubeconfig.Status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "KubeconfigLoadError", err.Error()))
			// patch status before returning on error
			result, perr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
				kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
				return nil
			})
			if perr != nil {
				log.FromContext(ctx).Error(perr, "error setting status")
			}
			l.Info("status updated", "result", result)
			return ctrl.Result{}, nil
		}

		for _, v := range kubeCfg.Clusters {
			clusterCfg = v
			break
		}
	}
	if clusterCfg == nil {
		kubeconfig.Status.Conditions.SetConditions(
			greenhousemetav1alpha1.TrueCondition(
				v1alpha1.KubeconfigReconcileFailedCondition,
				"KubeconfigInvalid",
				"no clusters found in kubeconfig",
			),
		)
		// patch status before returning on error
		result, perr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
			kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
			return nil
		})
		if perr != nil {
			log.FromContext(ctx).Error(perr, "error setting status")
		}
		l.Info("status updated", "result", result)
		return ctrl.Result{}, nil
	}

	// Resolve spec-level overrides before writing so they can be applied atomically in the
	// single CreateOrPatch below — two separate patches would re-enqueue the resource on every
	// reconcile because the controller watches ClusterKubeconfig via For(...).
	var resolvedOverrides *resolvedSpecOverrides
	if kubeconfig.Spec.ConfigOverrides != nil {
		var oErr error
		resolvedOverrides, oErr = resolveSpecOverrides(ctx, r.Client, cluster.Namespace, kubeconfig.Spec.ConfigOverrides)
		if oErr != nil {
			kubeconfig.Status.Conditions.SetConditions(
				greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "OverrideResolveError", oErr.Error()),
			)
			_, perr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
				kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
				return nil
			})
			if perr != nil {
				log.FromContext(ctx).Error(perr, "error setting status")
			}
			return ctrl.Result{}, oErr
		}
	}

	// collect oidc data and update kubeconfig — overrides applied inside the same patch.
	// All controller-managed fields are reset on every reconcile so that removing the
	// configOverrides spec field fully restores the baseline values.
	var overrideErr error
	result, err := clientutil.CreateOrPatch(ctx, r.Client, &kubeconfig, func() error {
		kubeconfig.Spec.Kubeconfig.Kind = "Config"
		kubeconfig.Spec.Kubeconfig.APIVersion = "v1"
		kubeconfig.Spec.Kubeconfig.CurrentContext = cluster.Name
		kubeconfig.Spec.Kubeconfig.Contexts = []v1alpha1.ClusterKubeconfigContextItem{
			{
				Name: cluster.Name,
				Context: v1alpha1.ClusterKubeconfigContext{
					Cluster:   cluster.Name,
					AuthInfo:  "oidc@" + cluster.Name,
					Namespace: "default",
				},
			},
		}
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
		if resolvedOverrides != nil {
			if err := applyResolvedOverrides(&kubeconfig.Spec.Kubeconfig, resolvedOverrides); err != nil {
				overrideErr = err
				return err
			}
		}
		return nil
	})

	if overrideErr != nil {
		kubeconfig.Status.Conditions.SetConditions(
			greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "OverrideValidationError", overrideErr.Error()),
		)
		_, perr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
			kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
			return nil
		})
		if perr != nil {
			log.FromContext(ctx).Error(perr, "error setting status")
		}
		return ctrl.Result{}, overrideErr
	}

	if err != nil {
		// When overrides participated in the desired spec, classify the patch failure accordingly.
		reason := greenhousemetav1alpha1.ConditionReason("PatchError")
		if resolvedOverrides != nil {
			reason = "OverridePatchError"
		}
		kubeconfig.Status.Conditions.SetConditions(
			greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, reason, err.Error()),
		)
		// patch status before returning on error
		sResult, perr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
			kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
			return nil
		})
		if perr != nil {
			log.FromContext(ctx).Error(perr, "error setting status")
		}
		l.Info("status updated", "result", sResult)
		return ctrl.Result{}, err
	}
	if resolvedOverrides != nil {
		l.Info("kubeconfig spec overrides applied")
	}
	l.Info("kubeconfig updated", "result", result)

	// Success path: clear any previous failure condition and patch status.
	kubeconfig.Status.Conditions.SetConditions(
		greenhousemetav1alpha1.FalseCondition(v1alpha1.KubeconfigReconcileFailedCondition, "NoError", ""),
	)
	sResult, sErr := clientutil.PatchStatus(ctx, r.Client, &kubeconfig, func() error {
		kubeconfig.Status = calculateKubeconfigStatus(&kubeconfig)
		return nil
	})
	if sErr != nil {
		log.FromContext(ctx).Error(sErr, "error setting status")
	}
	l.Info("status updated", "result", sResult)
	return ctrl.Result{}, nil
}

type OIDCInfo struct {
	ClientID     string
	ClientSecret string
	IssuerURL    string
}

// getOIDCInfo fetches OIDC configuration from the Organization.
// If the Cluster carries a greenhouse.sap/oidc-override annotation, those secret references
// take precedence over the organization defaults for backward compatibility.
func (r *KubeconfigReconciler) getOIDCInfo(ctx context.Context, orgName string, cluster *v1alpha1.Cluster) (OIDCInfo, error) {
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

	// Cluster-level overrides via annotation greenhouse.sap/oidc-override (backward compat).
	// Prefer configOverrides in Spec for new clusters; this annotation path is kept so that
	// existing clusters using it are not silently broken on upgrade.
	if cluster != nil && cluster.Annotations != nil {
		const annKey = "greenhouse.sap/oidc-override"
		if raw, ok := cluster.Annotations[annKey]; ok && raw != "" {
			type overrideSpec struct {
				ClientIDReference     *v1alpha1.SecretKeyReference `json:"clientIDReference"`
				ClientSecretReference *v1alpha1.SecretKeyReference `json:"clientSecretReference"`
			}
			var spec overrideSpec
			if err := json.Unmarshal([]byte(raw), &spec); err != nil {
				return OIDCInfo{}, fmt.Errorf("oidc-override: %w", err)
			}
			if spec.ClientIDReference != nil && spec.ClientIDReference.Name != "" && spec.ClientIDReference.Key != "" {
				val, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, orgName, *spec.ClientIDReference)
				if err != nil {
					return OIDCInfo{}, err
				}
				oidc.ClientID = val
			}
			if spec.ClientSecretReference != nil && spec.ClientSecretReference.Name != "" && spec.ClientSecretReference.Key != "" {
				val, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, orgName, *spec.ClientSecretReference)
				if err != nil {
					return OIDCInfo{}, err
				}
				oidc.ClientSecret = val
			}
		}
	}

	return oidc, nil
}

// resolvedSpecOverrides holds the fully-resolved (secretRef → plaintext) values from
// a ClusterKubeconfigOverrides struct, ready to be applied to the kubeconfig data.
type resolvedSpecOverrides struct {
	clusters       []resolvedClusterOverride
	users          []resolvedUserOverride
	contexts       []v1alpha1.ContextItemOverride
	currentContext string
}

type resolvedClusterOverride struct {
	name                     string
	server                   string
	certificateAuthorityData []byte
}

type resolvedUserOverride struct {
	name         string
	clientID     string
	clientSecret string
	issuerURL    string
}

// resolveSpecOverrides reads a ClusterKubeconfigOverrides and resolves any secretRef fields
// by fetching the referenced secrets. Plain string values are passed through as-is.
func resolveSpecOverrides(ctx context.Context, c client.Client, namespace string, overrides *v1alpha1.ClusterKubeconfigOverrides) (*resolvedSpecOverrides, error) {
	if overrides == nil {
		return nil, nil
	}
	out := &resolvedSpecOverrides{
		currentContext: overrides.CurrentContext,
		contexts:       overrides.Contexts,
	}

	for _, co := range overrides.Clusters {
		r := resolvedClusterOverride{name: co.Name, server: co.Server}
		if co.CertificateAuthorityDataRef != nil {
			val, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, c, namespace, *co.CertificateAuthorityDataRef)
			if err != nil {
				return nil, err
			}
			r.certificateAuthorityData = []byte(val)
		}
		out.clusters = append(out.clusters, r)
	}

	for _, uo := range overrides.Users {
		r := resolvedUserOverride{name: uo.Name}
		if uo.OIDC == nil {
			out.users = append(out.users, r)
			continue
		}
		// Inline values take precedence; secretRef is only resolved when the inline field is empty.
		r.clientID = uo.OIDC.ClientID
		r.issuerURL = uo.OIDC.IssuerURL
		if r.clientID == "" && uo.OIDC.ClientIDRef != nil {
			val, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, c, namespace, *uo.OIDC.ClientIDRef)
			if err != nil {
				return nil, err
			}
			r.clientID = val
		}
		if uo.OIDC.ClientSecretRef != nil {
			val, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, c, namespace, *uo.OIDC.ClientSecretRef)
			if err != nil {
				return nil, err
			}
			r.clientSecret = val
		}
		out.users = append(out.users, r)
	}

	return out, nil
}

// applyResolvedOverrides merges the resolved overrides into data by matching on name.
// Fields left empty in an override are not written, preserving the baseline value.
// Returns an error if an override references a name that does not exist in data, or if
// currentContext points to a non-existent context.
func applyResolvedOverrides(data *v1alpha1.ClusterKubeconfigData, overrides *resolvedSpecOverrides) error {
	if overrides == nil {
		return nil
	}

	if overrides.currentContext != "" {
		found := false
		for _, ctx := range data.Contexts {
			if ctx.Name == overrides.currentContext {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("configOverrides.currentContext %q does not match any existing context", overrides.currentContext)
		}
		data.CurrentContext = overrides.currentContext
	}

	for _, co := range overrides.clusters {
		found := false
		for i := range data.Clusters {
			if data.Clusters[i].Name != co.name {
				continue
			}
			found = true
			if co.server != "" {
				data.Clusters[i].Cluster.Server = co.server
			}
			if len(co.certificateAuthorityData) > 0 {
				data.Clusters[i].Cluster.CertificateAuthorityData = co.certificateAuthorityData
			}
		}
		if !found {
			return fmt.Errorf("configOverrides.clusters: name %q does not match any existing cluster entry", co.name)
		}
	}

	for _, uo := range overrides.users {
		found := false
		for i := range data.AuthInfo {
			if data.AuthInfo[i].Name != uo.name {
				continue
			}
			found = true
			cfg := data.AuthInfo[i].AuthInfo.AuthProvider.Config
			if cfg == nil {
				cfg = make(map[string]string)
				data.AuthInfo[i].AuthInfo.AuthProvider.Config = cfg
			}
			if uo.clientID != "" {
				cfg["client-id"] = uo.clientID
			}
			if uo.clientSecret != "" {
				cfg["client-secret"] = uo.clientSecret
			}
			if uo.issuerURL != "" {
				cfg["idp-issuer-url"] = uo.issuerURL
			}
		}
		if !found {
			return fmt.Errorf("configOverrides.users: name %q does not match any existing user entry", uo.name)
		}
	}

	for _, co := range overrides.contexts {
		found := false
		for i := range data.Contexts {
			if data.Contexts[i].Name != co.Name {
				continue
			}
			found = true
			if co.Cluster != "" {
				// Validate that the referenced cluster entry exists.
				clusterFound := false
				for _, cl := range data.Clusters {
					if cl.Name == co.Cluster {
						clusterFound = true
						break
					}
				}
				if !clusterFound {
					return fmt.Errorf("configOverrides.contexts[%q].cluster %q does not match any existing cluster entry", co.Name, co.Cluster)
				}
				data.Contexts[i].Context.Cluster = co.Cluster
			}
			if co.User != "" {
				// Validate that the referenced user entry exists.
				userFound := false
				for _, u := range data.AuthInfo {
					if u.Name == co.User {
						userFound = true
						break
					}
				}
				if !userFound {
					return fmt.Errorf("configOverrides.contexts[%q].user %q does not match any existing user entry", co.Name, co.User)
				}
				data.Contexts[i].Context.AuthInfo = co.User
			}
			if co.Namespace != "" {
				data.Contexts[i].Context.Namespace = co.Namespace
			}
		}
		if !found {
			return fmt.Errorf("configOverrides.contexts: name %q does not match any existing context entry", co.Name)
		}
	}

	return nil
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
	// start from current status
	status := ck.Status.DeepCopy()

	// on first creation
	if len(status.Conditions.Conditions) == 0 {
		status.Conditions.SetConditions(
			greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigCreatedCondition, "NewCreation", ""),
		)
	}

	// ensure all exposed conditions exist
	for _, ct := range ExposedKubeconfigConditions {
		if status.Conditions.GetConditionByType(ct) == nil {
			status.Conditions.SetConditions(greenhousemetav1alpha1.UnknownCondition(ct, "", ""))
		}
	}

	// determine whether the kubeconfig data is now present/usable
	hasCluster := len(ck.Spec.Kubeconfig.Clusters) > 0
	hasAuth := len(ck.Spec.Kubeconfig.AuthInfo) > 0
	hasContext := len(ck.Spec.Kubeconfig.Contexts) > 0 && ck.Spec.Kubeconfig.CurrentContext != ""

	isConfigured := hasCluster && hasAuth && hasContext

	failed := status.Conditions.GetConditionByType(v1alpha1.KubeconfigReconcileFailedCondition)

	switch {
	// Success path: spec is configured AND we didn't set a failure in this reconcile.
	case isConfigured && (failed == nil || failed.Status != metav1.ConditionTrue):
		status.Conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReadyCondition, "Complete", ""))
		status.Conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(v1alpha1.KubeconfigReconcileFailedCondition, "ReadyState", ""))
		status.Conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(v1alpha1.KubeconfigCreatedCondition, "ReadyState", ""))
	// Error path: controller explicitly marked failure this run.
	case failed != nil && failed.Status == metav1.ConditionTrue:
		status.Conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(v1alpha1.KubeconfigReadyCondition, "ReconcileFailed", ""))
	// Incomplete but no explicit failure (should still not be Ready).
	default:
		status.Conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(v1alpha1.KubeconfigReadyCondition, "Incomplete", ""))
	}

	return *status
}

var ExposedKubeconfigConditions = []greenhousemetav1alpha1.ConditionType{
	v1alpha1.KubeconfigCreatedCondition,
	v1alpha1.KubeconfigReconcileFailedCondition,
	v1alpha1.KubeconfigReadyCondition,
}

// nonLabelChars matches any character that is not alphanumeric, '-', '_', or '.'.
var nonLabelChars = regexp.MustCompile(`[^a-zA-Z0-9\-_.]+`)

// labelSafeVersion converts a Kubernetes version string (e.g. "v1.31.4+k3s1") into
// a value that satisfies Kubernetes label-value constraints (≤63 chars,
// alphanumeric/-/./_ only, must start and end with alphanumeric).
// Returns the sanitized value and true when the result is valid, or ("", false)
// when the version string cannot be represented as a valid label value.
// Selectors should match the normalized form, e.g. "v1.31.4-k3s1".
func labelSafeVersion(version string) (string, bool) {
	// Replace '+' with '-' to preserve build metadata readably (v1.31.4+k3s1 → v1.31.4-k3s1).
	safe := strings.ReplaceAll(version, "+", "-")
	// Replace any remaining disallowed characters with '-'.
	safe = nonLabelChars.ReplaceAllString(safe, "-")
	// Truncate to 63 characters.
	if len(safe) > 63 {
		safe = safe[:63]
	}
	// Trim leading/trailing characters that are not alphanumeric.
	safe = strings.TrimRight(safe, "-_.")
	safe = strings.TrimLeft(safe, "-_.")
	if safe == "" {
		return "", false
	}
	if len(k8svalidation.IsValidLabelValue(safe)) > 0 {
		return "", false
	}
	return safe, true
}

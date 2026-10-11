// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
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
		Watches(&v1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(sameNameResource)).
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

	var clusterCfg *clientcmdapi.Cluster
	for _, v := range kubeCfg.Clusters {
		clusterCfg = v
		break
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

	// collect oidc data and update kubeconfig
	result, err := clientutil.CreateOrPatch(ctx, r.Client, &kubeconfig, func() error {
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
		// Reflect the error in status and avoid sticky issues later.
		kubeconfig.Status.Conditions.SetConditions(
			greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "PatchError", err.Error()),
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
	l.Info("kubeconfig updated", "result", result)

	// Apply generic kubeconfig overrides from annotation greenhouse.sap/kubeconfig-overrides.
	// This runs after oidc-override and the main spec patch, so it wins on any overlapping field.
	if raw, ok := cluster.Annotations[greenhouseapis.AnnotationKeyKubeconfigOverrides]; ok && raw != "" {
		resolved, oErr := resolveKubeconfigOverrides(ctx, r.Client, cluster.Namespace, raw)
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
		patched, oErr := applyKubeconfigOverrides(kubeconfig.Spec.Kubeconfig, resolved)
		if oErr != nil {
			kubeconfig.Status.Conditions.SetConditions(
				greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "OverrideError", oErr.Error()),
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
		_, oErr = clientutil.CreateOrPatch(ctx, r.Client, &kubeconfig, func() error {
			kubeconfig.Spec.Kubeconfig = patched
			return nil
		})
		if oErr != nil {
			kubeconfig.Status.Conditions.SetConditions(
				greenhousemetav1alpha1.TrueCondition(v1alpha1.KubeconfigReconcileFailedCondition, "OverridePatchError", oErr.Error()),
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
		l.Info("kubeconfig overrides applied")
	}

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

// getOIDCInfo fetches OIDC configuration from the Organization and allows Cluster-level overrides via annotation-based secret references.
// Clusters can set annotation greenhouse.sap/oidc-override with JSON:
//
//	{
//	  "clientIDReference": {"name": "<secret-name>", "key": "<secret-key>"},
//	  "clientSecretReference": {"name": "<secret-name>", "key": "<secret-key>"}
//	}
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

	// Cluster-level overrides via annotation greenhouse.sap/oidc-override
	if cluster != nil && cluster.Annotations != nil {
		const annKey = "greenhouse.sap/oidc-override"
		if raw, ok := cluster.Annotations[annKey]; ok && raw != "" {
			// Define struct to deserialize annotation
			type overrideSpec struct {
				ClientIDReference     *v1alpha1.SecretKeyReference `json:"clientIDReference"`
				ClientSecretReference *v1alpha1.SecretKeyReference `json:"clientSecretReference"`
			}
			var spec overrideSpec
			if err := json.Unmarshal([]byte(raw), &spec); err != nil {
				return OIDCInfo{}, err
			}
			// Apply overrides from referenced secrets if provided
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

// indexedSegment matches a path segment with an array index, e.g. "clusters[0]".
var indexedSegment = regexp.MustCompile(`^([\w][\w-]*)\[(\d+)\]$`)

// secretRefValue is a helper struct used only for unmarshalling secretRef values in the override map.
type secretRefValue struct {
	SecretRef *v1alpha1.SecretKeyReference `json:"secretRef"`
}

// resolveKubeconfigOverrides unmarshals the raw annotation value and resolves any secretRef entries
// by reading the referenced secret. Plain string values are returned as-is.
func resolveKubeconfigOverrides(ctx context.Context, c client.Client, namespace string, raw string) (map[string]string, error) {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &rawMap); err != nil {
		return nil, fmt.Errorf("kubeconfig-overrides: invalid JSON: %w", err)
	}
	resolved := make(map[string]string, len(rawMap))
	for path, rawVal := range rawMap {
		var ref secretRefValue
		if err := json.Unmarshal(rawVal, &ref); err == nil && ref.SecretRef != nil {
			val, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, c, namespace, *ref.SecretRef)
			if err != nil {
				return nil, fmt.Errorf("kubeconfig-overrides: path %q: %w", path, err)
			}
			resolved[path] = val
			continue
		}
		var s string
		if err := json.Unmarshal(rawVal, &s); err != nil {
			return nil, fmt.Errorf("kubeconfig-overrides: path %q: value must be a string or secretRef object", path)
		}
		resolved[path] = s
	}
	return resolved, nil
}

// applyKubeconfigOverrides applies a flat map of dot-path → string value overrides onto data.
// Keys use dot-separated JSON field names with optional [n] array indexing, e.g.:
//
//	"clusters[0].cluster.server"
//	"users[0].user.auth-provider.config.client-id"
//	"contexts[0].context.namespace"
func applyKubeconfigOverrides(data v1alpha1.ClusterKubeconfigData, overrides map[string]string) (v1alpha1.ClusterKubeconfigData, error) {
	// Round-trip through map[string]any so we can apply arbitrary dot-path writes.
	b, err := json.Marshal(data)
	if err != nil {
		return data, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return data, err
	}

	for path, val := range overrides {
		if err := setDotPath(m, path, val); err != nil {
			return data, fmt.Errorf("kubeconfig-overrides: path %q: %w", path, err)
		}
	}

	patched, err := json.Marshal(m)
	if err != nil {
		return data, err
	}
	var result v1alpha1.ClusterKubeconfigData
	if err := json.Unmarshal(patched, &result); err != nil {
		return data, err
	}
	return result, nil
}

// setDotPath sets val at the dot-separated path inside m, creating intermediate maps as needed.
// Array elements (e.g. "clusters[0]") must already exist — the function sets inside them, not appends.
func setDotPath(m map[string]any, path string, val string) error {
	segments := strings.Split(path, ".")
	var cur any = m
	for i, seg := range segments {
		isLast := i == len(segments)-1

		if match := indexedSegment.FindStringSubmatch(seg); match != nil {
			key := match[1]
			idx, _ := strconv.Atoi(match[2])

			node, ok := cur.(map[string]any)
			if !ok {
				return fmt.Errorf("expected object at %q", strings.Join(segments[:i], "."))
			}
			arr, ok := node[key].([]any)
			if !ok || idx >= len(arr) {
				return fmt.Errorf("array %q index %d out of range", key, idx)
			}
			if isLast {
				arr[idx] = val
				return nil
			}
			cur = arr[idx]
		} else {
			node, ok := cur.(map[string]any)
			if !ok {
				return fmt.Errorf("expected object at %q", strings.Join(segments[:i], "."))
			}
			if isLast {
				node[seg] = val
				return nil
			}
			next, exists := node[seg]
			if !exists {
				next = map[string]any{}
				node[seg] = next
			}
			cur = next
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

// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/pkg/errors"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// BootstrapPhase holds the per-reconcile state shared by the bootstrap subroutines.
// It closes over the kubeconfig Secret being reconciled; each subroutine is idempotent
// so the chain re-runs from the top on every reconcile.
type BootstrapPhase struct {
	Client                  client.Client
	Scheme                  *runtime.Scheme
	Secret                  *corev1.Secret
	WorkloadIdentityEnabled bool
}

func (p *BootstrapPhase) EnsureCreatePhases() []lifecycle.SubRoutine {
	commonPhases := []lifecycle.SubRoutine{
		p.ensureCluster(),
		p.ensureOwnerReferences(),
	}

	if p.Secret.Type == greenhouseapis.SecretTypeOIDCConfig {
		// under workload identity Flux mints tokens itself: write the config map and skip the
		// requeue since there is no short-TTL kubeconfig to periodically rotat.
		if p.WorkloadIdentityEnabled {
			return append([]lifecycle.SubRoutine{p.ensureWorkloadIdentityConfigMap()}, commonPhases...)
		}
		return append([]lifecycle.SubRoutine{p.ensureOIDCKubeConfig()}, append(commonPhases, p.requeue())...)
	}
	return append(commonPhases, p.requeue())
}

func (p *BootstrapPhase) EnsureDeletePhases() []lifecycle.SubRoutine {
	return []lifecycle.SubRoutine{
		p.ensureClusterDeleted(),
	}
}

// ensureOIDCKubeConfig ensures a valid greenhousekubeconfig is present on the secret,
// regenerating it when missing or when the secret CA no longer matches the embedded CA.
func (p *BootstrapPhase) ensureOIDCKubeConfig() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		genTime, genTimeAvail := p.Secret.Annotations[greenhouseapis.SecretOIDCConfigGeneratedOnAnnotation]
		if !genTimeAvail || !clientutil.IsSecretContainsKey(p.Secret, greenhouseapis.GreenHouseKubeConfigKey) {
			sa := utils.NewServiceAccount(p.Secret.GetName(), p.Secret.GetNamespace())
			if _, err := controllerutil.CreateOrPatch(ctx, p.Client, sa, func() error {
				return controllerutil.SetOwnerReference(p.Secret, sa, p.Scheme)
			}); err != nil {
				return lifecycle.Break(), errors.Wrap(err, "failed creating service account for OIDC config")
			}
			log.FromContext(ctx).Info("OIDC config generated", "date", genTime, "namespace", p.Secret.GetNamespace(), "name", p.Secret.GetName())
			if err := p.createKubeConfigKey(ctx); err != nil {
				return lifecycle.Break(), err
			}
			return lifecycle.Continue(), nil
		}

		equality, err := compareCAWithKubeConfigCA(p.Secret)
		if err != nil {
			return lifecycle.Break(), errors.Wrap(err, "failed to create rest client from secret")
		}
		if !equality {
			log.FromContext(ctx).Info("KubeConfig CA does not match with secret CA, updating kubeconfig", "namespace", p.Secret.GetNamespace(), "name", p.Secret.GetName())
			if err := p.createKubeConfigKey(ctx); err != nil {
				return lifecycle.Break(), err
			}
		}
		return lifecycle.Continue(), nil
	}
}

// ensureWorkloadIdentityConfigMap ensures the ServiceAccount exists and writes the Flux
// remote ConfigMap consumed via kubeConfig.configMapRef. No static kubeconfig is generated;
// Flux mints ServiceAccount tokens itself via ObjectLevelWorkloadIdentity.
func (p *BootstrapPhase) ensureWorkloadIdentityConfigMap() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		sa := utils.NewServiceAccount(p.Secret.GetName(), p.Secret.GetNamespace())
		if _, err := controllerutil.CreateOrPatch(ctx, p.Client, sa, func() error {
			return controllerutil.SetOwnerReference(p.Secret, sa, p.Scheme)
		}); err != nil {
			return lifecycle.Break(), errors.Wrap(err, "failed creating service account for OIDC config")
		}

		caDecoded, err := base64.StdEncoding.DecodeString(string(p.Secret.Data[greenhouseapis.SecretAPIServerCAKey]))
		if err != nil {
			return lifecycle.Break(), errors.Wrap(err, "failed decoding certificate data")
		}

		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: p.Secret.GetName(), Namespace: p.Secret.GetNamespace()}}
		if _, err := controllerutil.CreateOrUpdate(ctx, p.Client, cm, func() error {
			cm.Data = map[string]string{
				fluxmeta.KubeConfigKeyAddress:            p.Secret.Annotations[greenhouseapis.SecretAPIServerURLAnnotation],
				fluxmeta.KubeConfigKeyAudiences:          greenhouseapis.OIDCAudience,
				fluxmeta.KubeConfigKeyServiceAccountName: sa.GetName(),
				fluxmeta.KubeConfigKeyProvider:           "generic",
				fluxmeta.KubeConfigKeyCACert:             string(caDecoded),
			}
			return controllerutil.SetOwnerReference(p.Secret, cm, p.Scheme)
		}); err != nil {
			return lifecycle.Break(), errors.Wrap(err, "failed to reconcile workload identity config map")
		}

		// drop static kubeconfig that existed before workload identity was enabled
		if clientutil.IsSecretContainsKey(p.Secret, greenhouseapis.GreenHouseKubeConfigKey) {
			delete(p.Secret.Data, greenhouseapis.GreenHouseKubeConfigKey)
			delete(p.Secret.Annotations, greenhouseapis.SecretOIDCConfigGeneratedOnAnnotation)
			if err := p.Client.Update(ctx, p.Secret); err != nil {
				return lifecycle.Break(), errors.Wrap(err, "failed removing stale kubeconfig from secret")
			}
		}
		return lifecycle.Continue(), nil
	}
}

// ensureCluster creates or updates the Cluster resource that mirrors the kubeconfig Secret.
func (p *BootstrapPhase) ensureCluster() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		cluster, err := p.getCluster(ctx)
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to get cluster", "namespace", p.Secret.GetNamespace(), "name", p.Secret.GetName())
			return lifecycle.Break(), err
		}
		if err := p.createOrUpdateCluster(ctx, cluster); err != nil {
			return lifecycle.Break(), err
		}
		return lifecycle.Continue(), nil
	}
}

// ensureOwnerReferences adds the ownerReference to the kubeconfig Secret so it is
// garbage collected when the Cluster is deleted.
func (p *BootstrapPhase) ensureOwnerReferences() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		cluster := &greenhousev1alpha1.Cluster{}
		if err := p.Client.Get(ctx, types.NamespacedName{Namespace: p.Secret.GetNamespace(), Name: p.Secret.GetName()}, cluster); err != nil {
			return lifecycle.Break(), err
		}
		if cluster.DeletionTimestamp != nil {
			return lifecycle.Continue(), nil
		}
		if _, err := controllerutil.CreateOrPatch(ctx, p.Client, p.Secret, func() error {
			return controllerutil.SetOwnerReference(cluster, p.Secret, p.Scheme)
		}); err != nil {
			return lifecycle.Break(), err
		}
		return lifecycle.Continue(), nil
	}
}

// requeue schedules the next periodic reconcile of the bootstrap Secret.
func (p *BootstrapPhase) requeue() lifecycle.SubRoutine {
	return func(_ context.Context) (lifecycle.Result, error) {
		return lifecycle.RequeueAfter(utils.DefaultRequeueInterval), nil
	}
}

// ensureClusterDeleted requests deletion of the Cluster owned by the Secret. Once the
// Cluster is gone the chain succeeds and lifecycle.ReconcileObject removes the finalizer.
func (p *BootstrapPhase) ensureClusterDeleted() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		cluster := &greenhousev1alpha1.Cluster{}
		if err := p.Client.Get(ctx, client.ObjectKeyFromObject(p.Secret), cluster); err != nil {
			// cluster is gone — chain succeeds and the finalizer is removed
			if client.IgnoreNotFound(err) == nil {
				return lifecycle.Continue(), nil
			}
			return lifecycle.Break(), err
		}
		log.FromContext(ctx).Info("Cluster secret is being deleted, requesting Cluster deletion", "cluster", client.ObjectKeyFromObject(p.Secret).String())
		if err := client.IgnoreNotFound(p.Client.Delete(ctx, cluster)); err != nil {
			return lifecycle.Break(), err
		}
		return lifecycle.RequeueAfter(10 * time.Second), nil
	}
}

func (p *BootstrapPhase) createKubeConfigKey(ctx context.Context) error {
	annotations := p.Secret.GetAnnotations()
	remoteAPIServerURL := annotations[greenhouseapis.SecretAPIServerURLAnnotation]
	certData := p.Secret.Data[greenhouseapis.SecretAPIServerCAKey]
	certDecoded, err := base64.StdEncoding.DecodeString(string(certData))
	if err != nil {
		return errors.Wrap(err, "failed decoding certificate data")
	}

	// create token request from SA with audience
	clusterResourceSA := utils.NewServiceAccount(p.Secret.GetName(), p.Secret.GetNamespace())
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{greenhouseapis.OIDCAudience},
			ExpirationSeconds: ptr.To[int64](600),
		},
	}
	if err := p.Client.SubResource("token").Create(ctx, clusterResourceSA, tokenRequest); err != nil {
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
	kubeconfigByte, err := clientcmd.Write(generator.RestConfigToAPIConfig(p.Secret.GetName()))
	if err != nil {
		return errors.Wrapf(err, "failed to generate kubeconfig for cluster %s", p.Secret.GetName())
	}
	// update secret with kubeconfig on greenhousekubeconfig key and update oidc generated on annotation
	p.Secret.Data[greenhouseapis.GreenHouseKubeConfigKey] = kubeconfigByte
	annotations[greenhouseapis.SecretOIDCConfigGeneratedOnAnnotation] = metav1.Now().Format(time.DateTime)
	p.Secret.Annotations = annotations
	return p.Client.Update(ctx, p.Secret)
}

// createOrUpdateCluster creates or updates the cluster resource.
func (p *BootstrapPhase) createOrUpdateCluster(ctx context.Context, cluster *greenhousev1alpha1.Cluster) error {
	// Ignore clusters about to be deleted.
	if !cluster.DeletionTimestamp.IsZero() {
		return nil
	}

	cluster.SetName(p.Secret.Name)
	cluster.SetNamespace(p.Secret.Namespace)

	annotations := cluster.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}
	switch p.Secret.Type {
	case greenhouseapis.SecretTypeKubeConfig:
		annotations[greenhouseapis.ClusterConnectivityAnnotation] = greenhouseapis.ClusterConnectivityKubeconfig
	case greenhouseapis.SecretTypeOIDCConfig:
		annotations[greenhouseapis.ClusterConnectivityAnnotation] = greenhouseapis.ClusterConnectivityOIDC
	}

	result, err := controllerutil.CreateOrUpdate(ctx, p.Client, cluster, func() error {
		cluster.SetAnnotations(annotations)
		cluster.Spec.AccessMode = greenhousev1alpha1.ClusterAccessModeDirect
		// Transport KubeConfigSecret labels to Cluster
		cluster = (lifecycle.NewPropagator(p.Secret, cluster).Apply()).(*greenhousev1alpha1.Cluster)
		return nil
	})
	if err != nil {
		return err
	}
	if result != controllerutil.OperationResultNone {
		log.FromContext(ctx).Info(fmt.Sprintf("%s cluster", result), "namespace", cluster.Namespace, "name", cluster.Name)
	}
	return nil
}

func (p *BootstrapPhase) getCluster(ctx context.Context) (*greenhousev1alpha1.Cluster, error) {
	cluster := new(greenhousev1alpha1.Cluster)
	err := p.Client.Get(ctx, client.ObjectKeyFromObject(p.Secret), cluster)
	return cluster, client.IgnoreNotFound(err)
}

func compareCAWithKubeConfigCA(secret *corev1.Secret) (bool, error) {
	restClient, err := clientutil.NewRestClientGetterFromSecret(secret, secret.GetNamespace())
	if err != nil {
		return false, err
	}
	restConfig, err := restClient.ToRESTConfig()
	if err != nil {
		return false, err
	}
	secretCertBytes, err := base64.StdEncoding.DecodeString(string(secret.Data[greenhouseapis.SecretAPIServerCAKey]))
	if err != nil {
		return false, errors.Wrap(err, "failed decoding certificate data from secret")
	}
	return strings.Compare(string(secretCertBytes), string(restConfig.CAData)) == 0, nil
}

// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/changemanagement"
)

const changeManagementConfigKey = "changeManagementConfig"

var changeReportRetryInterval = time.Minute

type changeManagementConfig struct {
	Endpoint        string `yaml:"endpoint"`
	SecretName      string `yaml:"secretName"`
	PayloadTemplate string `yaml:"payloadTemplate"`
}

// changeProperties are the well-known properties available to the payload template.
type changeProperties struct {
	Organization string
	Plugin       *greenhousev1alpha1.Plugin
	Cluster      *greenhousev1alpha1.Cluster
	Release      *helmv2.Snapshot
}

// reportChange reports each new HelmRelease revision of the Plugin once.
func (r *PluginReconciler) reportChange(ctx context.Context, plugin *greenhousev1alpha1.Plugin) error {
	if !r.ChangeManagementEnabled {
		plugin.RemoveCondition(greenhousev1alpha1.ChangeReportedCondition)
		return nil
	}
	config, err := getChangeManagementConfig(ctx, r.Client, plugin.Namespace)
	if err != nil {
		return changeReportFailed(plugin, err)
	}
	if config == nil {
		plugin.RemoveCondition(greenhousev1alpha1.ChangeReportedCondition)
		return nil
	}

	helmRelease := &helmv2.HelmRelease{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(plugin), helmRelease); client.IgnoreNotFound(err) != nil {
		return changeReportFailed(plugin, err)
	}
	latest := helmRelease.Status.History.Latest()
	if plugin.Status.ChangeManagement == nil {
		recordBaseline(plugin, latest)
	}
	if latest != nil && latest.Digest != plugin.Status.ChangeManagement.LastReportedDigest {
		if err := r.sendChange(ctx, plugin, config, latest); err != nil {
			return changeReportFailed(plugin, err)
		}
		plugin.Status.ChangeManagement.LastReportedDigest = latest.Digest
	}
	plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.ChangeReportedCondition, "", ""))
	return nil
}

// recordBaseline marks what is deployed when change management gets switched on as reported, without sending it.
// A Plugin without a release yet starts empty, so its first install is reported.
func recordBaseline(plugin *greenhousev1alpha1.Plugin, latest *helmv2.Snapshot) {
	plugin.Status.ChangeManagement = &greenhousev1alpha1.ChangeManagementStatus{}
	if latest != nil {
		plugin.Status.ChangeManagement.LastReportedDigest = latest.Digest
	}
}

// changeReportFailed sets ChangeReported to False with the error.
// Transport errors get a fixed message, because they carry addresses that change on every attempt and each new message triggers another reconcile.
func changeReportFailed(plugin *greenhousev1alpha1.Plugin, err error) error {
	message := err.Error()
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		message = "failed to send the change to the change management endpoint, see the controller logs"
	}
	plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ChangeReportedCondition, "", message))
	return err
}

// getChangeManagementConfig returns nil if the Organization has not configured change management.
func getChangeManagementConfig(ctx context.Context, c client.Client, organization string) (*changeManagementConfig, error) {
	org := &greenhousev1alpha1.Organization{}
	if err := c.Get(ctx, types.NamespacedName{Name: organization}, org); err != nil || org.Spec.ConfigMapRef == "" {
		return nil, client.IgnoreNotFound(err)
	}
	configMap := &corev1.ConfigMap{}
	if err := c.Get(ctx, types.NamespacedName{Name: org.Spec.ConfigMapRef, Namespace: organization}, configMap); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	data, ok := configMap.Data[changeManagementConfigKey]
	if !ok {
		return nil, nil
	}
	config := &changeManagementConfig{}
	if err := yaml.Unmarshal([]byte(data), config); err != nil {
		return nil, fmt.Errorf("invalid %s in ConfigMap %s: %w", changeManagementConfigKey, org.Spec.ConfigMapRef, err)
	}
	return config, nil
}

// sendChange collects the payload and credentials for the revision and sends them.
func (r *PluginReconciler) sendChange(ctx context.Context, plugin *greenhousev1alpha1.Plugin, config *changeManagementConfig, release *helmv2.Snapshot) error {
	if reported, err := r.alreadyReported(ctx, plugin, release.Digest); err != nil || reported {
		return err
	}
	cluster := &greenhousev1alpha1.Cluster{}
	if plugin.Spec.ClusterName != "" {
		if err := r.Get(ctx, types.NamespacedName{Name: plugin.Spec.ClusterName, Namespace: plugin.Namespace}, cluster); err != nil {
			return err
		}
	}
	payload, err := changemanagement.Render(config.PayloadTemplate, changeProperties{
		Organization: plugin.Namespace,
		Plugin:       plugin,
		Cluster:      cluster,
		Release:      release,
	})
	if err != nil || payload == nil {
		return err
	}
	auth := changemanagement.Auth{}
	if config.SecretName != "" {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: config.SecretName, Namespace: plugin.Namespace}, secret); err != nil {
			return err
		}
		auth.Username = string(secret.Data["username"])
		auth.Password = string(secret.Data["password"])
		if auth.Username == "" || auth.Password == "" {
			return fmt.Errorf("secret %s is missing username or password", config.SecretName)
		}
	}
	return changemanagement.New(config.Endpoint, auth).Send(ctx, payload)
}

// alreadyReported reads the Plugin from the API server, because the cache can lag behind the status written by the previous reconcile.
func (r *PluginReconciler) alreadyReported(ctx context.Context, plugin *greenhousev1alpha1.Plugin, digest string) (bool, error) {
	current := &greenhousev1alpha1.Plugin{}
	if err := r.apiReader.Get(ctx, client.ObjectKeyFromObject(plugin), current); err != nil {
		return false, err
	}
	return current.Status.ChangeManagement != nil && current.Status.ChangeManagement.LastReportedDigest == digest, nil
}

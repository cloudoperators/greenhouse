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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/releaseevents"
)

const (
	releaseEventsConfigKey     = "releaseEventsConfig"
	releaseEventsSecretNameKey = "releaseEventsSecretName"
)

var releaseEventRetryInterval = time.Minute

type releaseEventsConfig struct {
	SecretName      string
	PayloadTemplate string
}

// releaseEventProperties are the well-known properties available to the payload template.
type releaseEventProperties struct {
	Organization string
	Plugin       *greenhousev1alpha1.Plugin
	Cluster      *greenhousev1alpha1.Cluster
	Release      *helmv2.Snapshot
}

// reportRelease reports each new HelmRelease revision of the Plugin once.
func (r *PluginReconciler) reportRelease(ctx context.Context, plugin *greenhousev1alpha1.Plugin) error {
	if !r.ReleaseEventsEnabled {
		plugin.RemoveCondition(greenhousev1alpha1.ReleaseEventSentCondition)
		return nil
	}
	config, err := getReleaseEventsConfig(ctx, r.Client, plugin.Namespace)
	if err != nil {
		return releaseEventFailed(plugin, err)
	}
	if config == nil {
		plugin.RemoveCondition(greenhousev1alpha1.ReleaseEventSentCondition)
		return nil
	}

	helmRelease := &helmv2.HelmRelease{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(plugin), helmRelease); client.IgnoreNotFound(err) != nil {
		return releaseEventFailed(plugin, err)
	}
	latest := helmRelease.Status.History.Latest()
	if plugin.Status.ReleaseEvents == nil {
		recordBaseline(plugin, latest)
	}
	if latest != nil && latest.Digest != plugin.Status.ReleaseEvents.LastReportedDigest {
		if err := r.sendReleaseEvent(ctx, plugin, config, latest); err != nil {
			return releaseEventFailed(plugin, err)
		}
		plugin.Status.ReleaseEvents.LastReportedDigest = latest.Digest
	}
	plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.ReleaseEventSentCondition, "", ""))
	return nil
}

// recordBaseline marks what is deployed when release events get switched on as reported, without sending it.
// A Plugin without a release yet starts empty, so its first install is reported.
func recordBaseline(plugin *greenhousev1alpha1.Plugin, latest *helmv2.Snapshot) {
	plugin.Status.ReleaseEvents = &greenhousev1alpha1.ReleaseEventsStatus{}
	if latest != nil {
		plugin.Status.ReleaseEvents.LastReportedDigest = latest.Digest
	}
}

// releaseEventFailed sets ReleaseEventSent to False with the error.
// Transport errors get a fixed message, because they carry addresses that change on every attempt and each new message triggers another reconcile.
func releaseEventFailed(plugin *greenhousev1alpha1.Plugin, err error) error {
	message := err.Error()
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		message = "failed to send the release event, see the controller logs"
	}
	plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ReleaseEventSentCondition, "", message))
	return err
}

// getReleaseEventsConfig returns nil if the Organization has not configured release events.
func getReleaseEventsConfig(ctx context.Context, c client.Client, organization string) (*releaseEventsConfig, error) {
	org := &greenhousev1alpha1.Organization{}
	if err := c.Get(ctx, types.NamespacedName{Name: organization}, org); err != nil || org.Spec.ConfigMapRef == "" {
		return nil, client.IgnoreNotFound(err)
	}
	configMap := &corev1.ConfigMap{}
	if err := c.Get(ctx, types.NamespacedName{Name: org.Spec.ConfigMapRef, Namespace: organization}, configMap); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	payloadTemplate, ok := configMap.Data[releaseEventsConfigKey]
	if !ok {
		return nil, nil
	}
	secretName := configMap.Data[releaseEventsSecretNameKey]
	if secretName == "" {
		return nil, fmt.Errorf("%s is missing in ConfigMap %s", releaseEventsSecretNameKey, org.Spec.ConfigMapRef)
	}
	return &releaseEventsConfig{SecretName: secretName, PayloadTemplate: payloadTemplate}, nil
}

// sendReleaseEvent collects the payload and credentials for the revision and sends them.
func (r *PluginReconciler) sendReleaseEvent(ctx context.Context, plugin *greenhousev1alpha1.Plugin, config *releaseEventsConfig, release *helmv2.Snapshot) error {
	if reported, err := r.alreadyReported(ctx, plugin, release.Digest); err != nil || reported {
		return err
	}
	cluster := &greenhousev1alpha1.Cluster{}
	if plugin.Spec.ClusterName != "" {
		if err := r.Get(ctx, types.NamespacedName{Name: plugin.Spec.ClusterName, Namespace: plugin.Namespace}, cluster); err != nil {
			return err
		}
	}
	payload, err := releaseevents.Render(config.PayloadTemplate, releaseEventProperties{
		Organization: plugin.Namespace,
		Plugin:       plugin,
		Cluster:      cluster,
		Release:      release,
	})
	if err != nil || payload == nil {
		return err
	}
	releaseEventsClient, err := r.getReleaseEventsClient(ctx, plugin.Namespace, config.SecretName)
	if err != nil {
		return err
	}
	return releaseEventsClient.Send(ctx, payload)
}

func (r *PluginReconciler) getReleaseEventsClient(ctx context.Context, namespace, secretName string) (*releaseevents.Client, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err != nil {
		return nil, err
	}
	endpoint := string(secret.Data["endpoint"])
	auth := releaseevents.Auth{
		Username:    string(secret.Data["username"]),
		Password:    string(secret.Data["password"]),
		TokenHeader: string(secret.Data["tokenHeader"]),
	}
	if endpoint == "" || auth.Username == "" || auth.Password == "" {
		return nil, fmt.Errorf("secret %s is missing endpoint, username or password", secretName)
	}
	switch authType := string(secret.Data["authType"]); authType {
	case "BASIC":
	case "OAUTH2":
		auth.TokenURL = string(secret.Data["tokenURL"])
		if auth.TokenURL == "" {
			return nil, fmt.Errorf("secret %s is missing tokenURL", secretName)
		}
	default:
		return nil, fmt.Errorf("secret %s has unsupported authType %q, expected BASIC or OAUTH2", secretName, authType)
	}
	return releaseevents.New(endpoint, auth), nil
}

// alreadyReported reads the Plugin from the API server, because the cache can lag behind the status written by the previous reconcile.
func (r *PluginReconciler) alreadyReported(ctx context.Context, plugin *greenhousev1alpha1.Plugin, digest string) (bool, error) {
	current := &greenhousev1alpha1.Plugin{}
	if err := r.apiReader.Get(ctx, client.ObjectKeyFromObject(plugin), current); err != nil {
		return false, err
	}
	return current.Status.ReleaseEvents != nil && current.Status.ReleaseEvents.LastReportedDigest == digest, nil
}

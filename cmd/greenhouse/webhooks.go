// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudoperators/greenhouse/internal/clientutil"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
	webhookv1alpha2 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha2"
)

const (
	webhookCertSecretName    = "greenhouse-webhook-cert"
	webhookServiceName       = "greenhouse-webhook-service"
	webhookCertDir           = "/tmp/k8s-webhook-server/serving-certs"
	webhookServerPort        = 9443
	validatingWebhookCfgName = "greenhouse-validating-webhook-configuration"
	mutatingWebhookCfgName   = "greenhouse-mutating-webhook-configuration"
)

var knownWebhooks = map[string]func(mgr ctrl.Manager) error{
	"cluster":                  webhookv1alpha1.SetupClusterWebhookWithManager,
	"secrets":                  webhookv1alpha1.SetupSecretWebhookWithManager,
	"organization":             webhookv1alpha1.SetupOrganizationWebhookWithManager,
	"pluginDefinition":         webhookv1alpha1.SetupPluginDefinitionWebhookWithManager,
	"plugin":                   webhookv1alpha1.SetupPluginWebhookWithManager,
	"pluginPreset":             webhookv1alpha1.SetupPluginPresetWebhookWithManager,
	"teamrole":                 webhookv1alpha1.SetupTeamRoleWebhookWithManager,
	"teamrolebinding_v1alpha2": webhookv1alpha2.SetupTeamRoleBindingWebhookWithManager,
	"team":                     webhookv1alpha1.SetupTeamWebhookWithManager,
	"clusterPluginDefinition":  webhookv1alpha1.SetupClusterPluginDefinitionWebhookWithManager,
	"serviceaccounts":          webhookv1alpha1.SetupServiceAccountWebhookWithManager,
}

// setupCertRotation registers the cert-controller rotator with the manager. The rotator
// generates a self-signed CA and serving cert, stores them in the webhook cert Secret.
func setupCertRotation(mgr ctrl.Manager) chan struct{} {
	namespace := clientutil.GetEnvOrDefault(podNamespaceEnv, defaultPodNamespace)
	certsReady := make(chan struct{})
	handleError(rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: namespace,
			Name:      webhookCertSecretName,
		},
		CertDir:        webhookCertDir,
		CAName:         "greenhouse-ca",
		CAOrganization: "greenhouse",
		DNSName:        fmt.Sprintf("%s.%s.svc", webhookServiceName, namespace),
		IsReady:        certsReady,
		Webhooks: []rotator.WebhookInfo{
			{Name: validatingWebhookCfgName, Type: rotator.Validating},
			{Name: mutatingWebhookCfgName, Type: rotator.Mutating},
		},
		FieldOwner:           "greenhouse-webhook",
		EnableReadinessCheck: true,
	}), "unable to set up cert rotation")
	handleError(mgr.AddReadyzCheck("webhook-tls", webhookTLSChecker(webhookCertDir, webhookServerPort)), "unable to set up webhook tls ready check")
	return certsReady
}

func webhookTLSChecker(certDir string, port int) func(*http.Request) error {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	// skip CA verification: the probe compares the served cert against certDir directly.
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	// disable keep-alive so a connection isn't reused across a cert rotation.
	tr.DisableKeepAlives = true
	insecureClient := &http.Client{Transport: tr}
	probeURL := "https://%s" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	return func(r *http.Request) error {
		ctx := context.Background()
		if r != nil {
			ctx = r.Context()
		}

		certPath := filepath.Join(certDir, "tls.crt")
		keyPath := filepath.Join(certDir, "tls.key")
		loadCert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return fmt.Errorf("unable to load certificate from certDir %s: %w", certDir, err)
		}

		return probeTLSURL(ctx, insecureClient, probeURL, loadCert.Certificate)
	}
}

func probeTLSURL(ctx context.Context, insecureClient *http.Client, probeURL string, expectedCerts [][]byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("unable to create probe request: %w", err)
	}
	resp, err := insecureClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to connect to server: %w", err)
	}

	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("unable to read probe response: %w", err)
	}
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return errors.New("webhook does not serve TLS certificate")
	}

	serverCerts := resp.TLS.PeerCertificates
	if len(serverCerts) != len(expectedCerts) {
		return fmt.Errorf("server certificate chain length does not match certificate in certDir, %d vs %d", len(serverCerts), len(expectedCerts))
	}
	for i, serverCert := range serverCerts {
		if !bytes.Equal(serverCert.Raw, expectedCerts[i]) {
			return fmt.Errorf("server certificate %d does not match certificate %d in certDir", i, i)
		}
	}
	return nil
}

func registerWebhooks(mgr ctrl.Manager) {
	for webhookName, hookFunc := range knownWebhooks {
		setupLog.Info("registering webhook", "name", webhookName)
		handleError(hookFunc(mgr), "unable to create webhook", "name", webhookName)
	}
}

func runWebhooks(mgr ctrl.Manager) {
	if !certRotatorEnabled {
		registerWebhooks(mgr)
		return
	}
	certsReady := setupCertRotation(mgr)
	go func() {
		<-certsReady // closed once the certs are ready, not opened during cert rotation.
		registerWebhooks(mgr)
	}()
}

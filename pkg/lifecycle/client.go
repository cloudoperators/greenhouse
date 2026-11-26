// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/transport"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var knownSchemes = []func(*runtime.Scheme) error{
	clientgoscheme.AddToScheme,
	apiextensionsv1.AddToScheme,
}

// NewRemoteClient returns a controller-runtime client for the given remote cluster.
func NewRemoteClient(ctx context.Context, kubeClient client.Client, cluster *greenhouseapisv1alpha1.Cluster) (client.Client, error) {
	cfg, err := NewRemoteKubeCfg(ctx, kubeClient, cluster)
	if err != nil {
		return nil, err
	}
	// For workload identity clusters the bearer token minted by GetWorkloadIdentityCfg expires after 600s.
	// To keep the client valid long-term, we clear the static token and attach a refreshing token source
	// that automatically mints a new token before the current one expires.
	if cluster.Annotations[greenhouseapis.ClusterWorkloadIdentityAnnotation] == greenhouseapis.ClusterWorkloadIdentityEnabled {
		cfg.BearerToken = ""
		cfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			src := &tokenRequestSource{
				client:    kubeClient,
				namespace: cluster.Namespace,
				name:      cluster.Name,
				tokenCtx: func() (context.Context, context.CancelFunc) {
					return context.WithTimeout(ctx, tokenMintTimeout)
				},
			}
			return transport.ResettableTokenSourceWrapTransport(transport.NewCachedTokenSource(src))(rt)
		}
	}
	sch := runtime.NewScheme()
	for _, addScheme := range knownSchemes {
		if err := addScheme(sch); err != nil {
			return nil, fmt.Errorf("scheme registration: %w", err)
		}
	}
	return client.New(cfg, client.Options{Scheme: sch})
}

const tokenMintTimeout = 10 * time.Second

type tokenRequestSource struct {
	client    client.Client
	namespace string
	name      string
	tokenCtx  func() (context.Context, context.CancelFunc)
}

func (s *tokenRequestSource) Token() (*oauth2.Token, error) {
	ctx, cancel := s.tokenCtx()
	defer cancel()
	tokenRequest, err := MintServiceAccountToken(ctx, s.client, s.namespace, s.name)
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken: tokenRequest.Status.Token,
		TokenType:   "Bearer",
		Expiry:      tokenRequest.Status.ExpirationTimestamp.Add(-60 * time.Second),
	}, nil
}

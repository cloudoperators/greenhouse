// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"k8s.io/client-go/transport"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const tokenExpiryLeeway = 60 * time.Second

func (pm *PmManager) workloadIdentityTransport(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (http.RoundTripper, string, error) {
	restCfg, err := lifecycle.NewRemoteKubeCfg(ctx, pm.reader, cluster)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build rest config: %w", err)
	}

	base, err := transport.New(&transport.Config{
		TLS: transport.TLSConfig{CAData: restCfg.CAData},
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to build base transport: %w", err)
	}

	src := &tokenRequestSource{
		client:    pm.reader,
		namespace: cluster.Namespace,
		name:      cluster.Name,
	}
	rt := transport.ResettableTokenSourceWrapTransport(transport.NewCachedTokenSource(src))(base)

	return rt, restCfg.Host, nil
}

type tokenRequestSource struct {
	client    client.Client
	namespace string
	name      string
}

func (s *tokenRequestSource) Token() (*oauth2.Token, error) {
	tokenRequest, err := lifecycle.MintServiceAccountToken(context.Background(), s.client, s.namespace, s.name)
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken: tokenRequest.Status.Token,
		TokenType:   "Bearer",
		Expiry:      tokenRequest.Status.ExpirationTimestamp.Add(-tokenExpiryLeeway),
	}, nil
}

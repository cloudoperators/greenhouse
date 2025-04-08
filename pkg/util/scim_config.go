// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

func GreenhouseSCIMConfigToSCIMConfig(ctx context.Context, k8sClient client.Client, config *greenhouseapisv1alpha1.SCIMConfig, namespace string) (*scim.Config, error) {
	cfg := &scim.Config{
		URL:      config.BaseURL,
		AuthType: config.AuthType,
	}
	switch cfg.AuthType {
	case scim.Basic:
		var err error
		username, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *config.BasicAuthUser.Secret)
		if err != nil {
			return nil, fmt.Errorf("secret for '.SCIMConfig.BasicAuthUser' is missing: %s", err.Error())
		}
		password, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *config.BasicAuthPw.Secret)
		if err != nil {
			return nil, fmt.Errorf("secret for BasicAuthPw is missing: %s", err.Error())
		}
		cfg.BasicAuth = &scim.BasicAuthConfig{
			Username: strings.Trim(string(username), "\n"),
			Password: strings.Trim(string(password), "\n"),
		}
	case scim.BearerToken:
		var err error
		token, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *config.BearerToken.Secret)
		if err != nil {
			return nil, fmt.Errorf("secret for BearerToken is missing: %s", err.Error())
		}
		cfg.BearerToken = &scim.BearerTokenConfig{
			Prefix: config.BearerPrefix,
			Header: config.BearerHeader,
			Token:  strings.Trim(string(token), "\n"),
		}
		if cfg.BearerToken.Header == "" {
			cfg.BearerToken.Header = "Authorization"
		}
		if cfg.BearerToken.Prefix == "" {
			cfg.BearerToken.Prefix = "Bearer"
		}
	default:
		return nil, fmt.Errorf("invalid authentication type: %s", config.AuthType)
	}
	return cfg, nil
}

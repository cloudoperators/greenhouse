// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

func GreenhouseSCIMConfigToSCIMConfig(ctx context.Context, c greenhouseapisv1alpha1.SCIMConfig, k8sClient client.Client, namespace string) (*scim.Config, error) {
	var basicAuthConfig *scim.BasicAuthConfig
	var bearerTokenConfig *scim.BearerTokenConfig
	switch c.AuthType {
	case scim.Basic:
		var err error
		basicAuthConfig = &scim.BasicAuthConfig{}
		basicAuthConfig.Username, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *c.BasicAuthUser.Secret)
		if err != nil {
			return nil, fmt.Errorf("secret for BasicAuthUser is missing: %s", err.Error())
		}
		basicAuthConfig.Password, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *c.BasicAuthPw.Secret)
		if err != nil {
			return nil, fmt.Errorf("secret for BasicAuthPw is missing: %s", err.Error())
		}
	case scim.BearerToken:
		var err error
		bearerTokenConfig = &scim.BearerTokenConfig{
			Prefix: c.BearerPrefix,
			Header: c.BearerHeader,
		}
		bearerTokenConfig.Token, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *c.BearerToken.Secret)
		if err != nil {
			return nil, fmt.Errorf("secret for BearerToken is missing: %s", err.Error())
		}
	default:
		return nil, errors.New("SCIM Config is not provided")
	}
	config := &scim.Config{
		URL:         c.BaseURL,
		AuthType:    c.AuthType,
		BasicAuth:   basicAuthConfig,
		BearerToken: bearerTokenConfig,
	}

	return config, nil
}

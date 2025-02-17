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

func GreenhouseSCIMConfigToSCIMConfig(ctx context.Context, k8sClient client.Client, config *greenhouseapisv1alpha1.SCIMConfig, namespace string) (*scim.Config, error) {
	basicAuthConfig := &scim.BasicAuthConfig{Username: "", Password: ""}
	bearerTokenConfig := &scim.BearerTokenConfig{
		Prefix: config.BearerPrefix,
		Header: config.BearerHeader,
	}
	switch config.AuthType {
	case scim.Basic:
		var err error
		basicAuthConfig.Username, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *config.BasicAuthUser.Secret)
		if err != nil {
			return nil, fmt.Errorf("secret for BasicAuthUser is missing: %s", err.Error())
		}
		basicAuthConfig.Password, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *config.BasicAuthPw.Secret)
		if err != nil {
			return nil, fmt.Errorf("secret for BasicAuthPw is missing: %s", err.Error())
		}
	case scim.BearerToken:
		var err error
		bearerTokenConfig.Token, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *config.BearerToken.Secret)
		if err != nil {
			return nil, fmt.Errorf("secret for BearerToken is missing: %s", err.Error())
		}
	default:
		return nil, errors.New("SCIM Config is not provided")
	}
	cfg := &scim.Config{
		URL:         config.BaseURL,
		AuthType:    config.AuthType,
		BasicAuth:   basicAuthConfig,
		BearerToken: bearerTokenConfig,
	}

	return cfg, nil
}

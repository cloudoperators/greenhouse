// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

func GreenhouseSCIMConfigToSCIMConfig(ctx context.Context, c greenhouseapisv1alpha1.SCIMConfig, k8sClient client.Client, namespace string, conditionType greenhouseapisv1alpha1.ConditionType) (*scim.Config, greenhouseapisv1alpha1.Condition, error) {
	var basicAuthConfig *scim.BasicAuthConfig
	var bearerTokenConfig *scim.BearerTokenConfig
	switch c.AuthType {
	case scim.Basic:
		var err error
		basicAuthConfig = &scim.BasicAuthConfig{}
		basicAuthConfig.Username, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *c.BasicAuthUser.Secret)
		if err != nil {
			return nil, greenhouseapisv1alpha1.FalseCondition(conditionType, greenhouseapisv1alpha1.SecretNotFoundReason, "BasicAuthUser missing"), err
		}
		basicAuthConfig.Password, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *c.BasicAuthPw.Secret)
		if err != nil {
			return nil, greenhouseapisv1alpha1.FalseCondition(conditionType, greenhouseapisv1alpha1.SecretNotFoundReason, "BasicAuthPw missing"), err
		}
	case scim.BearerToken:
		var err error
		bearerTokenConfig = &scim.BearerTokenConfig{
			Prefix: c.BearerPrefix,
			Header: c.BearerHeader,
		}
		bearerTokenConfig.Token, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *c.BearerToken.Secret)
		if err != nil {
			return nil, greenhouseapisv1alpha1.FalseCondition(conditionType, greenhouseapisv1alpha1.SecretNotFoundReason, "BearerToken missing"), err
		}
	default:
		return nil, greenhouseapisv1alpha1.FalseCondition(conditionType, greenhouseapisv1alpha1.SCIMConfigNotProvidedReason, "SCIM Config is not provided"), errors.New("SCIM Config is not provided")
	}
	config := &scim.Config{
		URL:         c.BaseURL,
		AuthType:    c.AuthType,
		BasicAuth:   basicAuthConfig,
		BearerToken: bearerTokenConfig,
	}

	return config, greenhouseapisv1alpha1.Condition{}, nil
}

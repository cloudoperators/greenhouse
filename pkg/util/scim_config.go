// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"errors"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

func GreenhouseSCIMConfigToSCIMConfig(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization, namespace string) (*scim.Config, error) {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: org.Spec.Authentication.SecretRef}, secret)
	if err != nil {
		return nil, err
	}

	basicAuthConfig := &scim.BasicAuthConfig{Username: "", Password: ""}
	bearerTokenConfig := &scim.BearerTokenConfig{
		Prefix: org.Spec.Authentication.SCIMConfig.BearerPrefix,
		Header: org.Spec.Authentication.SCIMConfig.BearerHeader,
	}
	switch org.Spec.Authentication.SCIMConfig.AuthType {
	case scim.Basic:
		scimBasicAuthUser, ok := secret.Data[greenhouseapisv1alpha1.SCIMBasicAuthUserKey]
		if !ok {
			return nil, errors.New("missing scimBasicAuthUser")
		}
		scimBasicAuthPassword, ok := secret.Data[greenhouseapisv1alpha1.SCIMBasicAuthPasswordKey]
		if !ok {
			return nil, errors.New("missing scimBasicAuthPassword")
		}

		basicAuthConfig = &scim.BasicAuthConfig{
			Username: strings.Trim(string(scimBasicAuthUser), "\n"),
			Password: strings.Trim(string(scimBasicAuthPassword), "\n")}

	case scim.BearerToken:
		scimBearerToken, ok := secret.Data[greenhouseapisv1alpha1.SCIMBearerTokenKey]
		if !ok {
			return nil, errors.New("scimBearerToken is missing")
		}

		bearerTokenConfig.Token = strings.Trim(string(scimBearerToken), "\n")
	default:
		return nil, errors.New("SCIM Config is not provided")
	}
	cfg := &scim.Config{
		URL:         org.Spec.Authentication.SCIMConfig.BaseURL,
		AuthType:    org.Spec.Authentication.SCIMConfig.AuthType,
		BasicAuth:   basicAuthConfig,
		BearerToken: bearerTokenConfig,
	}

	return cfg, nil
}

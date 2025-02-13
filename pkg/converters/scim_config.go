// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package converters

/*func GreenhouseSCIMConfigToSCIMConfig(ctx context.Context, c greenhouseapisv1alpha1.SCIMConfig, k8sClient client.Client, namespace string, conditionType greenhouseapisv1alpha1.ConditionType) (*scim.Config, greenhouseapisv1alpha1.Condition, error) {
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
}*/

/*func GreenhouseSCIMConfigToSCIMConfig(ctx context.Context, c greenhouseapisv1alpha1.SCIMConfig, k8sClient client.Client, namespace string) (*scim.Config, error) {
	var basicAuthConfig *scim.BasicAuthConfig
	var bearerTokenConfig *scim.BearerTokenConfig
	switch c.AuthType {
	case scim.Basic:
		var err error
		basicAuthConfig = &scim.BasicAuthConfig{}
		basicAuthConfig.Username, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *c.BasicAuthUser.Secret)
		if err != nil {
			return nil, errors.Wrap(err, "BasicAuthUser missing")
		}
		basicAuthConfig.Password, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *c.BasicAuthPw.Secret)
		if err != nil {
			return nil, errors.Wrap(err, "BasicAuthPw missing")
		}
	case scim.BearerToken:
		var err error
		bearerTokenConfig = &scim.BearerTokenConfig{
			Prefix: c.BearerPrefix,
			Header: c.BearerHeader,
		}
		bearerTokenConfig.Token, err = clientutil.GetSecretKeyFromSecretKeyReference(ctx, k8sClient, namespace, *c.BearerToken.Secret)
		if err != nil {
			return nil, errors.Wrap(err, "BearerToken missing")
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
*/

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

export const KUBECONFIGTEMPLATE = 
`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: ##ENDPOINT##
  name: default
contexts:
- context:
    cluster: default
    user: default
    namespace: ##NAMESPACE##
  name: default
current-context: default
preferences: {}
users:
- name: default
  user:
    token: ##TOKEN##`;

export const ENDPOINT_IDENTIFIER = '##ENDPOINT##';
export const NAMESPACE_IDENTIFIER = '##NAMESPACE##';
export const TOKEN_IDENTIFIER = '##TOKEN##';

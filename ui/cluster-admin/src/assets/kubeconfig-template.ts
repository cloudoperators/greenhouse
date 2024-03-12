/*
 * Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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

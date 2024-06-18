/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { ResultMessage, Secret } from "../../../types/types"
import useApi, { ApiResponse } from "./useApi"
import useNamespace from "./useNamespace"

export type SecretApiResponse = ResultMessage & {
  response?: Secret
}

export type SecretListApiResponse = ResultMessage & {
  response?: Secret[]
}

export const useSecretApi = () => {
  const { get, create, update, deleteObject } = useApi(false) // No debug logs on secrets
  const { namespace } = useNamespace()

  const getSecret = (secret: Secret): Promise<SecretApiResponse> => {
    return get<Secret>(
      `/api/v1/namespaces/${namespace}/secrets`,
      secret
    ) as Promise<SecretApiResponse>
  }

  const createSecret = (secret: Secret): Promise<SecretApiResponse> => {
    return create<Secret>(
      `/api/v1/namespaces/${namespace}/secrets`,
      secret
    ) as Promise<SecretApiResponse>
  }

  const updateSecret = (secret: Secret): Promise<SecretApiResponse> => {
    return update<Secret>(
      `/api/v1/namespaces/${namespace}/secrets`,
      secret
    ) as Promise<SecretApiResponse>
  }

  const deleteSecret = (secret: Secret): Promise<SecretApiResponse> => {
    return deleteObject<Secret>(
      `/api/v1/namespaces/${namespace}/secrets`,
      secret
    ) as Promise<SecretApiResponse>
  }

  return { getSecret, createSecret, updateSecret, deleteSecret }
}

export default useSecretApi

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { ResultMessage, Secret } from "../../../types/types"
import useStore from "../store"
import useApi from "./useApi"
import useNamespace from "./useNamespace"

export type SecretApiResponse = ResultMessage & {
  response?: Secret
}

export type SecretListApiResponse = ResultMessage & {
  response?: Secret[]
}

export const useSecretApi = () => {
  const { get, create, update, deleteObject, watch } = useApi(false) // No debug logs on secrets
  const { namespace } = useNamespace()
  const modifySecrets = useStore((state) => state.modifySecrets)
  const deleteSecrets = useStore((state) => state.deleteSecrets)

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

  const watchSecrets = () => {
    return watch<Secret>(
      `/api/v1/namespaces/${namespace}/secrets`,
      "Secret",
      modifySecrets,
      modifySecrets,
      deleteSecrets
    )
  }

  const watchSecretsWithoutHelm = () => {
    // exclude helm secrets from watch
    const fieldSelectorKey = "type"
    const isHelmSecretValue = "helm.sh/release.v1"
    const fieldSelector = `${fieldSelectorKey}!=${isHelmSecretValue}`
    const params = { fieldSelector: fieldSelector }

    return watch<Secret>(
      `/api/v1/namespaces/${namespace}/secrets`,
      "Secret",
      modifySecrets,
      modifySecrets,
      deleteSecrets,
      params
    )
  }

  return {
    getSecret,
    createSecret,
    updateSecret,
    deleteSecret,
    watchSecrets,
    watchSecretsWithoutHelm,
  }
}

export default useSecretApi

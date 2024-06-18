/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useCallback } from "react"
import useClient from "./useClient"
import useNamespace from "./useNamespace"
import useSecretApi from "./useSecretApi"
import { ResultMessage } from "../../../types/types"

export const useCheckAuthorized = () => {
  const { namespace } = useNamespace()
  const { client: client } = useClient()
  const { getSecret } = useSecretApi()

  const canListSecrets = useCallback(async (): Promise<ResultMessage> => {
    if (!client || !namespace) {
      return { ok: false, message: "" }
    }
    let result = await getSecret({ metadata: { name: "" }, kind: "Secret" })

    // we get back an empty secret list, if we are authorized
    // @ts-ignore
    if (result.response?.kind == "SecretList") {
      return { ok: true, message: "" }
    }

    return { ok: result.ok, message: result.message }
  }, [client, namespace])

  return { canListSecrets }
}

export default useCheckAuthorized

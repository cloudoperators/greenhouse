/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect } from "react"
import useCommunication from "../hooks/useCommunication"
import useUrlState from "../hooks/useUrlState"
import useSecretApi from "../../plugindefinitions/hooks/useSecretApi"
import usePluginDefinitionApi from "../../plugindefinitions/hooks/usePluginDefinitionApi"
import useStore from "../../plugindefinitions/store"
import { useActions } from "@cloudoperators/juno-messages-provider"

const AsyncWorker = () => {
  useCommunication()
  useUrlState()
  const { watchSecretsWithoutHelm } = useSecretApi()
  const { watchPluginDefinitions } = usePluginDefinitionApi()
  const auth = useStore((state) => state.auth)
  const { addMessage } = useActions()

  useEffect(() => {
    watchPluginDefinitions()
  }, [auth])

  useEffect(() => {
    watchSecretsWithoutHelm().then((res) => {
      // we bubble up a warning, if user is not authorized to watch secrets.
      // UI will still work for plugins, but user will not see secrets
      if (!res.ok) {
        if (res.message && res.status == 403) {
          addMessage({
            variant: "warning",
            text: res.message,
          })
        }
      }
    })
  }, [auth])

  return null
}

export default AsyncWorker

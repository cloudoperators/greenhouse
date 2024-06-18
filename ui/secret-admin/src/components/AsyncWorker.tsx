/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useActions } from "messages-provider"
import { useEffect } from "react"
import useCommunication from "../hooks/useCommunication"
import useCheckAuthorized from "../hooks/useIsAuthorized"
import useUrlState from "../hooks/useUrlState"
import useWatch from "../hooks/useWatch"
import useStore from "../store"

interface AsyncWorkerProps {
  consumerId: string
}

const AsyncWorker: React.FC<AsyncWorkerProps> = (props: AsyncWorkerProps) => {
  useUrlState(props.consumerId)
  useCommunication()
  const { canListSecrets } = useCheckAuthorized()
  const { addMessage } = useActions()
  const auth = useStore((state) => state.auth)
  const { watchSecrets } = useWatch()

  useEffect(() => {
    canListSecrets().then((res) => {
      if (!res.ok) {
        if (res.message) {
          addMessage({
            variant: "error",
            text: res.message,
          })
        }
      } else {
        if (!watchSecrets) return
        const unwatch = watchSecrets()
        return unwatch
      }
    })
  }, [auth])

  return null
}

export default AsyncWorker

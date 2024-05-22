/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect } from "react"
import useUrlState from "../hooks/useUrlState"
import useWatch from "../hooks/useWatch"
import useCommunication from "../hooks/useCommunication"

interface AsyncWorkerProps {
  consumerId: string
}

const AsyncWorker: React.FC<AsyncWorkerProps> = (props: AsyncWorkerProps) => {
  useUrlState(props.consumerId)
  useCommunication()

  const { watchSecrets } = useWatch()

  useEffect(() => {
    if (!watchSecrets) return
    const unwatch = watchSecrets()
    return unwatch
  }, [watchSecrets])

  return null
}

export default AsyncWorker

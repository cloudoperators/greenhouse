/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect } from "react"
import { useClusterApi } from "../hooks/useClusterApi"
import useUrlState from "../hooks/useUrlState"
import useStore from "../store"

interface AsyncWorkerProps {
  consumerId: string
}

const AsyncWorker: React.FC<AsyncWorkerProps> = (props: AsyncWorkerProps) => {
  useUrlState(props.consumerId)
  const auth = useStore((state) => state.auth)
  const { watchClusters } = useClusterApi()

  useEffect(() => {
    watchClusters()
  }, [auth])

  return null
}

export default AsyncWorker

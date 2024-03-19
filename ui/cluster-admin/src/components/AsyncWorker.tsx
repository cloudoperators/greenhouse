/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect } from "react"
import useUrlState from "../hooks/useUrlState"
import useWatch from "../hooks/useWatch"

interface AsyncWorkerProps {
  consumerId: string
}

const AsyncWorker: React.FC<AsyncWorkerProps> = (props: AsyncWorkerProps) => {
  useUrlState(props.consumerId)

  const { watchClusters: watchClusters } = useWatch()

  useEffect(() => {
    if (!watchClusters) return
    const unwatch = watchClusters()
    return unwatch
  }, [watchClusters])

  return null
}

export default AsyncWorker

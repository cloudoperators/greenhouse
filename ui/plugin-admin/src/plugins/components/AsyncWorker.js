/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {useEffect} from "react"
import useCommunication from "../hooks/useCommunication"
import useUrlState from "../hooks/useUrlState"
import useWatch from "../../plugindefinitions/hooks/useWatch"

const AsyncWorker = () => {
  useCommunication()
  useUrlState()

  const { watchPluginDefinitions, watchSecrets } = useWatch()

  useEffect(() => {
    if (!watchPluginDefinitions) return
    const unwatch = watchPluginDefinitions()
    return unwatch
  }, [watchPluginDefinitions])

  useEffect(() => {
    if (!watchSecrets) return
    console.log("watching secrets")
    const unwatch = watchSecrets()
    return unwatch
  }, [watchSecrets])


  return null
}

export default AsyncWorker

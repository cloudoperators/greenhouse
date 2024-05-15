/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import useApi, { ApiResponse } from "./useApi"
import { Plugin, PluginPreset } from "../../../../types/types"
import useNamespace from "./useNamespace"
import useClient from "./useClient"
import { useCallback } from "react"

export type PluginPresetApiResponse = {
  ok: boolean
  message: string
  response?: PluginPreset
}

export const usePluginPresetApi = () => {
  const { get, create, update, deleteObject } = useApi()
  const { namespace } = useNamespace()
  const { client } = useClient()

  const getPluginPreset = (
    pluginPreset: PluginPreset
  ): Promise<PluginPresetApiResponse> => {
    return get<PluginPreset>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/pluginpresets`,
      pluginPreset
    ) as Promise<PluginPresetApiResponse>
  }

  const createPluginPreset = (
    pluginPreset: PluginPreset
  ): Promise<PluginPresetApiResponse> => {
    return create<PluginPreset>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/pluginpresets`,
      pluginPreset
    ) as Promise<PluginPresetApiResponse>
  }

  const updatePluginPreset = (
    pluginPreset: PluginPreset
  ): Promise<PluginPresetApiResponse> => {
    return update<PluginPreset>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/pluginpresets`,
      pluginPreset
    ) as Promise<PluginPresetApiResponse>
  }

  const deletePluginPreset = (
    pluginPreset: PluginPreset
  ): Promise<PluginPresetApiResponse> => {
    return deleteObject<PluginPreset>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/pluginpresets`,
      pluginPreset
    ) as Promise<PluginPresetApiResponse>
  }

  return {
    getPluginPreset,
    createPluginPreset,
    updatePluginPreset,
    deletePluginPreset,
  }
}

export default usePluginPresetApi

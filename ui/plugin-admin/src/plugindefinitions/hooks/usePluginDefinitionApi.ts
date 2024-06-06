/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { PluginDefinition } from "../../../../types/types"
import useApi from "./useApi"

export type PluginDefinitionApiResponse = {
  ok: boolean
  message: string
  response?: PluginDefinition
}

export const usePluginDefinitionApi = () => {
  const { get, create, update, deleteObject } = useApi()

  const getPluginDefinition = (
    pluginDefinition: PluginDefinition
  ): Promise<PluginDefinitionApiResponse> => {
    return get<PluginDefinition>(
      `/apis/greenhouse.sap/v1alpha1/plugindefinitions`,
      pluginDefinition
    ) as Promise<PluginDefinitionApiResponse>
  }

  const createPluginDefinition = (
    pluginDefinition: PluginDefinition
  ): Promise<PluginDefinitionApiResponse> => {
    return create<PluginDefinition>(
      `/apis/greenhouse.sap/v1alpha1/plugindefinitions`,
      pluginDefinition
    ) as Promise<PluginDefinitionApiResponse>
  }

  const updatePluginDefinition = (
    pluginDefinition: PluginDefinition
  ): Promise<PluginDefinitionApiResponse> => {
    return update<PluginDefinition>(
      `/apis/greenhouse.sap/v1alpha1/plugindefinitions`,
      pluginDefinition
    ) as Promise<PluginDefinitionApiResponse>
  }

  const deletePluginDefinition = (
    pluginDefinition: PluginDefinition
  ): Promise<PluginDefinitionApiResponse> => {
    return deleteObject<PluginDefinition>(
      `/apis/greenhouse.sap/v1alpha1/plugindefinitions`,
      pluginDefinition
    ) as Promise<PluginDefinitionApiResponse>
  }

  return {
    getPluginDefinition,
    createPluginDefinition,
    updatePluginDefinition,
    deletePluginDefinition,
  }
}

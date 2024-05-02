/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { create } from "zustand"
import {
  PluginDefinition,
  Plugin,
  UpdateObjectAction,
  Secret,
  UpdatePluginInput as UpdatePluginDefinitionInput
} from "../../../types/types"

export interface State {
  endpoint: string
  setEndpoint: (newEndpoint: string) => void
  urlStateKey: string
  setUrlStateKey: (newUrlStateKey: string) => void

  auth: any
  setAuth: (auth: any) => void
  loggedIn: boolean
  setLoggedIn: (loggedIn: boolean) => void
  logout: any

  pluginDefinitions: PluginDefinition[]
  updatePluginDefinitions: (input: UpdatePluginDefinitionInput) => void
  showPluginDefinitionDetails: boolean
  setShowPluginDefinitionDetails: (showPluginDefinitionDetails: boolean) => void
  pluginDefinitionDetail: PluginDefinition | null
  setPluginDefinitionDetail: (pluginDefinition: PluginDefinition) => void
  showPluginDefinitionEdit: boolean
  setShowPluginEdit: (showPluginDefinitionEdit: boolean) => void
  pluginToEdit?: Plugin
  setPluginToEdit: (plugin?: Plugin) => void
  secretsToEdit?: Secret[]
  setSecretsToEdit: (secrets?: Secret[]) => void
  isEditMode: boolean
  setIsEditMode: (isEditMode: boolean) => void
}

// global zustand store. See how this works here: https://github.com/pmndrs/zustand
const usePluginDefinitionsStore = create<State>((set) => ({
  endpoint: "",
  setEndpoint: (newEndpoint) => set((state) => ({ endpoint: newEndpoint })),
  urlStateKey: "",
  setUrlStateKey: (newUrlStateKey) =>
    set((state) => ({ urlStateKey: newUrlStateKey })),

  auth: null,
  setAuth: (auth) => set((state) => ({ auth: auth })),
  loggedIn: false,
  setLoggedIn: (loggedIn) => set((state) => ({ loggedIn: loggedIn })),
  logout: null,

  pluginDefinitions: [],
  updatePluginDefinitions: (input: UpdatePluginDefinitionInput) =>
    set((state) => {
      console.log("updatePluginDefinitions", input)
      let pluginDefinitions = [...state.pluginDefinitions]
      // validate plugins: only accept input.plugins that have metadata.name set
      input.pluginDefinitions = input.pluginDefinitions.filter((pluginDefinition) => {
        return pluginDefinition.metadata?.name ?? undefined !== undefined
      })

      if (input.action === UpdateObjectAction.delete) {
        pluginDefinitions = pluginDefinitions.filter((knownPluginDefinition) => {
          return input.pluginDefinitions.some((inputPluginDefinition) => {
            return knownPluginDefinition.metadata!.name !== inputPluginDefinition.metadata!.name
          })
        })
        return { ...state, pluginDefinitions: pluginDefinitions }
      }

      input.pluginDefinitions.forEach((inputPluginDefinition) => {
        const index = pluginDefinitions.findIndex((knownPluginDefinition) => {
          return knownPluginDefinition.metadata!.name === inputPluginDefinition.metadata!.name
        })
        if (index >= 0) {
          pluginDefinitions[index] = inputPluginDefinition
        } else {
          pluginDefinitions.push(inputPluginDefinition)
        }
      })
      return { ...state, pluginDefinitions: pluginDefinitions }
    }),
  showPluginDefinitionDetails: false,
  setShowPluginDefinitionDetails: (showPluginDefinitionDetails) =>
    set((state) => ({ ...state, showPluginDefinitionDetails: showPluginDefinitionDetails })),

  pluginDefinitionDetail: null,
  setPluginDefinitionDetail: (pluginDefinition) => set((state) => ({ pluginDefinitionDetail: pluginDefinition  })),

  showPluginDefinitionEdit: false,
  setShowPluginEdit: (showPluginDefinitionEdit) =>
    set((state) => ({ showPluginDefinitionEdit: showPluginDefinitionEdit})),

  pluginToEdit: undefined,
  setPluginToEdit: (plugin) => set((state) => ({ pluginToEdit: plugin})),

  secretsToEdit: undefined,
  setSecretsToEdit: (secrets) => set((state) => ({ secretsToEdit: secrets})),

  isEditMode: false,
  setIsEditMode: (isEditMode) => set((state) => ({ isEditMode: isEditMode}))

}))

export default usePluginDefinitionsStore

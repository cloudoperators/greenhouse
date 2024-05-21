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
  UpdatePluginDefinitionInput,
  UpdateSecretInput,
  LabelSelector,
} from "../../../types/types"

export type EditFormData = {
  metadata?: Plugin["metadata"]
  spec?: Plugin["spec"]
  labelSelector?: LabelSelector
}

export enum EditFormState {
  "PLUGIN_CREATE",
  "PLUGIN_EDIT",
  "PLUGIN_PRESET_CREATE",
  "PLUGIN_PRESET_EDIT",
}

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
  showEditForm: boolean
  setShowEditForm: (showEditForm: boolean) => void
  pluginToEdit?: Plugin
  setPluginToEdit: (plugin?: Plugin) => void

  editFormData: EditFormData
  setEditFormData: (editFormData: EditFormData) => void
  editFormState: EditFormState
  setEditFormState: (editFormState: EditFormState) => void
  isFormEditMode: boolean
  setIsFormEditMode: (isEditMode: boolean) => void
  isFormPluginPresetMode: boolean
  setIsFormPluginPresetMode: (isEditMode: boolean) => void

  isPluginEditMode: boolean
  setIsPluginEditMode: (isEditMode: boolean) => void
  secrets: Secret[]
  updateSecrets: (input: UpdateSecretInput) => void
  secretDetail?: Secret
  setSecretDetail: (secret?: Secret) => void
  showSecretEdit: boolean
  setShowSecretEdit: (showSecretEdit: boolean) => void
  isSecretEditMode: boolean
  setIsSecretEditMode: (isEditMode: boolean) => void
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
      input.pluginDefinitions = input.pluginDefinitions.filter(
        (pluginDefinition) => {
          return pluginDefinition.metadata?.name ?? undefined !== undefined
        }
      )

      if (input.action === UpdateObjectAction.delete) {
        pluginDefinitions = pluginDefinitions.filter(
          (knownPluginDefinition) => {
            return input.pluginDefinitions.some((inputPluginDefinition) => {
              return (
                knownPluginDefinition.metadata!.name !==
                inputPluginDefinition.metadata!.name
              )
            })
          }
        )
        return { ...state, pluginDefinitions: pluginDefinitions }
      }

      input.pluginDefinitions.forEach((inputPluginDefinition) => {
        const index = pluginDefinitions.findIndex((knownPluginDefinition) => {
          return (
            knownPluginDefinition.metadata!.name ===
            inputPluginDefinition.metadata!.name
          )
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
    set((state) => ({
      ...state,
      showPluginDefinitionDetails: showPluginDefinitionDetails,
    })),

  pluginDefinitionDetail: null,
  setPluginDefinitionDetail: (pluginDefinition) =>
    set((state) => ({ pluginDefinitionDetail: pluginDefinition })),

  showEditForm: false,
  setShowEditForm: (showEditForm) =>
    set((state) => ({ showEditForm: showEditForm })),

  editFormState: EditFormState.PLUGIN_CREATE,
  setEditFormState: (editFormState) =>
    set((state) => ({ editFormState: editFormState })),

  pluginToEdit: undefined,
  setPluginToEdit: (plugin) => set((state) => ({ pluginToEdit: plugin })),

  editFormData: {
    metadata: undefined,
    spec: undefined,
    labelSelector: undefined,
  },
  setEditFormData: (editFormData) =>
    set((state) => ({ editFormData: editFormData })),

  isFormEditMode: false,
  setIsFormEditMode: (isEditMode) =>
    set((state) => ({ isFormEditMode: isEditMode })),

  isFormPluginPresetMode: false,
  setIsFormPluginPresetMode: (isEditMode) =>
    set((state) => ({ isFormPluginPresetMode: isEditMode })),

  isPluginEditMode: false,
  setIsPluginEditMode: (isEditMode) =>
    set((state) => ({ isPluginEditMode: isEditMode })),

  secrets: [],
  updateSecrets: (input: UpdateSecretInput) =>
    set((state) => {
      let secrets = [...state.secrets]

      if (input.action === UpdateObjectAction.delete) {
        secrets = secrets.filter((knownSecret) => {
          return input.secrets.some((inputSecret) => {
            return knownSecret.metadata!.name !== inputSecret.metadata!.name
          })
        })
        return { ...state, secrets: secrets }
      }

      input.secrets.forEach((inputSecret) => {
        const index = secrets.findIndex((knownSecret) => {
          return knownSecret.metadata!.name === inputSecret.metadata!.name
        })
        if (index >= 0) {
          secrets[index] = inputSecret
        } else {
          secrets.push(inputSecret)
        }
      })
      return { ...state, secrets: secrets }
    }),

  secretDetail: undefined,
  setSecretDetail: (secret) => set((state) => ({ secretDetail: secret })),
  showSecretEdit: false,
  setShowSecretEdit: (showSecretEdit) =>
    set((state) => ({ showSecretEdit: showSecretEdit })),

  isSecretEditMode: false,
  setIsSecretEditMode: (isEditMode) =>
    set((state) => ({ isSecretEditMode: isEditMode })),
}))

export default usePluginDefinitionsStore

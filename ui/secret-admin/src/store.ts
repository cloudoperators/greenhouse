/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { create } from "zustand"
import {
  Secret,
  UpdateObjectAction,
  UpdateSecretInput,
} from "../../types/types"

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
const useStore = create<State>((set) => ({
  endpoint: "",
  setEndpoint: (newEndpoint) => set((state) => ({ endpoint: newEndpoint })),
  urlStateKey: "secret-admin",
  setUrlStateKey: (newUrlStateKey) =>
    set((state) => ({ urlStateKey: newUrlStateKey })),

  auth: null,
  setAuth: (auth) => set((state) => ({ auth: auth })),
  loggedIn: false,
  setLoggedIn: (loggedIn) => set((state) => ({ loggedIn: loggedIn })),
  logout: null,

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

export default useStore

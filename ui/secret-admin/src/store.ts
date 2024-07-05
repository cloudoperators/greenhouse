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
  deleteSecrets: (secrets: Secret[]) => void
  modifySecrets: (secrets: Secret[]) => void
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
  modifySecrets: (secrets) =>
    set((state) => {
      let newSecrets = [...state.secrets]
      secrets.forEach((inputSecret) => {
        const index = newSecrets.findIndex((knownSecret) => {
          return knownSecret.metadata!.name === inputSecret.metadata!.name
        })
        if (index >= 0) {
          newSecrets[index] = inputSecret
        } else {
          newSecrets.push(inputSecret)
        }
      })
      return { ...state, secrets: newSecrets }
    }),
  deleteSecrets: (secrets) =>
    set((state) => {
      const newSecrets = state.secrets.filter((knownSecret) => {
        return !secrets.some((inputSecret) => {
          return knownSecret.metadata!.name === inputSecret.metadata!.name
        })
      })
      return { secrets: newSecrets }
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

const updateSecrets = (
  existingSecrets: Secret[],
  newSecrets: Secret[]
): Secret[] => {
  let returnSecrets = existingSecrets
  newSecrets.forEach((inputSecret) => {
    const index = existingSecrets.findIndex((knownSecret) => {
      return knownSecret.metadata!.name === inputSecret.metadata!.name
    })
    if (index >= 0) {
      returnSecrets[index] = inputSecret
    } else {
      returnSecrets.push(inputSecret)
    }
  })
  return returnSecrets
}

export default useStore

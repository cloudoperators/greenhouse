/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Secret } from "../../../types/types"

export const initSecret = (): Secret => {
  return {
    kind: "Secret",
    metadata: {
      name: "",
      namespace: "",
    },
    data: {},
  }
}

export const base64EncodeSecretData = (secret: Secret): Secret => {
  if (secret.data) {
    let data = {}
    Object.keys(secret.data).forEach((key) => {
      data[key] = btoa(secret.data![key])
    })
    secret.data = data
  }
  return secret
}

export const base64DecodeSecretData = (secret: Secret): Secret => {
  if (secret.data) {
    let data = {}
    Object.keys(secret.data).forEach((key) => {
      data[key] = atob(secret.data![key])
    })
    secret.data = data
  }
  return secret
}

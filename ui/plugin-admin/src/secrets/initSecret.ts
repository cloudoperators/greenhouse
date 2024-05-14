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

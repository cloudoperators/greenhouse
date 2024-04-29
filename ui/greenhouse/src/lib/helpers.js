/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

export const parseError = (error) => {
  let errMsg = JSON.stringify(error)
  if (error?.message) {
    errMsg = error?.message
    try {
      errMsg = JSON.parse(error?.message).msg
    } catch (error) {}
  }
  return errMsg
}

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import useUrlState from "../hooks/useUrlState"
import useCommunication from "../hooks/useCommunication"

const AsyncWorker = () => {
  useUrlState()
  useCommunication()
  return null
}

export default AsyncWorker

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import useCommunication from "../hooks/useCommunication"
import useUrlState from "../hooks/useUrlState"

const AsyncWorker = () => {
  useCommunication()
  useUrlState()
  return null
}

export default AsyncWorker

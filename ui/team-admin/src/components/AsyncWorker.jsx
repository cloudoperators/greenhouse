/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import useUrlState from "../hooks/useUrlState"

const AsyncWorker = ({ consumerId }) => {
  useUrlState(consumerId)
  return null
}

export default AsyncWorker

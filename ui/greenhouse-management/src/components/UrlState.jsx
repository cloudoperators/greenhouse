/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import HintLoading from "./shared/HintLoading"
import { useIsUrlStateSetup } from "./StoreProvider"

const UrlState = ({ children }) => {
  const isUrlStateSetup = useIsUrlStateSetup()

  return (
    <>
      {isUrlStateSetup ? children : <HintLoading text="Loading..." centered />}
    </>
  )
}

export default UrlState

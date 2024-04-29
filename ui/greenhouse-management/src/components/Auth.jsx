/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { useIsLoggedIn } from "./StoreProvider"
import HintLoading from "./shared/HintLoading"

// Adds a loading screen while during auth
// Shows children when auth is complete
const Auth = ({ children }) => {
  const authLoggedIn = useIsLoggedIn()

  return (
    <>
      {!!authLoggedIn ? (
        children
      ) : (
        <HintLoading text="Logging you in..." centered />
      )}
    </>
  )
}

export default Auth

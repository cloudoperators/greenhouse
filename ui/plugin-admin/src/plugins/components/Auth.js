/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { useLoggedIn } from "./StoreProvider"

// Adds a loading screen while during auth
// Shows children when auth is complete

const Auth = ({ children }) => {
  const loggedIn = useLoggedIn()

  return (
    <>
      {!!loggedIn && children}
      {!loggedIn && null}
    </>
  )
}

export default Auth

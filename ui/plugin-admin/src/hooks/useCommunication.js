/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect } from "react"
import { useGlobalsActions } from "../components/StoreProvider"

import { get, watch } from "communicator"

const useCommunication = () => {
  const { setAuthData, setLoggedIn } = useGlobalsActions()
  useEffect(() => {
    // get manually the current auth object in case the this app mist the first auth update message
    // this is the case this app is loaded after the Auth app.
    get(
      "AUTH_GET_DATA",
      (data) => {
        setAuthData(data.auth)
        setLoggedIn(data.loggedIn)
      },
      { debug: true }
    )
    // watch for auth updates messages
    // with the watcher we get the auth object when this app is loaded before the Auth app
    const unwatch = watch(
      "AUTH_UPDATE_DATA",
      (data) => {
        setAuthData(data.auth)
        setLoggedIn(data.loggedIn)
      },
      { debug: true }
    )
    return unwatch
  }, [setAuthData, setLoggedIn])
}

export default useCommunication

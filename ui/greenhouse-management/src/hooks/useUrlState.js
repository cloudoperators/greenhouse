/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect, useLayoutEffect } from "react"
import { registerConsumer } from "url-state-provider"
import {
  useActions,
  useIsUrlStateSetup,
  usePluginActive,
  useIsLoggedIn,
} from "../components/StoreProvider"

// url state manager
const URL_APP_STATE_KEY = "greenhouse-management"
const ACTIVE_APP_KEY = "a"
const urlStateManager = registerConsumer(URL_APP_STATE_KEY)

const useUrlState = () => {
  const { setPluginActive, setIsUrlStateSetup } = useActions()
  const isUrlStateSetup = useIsUrlStateSetup()
  const pluginActive = usePluginActive()
  const isLoggedIn = useIsLoggedIn()

  // Initial state from URL AFTER
  // WARNING. To get the right state from the URL do following:
  // If this app is embbeded in another app with authentication
  //  - Mount this app after the login is success in the parent app
  // or
  //  - Wait here until you get logged in
  useLayoutEffect(() => {
    if (!isLoggedIn || isUrlStateSetup) return

    let active = urlStateManager.currentState()?.[ACTIVE_APP_KEY]
    if (active) setPluginActive(active)
    setIsUrlStateSetup(true)
  }, [isUrlStateSetup, isLoggedIn])

  // sync URL state
  useEffect(() => {
    if (!isUrlStateSetup) return

    // if the current state is the same as the new state, don't push
    // this prevents the history from being filled with the same state
    // and therefore prevents the forward button from being disabled
    // This small optimization allows the user to go back and forth!
    if (urlStateManager.currentState()?.[ACTIVE_APP_KEY] === pluginActive)
      return

    urlStateManager.push({ [ACTIVE_APP_KEY]: pluginActive })
  }, [isUrlStateSetup, pluginActive])
}

export default useUrlState

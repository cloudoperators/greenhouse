/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useState, useEffect } from "react"
import { registerConsumer } from "url-state-provider"
import {
  useLoggedIn,
  useCurrentTeam,
  useDefaultTeam,
  useStoreActions,
} from "../components/StoreProvider"

const URL_STATE_KEY = "greenhouse-team-admin"
const TEAM_NAME = "team"

const useUrlState = (key) => {
  const [isURLRead, setIsURLRead] = useState(false)
  // it is possible to have two apps instances on the same page
  // int his case the key should be different per app
  const urlStateManager = registerConsumer(key || URL_STATE_KEY)

  // read variables from store
  const loggedIn = useLoggedIn()
  const { setCurrentTeam } = useStoreActions()
  const currentTeam = useCurrentTeam()
  const defaultTeam = useDefaultTeam()

  // Set initial state from URL (on login)
  useEffect(() => {
    /* !!!IMPORTANT!!!
      don't read the url if we are already read it or if we are not logged in!!!!!
    */
    if (isURLRead || !loggedIn) return
    console.log(
      `Team-Admin: (${key || URL_STATE_KEY}) setting up state from url:`,
      urlStateManager.currentState()
    )

    // READ the url state and set the state
    const newTeamName = urlStateManager.currentState()?.[TEAM_NAME]

    // SAVE the state
    if (newTeamName) setCurrentTeam(newTeamName)
    setIsURLRead(true)
  }, [loggedIn, setCurrentTeam])

  // SYNC states to URL state
  useEffect(() => {
    // don't sync if we are not logged in OR URL ist not yet read
    if (!isURLRead) return
    urlStateManager.push({
      [TEAM_NAME]: currentTeam || defaultTeam,
    })
    console.log(
      "useUrlStateKey - after  urlStateManager.push: ",
      urlStateManager.currentState()?.[TEAM_NAME]
    )
  }, [isURLRead, currentTeam, defaultTeam])
}

export default useUrlState

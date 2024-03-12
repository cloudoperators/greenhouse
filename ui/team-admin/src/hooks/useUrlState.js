/*
 * Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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

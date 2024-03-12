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

import createStore from "./store"

describe("Zustand Store", () => {
  let useStore

  beforeEach(() => {
    // Reset the store before each test to isolate tests from each other
    useStore = createStore()
  })

  test("initial state", () => {
    // Test initial state values
    expect(useStore.getState().endpoint).toEqual("")
    expect(useStore.getState().loggedIn).toEqual(false)
    expect(useStore.getState().currentTeam).toEqual("")
    expect(useStore.getState().defaultTeam).toEqual("")
    expect(useStore.getState().teamMemberships).toEqual([])
    expect(useStore.getState().namespace).toEqual("")
    expect(useStore.getState().isMock).toEqual(false)
  })

  test("setEndpoint action", () => {
    useStore.getState().actions.setEndpoint("example.com")

    expect(useStore.getState().endpoint).toEqual("example.com")
  })

  test("setAuthData action", () => {
    useStore.getState().actions.setAuthData({ auth: {}, loggedIn: true })

    expect(useStore.getState().auth).toEqual({})
    expect(useStore.getState().loggedIn).toEqual(true)
  })

  test("setCurrentTeam action", () => {
    useStore.getState().actions.setCurrentTeam("team1")

    expect(useStore.getState().currentTeam).toEqual("team1")
  })

  test("setDefaultTeam action", () => {
    useStore.getState().auth = { parsed: { teams: ["team2"] } }
    useStore.getState().teamMemberships = [
      { metadata: { name: "team1" } },
      { metadata: { name: "team2" } },
    ]

    useStore.getState().actions.setDefaultTeam()

    expect(useStore.getState().defaultTeam).toEqual("team2")
  })

  test("setTeamMemberships action", () => {
    const teamMemberships = [{ name: "team1" }, { name: "team2" }]
    useStore.getState().actions.setTeamMemberships(teamMemberships)

    expect(useStore.getState().teamMemberships).toEqual(teamMemberships)
  })

  test("setMock action", () => {
    useStore.getState().actions.setMock(true)

    expect(useStore.getState().isMock).toEqual(true)
  })
})

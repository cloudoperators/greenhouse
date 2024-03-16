/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
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

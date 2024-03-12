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

import { createStore } from "zustand"

// global zustand store. See how this works here: https://github.com/pmndrs/zustand
export default () =>
  createStore((set, get) => ({
    endpoint: "",
    loggedIn: false,
    currentTeam: "",
    defaultTeam: "",
    teamMemberships: [],
    namespace: "",
    isMock: false,
    actions: {
      setEndpoint: (newEndpoint) => set({ endpoint: newEndpoint }),
      setAuthData: (data) => {
        const { auth, loggedIn } = data
        const namespace = auth ? auth.parsed?.organizations || null : null
        set({
          auth,
          loggedIn: loggedIn,
          namespace,
        })
      },
      setCurrentTeam: (team) => set({ currentTeam: team }),
      setDefaultTeam: () =>
        set((state) => {
          const firstTeam = get().auth?.parsed?.teams?.[0]
          const defaultTeam =
            firstTeam ||
            (get().teamMemberships.length > 0
              ? get().teamMemberships[0].metadata.name
              : "")
          return { ...state, defaultTeam }
        }),
      setTeamMemberships: (teamMemberships) =>
        set({ teamMemberships: teamMemberships }),
      setMock: (isMock) => set({ isMock: isMock }),
    },
  }))

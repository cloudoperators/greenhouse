/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { render } from "@testing-library/react"
import "@testing-library/jest-dom/extend-expect" // for better assertions
import StoreProvider, {
  useEndpoint,
  useIsMock,
  useNamespace,
  useAuth,
  useLoggedIn,
  useCurrentTeam,
  useDefaultTeam,
  useTeamMemberships,
  useStoreActions,
} from "./StoreProvider"

describe("StoreProvider", () => {
  test("provides store values correctly", () => {
    const TestComponent = () => {
      const endpoint = useEndpoint()
      const isMock = useIsMock()
      const namespace = useNamespace()
      const auth = useAuth()
      const loggedIn = useLoggedIn()
      const currentTeam = useCurrentTeam()
      const defaultTeam = useDefaultTeam()
      const teamMemberships = useTeamMemberships()
      const storeActions = useStoreActions()

      return (
        <div>
          <div data-testid="endpoint">{endpoint}</div>
          <div data-testid="isMock">{String(isMock)}</div>
          <div data-testid="namespace">{namespace}</div>
          <div data-testid="auth">{JSON.stringify(auth)}</div>
          <div data-testid="loggedIn">{String(loggedIn)}</div>
          <div data-testid="currentTeam">{currentTeam}</div>
          <div data-testid="defaultTeam">{defaultTeam}</div>
          <div data-testid="teamMemberships">
            {JSON.stringify(teamMemberships)}
          </div>
          <div data-testid="storeActions">{JSON.stringify(storeActions)}</div>
        </div>
      )
    }

    const { getByTestId } = render(
      <StoreProvider>
        <TestComponent />
      </StoreProvider>
    )

    // Assert on the rendered values
    expect(getByTestId("endpoint")).toBeInTheDocument()
    expect(getByTestId("isMock")).toBeInTheDocument()
    expect(getByTestId("namespace")).toBeInTheDocument()
    expect(getByTestId("auth")).toBeInTheDocument()
    expect(getByTestId("loggedIn")).toBeInTheDocument()
    expect(getByTestId("currentTeam")).toBeInTheDocument()
    expect(getByTestId("defaultTeam")).toBeInTheDocument()
    expect(getByTestId("teamMemberships")).toBeInTheDocument()
    expect(getByTestId("storeActions")).toBeInTheDocument()
  })
})

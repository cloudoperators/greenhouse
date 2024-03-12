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

import React from "react"
import { render } from "@testing-library/react"
import { screen } from "shadow-dom-testing-library"
import {
  useCurrentTeam,
  useDefaultTeam,
  useTeamMemberships,
} from "../StoreProvider"
import TeamList from "./TeamList"
import "../../../__mocks__/intersectionObserverMock"

jest.mock("../StoreProvider", () => ({
  useCurrentTeam: jest.fn(),
  useDefaultTeam: jest.fn(),
  useTeamMemberships: jest.fn(),
}))

describe("TeamList Component", () => {
  beforeEach(() => {
    useCurrentTeam.mockClear()
    useDefaultTeam.mockClear()
    useTeamMemberships.mockClear()
  })

  test("renders team members", () => {
    const mockTeamMembers = [
      {
        id: 1,
        firstName: "John",
        lastName: "Doe",
        email: "john.doe@example.com",
      },
      {
        id: 2,
        firstName: "Jane",
        lastName: "Smith",
        email: "jane.smith@example.com",
      },
    ]

    useCurrentTeam.mockReturnValue("someTeam")
    useDefaultTeam.mockReturnValue("someTeam")
    useTeamMemberships.mockReturnValue([
      { metadata: { name: "someTeam" }, spec: { members: mockTeamMembers } },
    ])

    jest.mock("utils", () => ({
      useEndlessScrollList: jest.fn(() => ({
        scrollListItems: mockTeamMembers,
        iterator: mockTeamMembers,
      })),
    }))

    render(<TeamList />)

    const johnElement = screen.getByText("John")
    const janeElement = screen.getByText("Jane")

    expect(johnElement).toBeInTheDocument()
    expect(janeElement).toBeInTheDocument()
  })

  test("renders no team members message when there are no team members", () => {
    useCurrentTeam.mockReturnValue("someTeam")
    useDefaultTeam.mockReturnValue("someTeam")
    useTeamMemberships.mockReturnValue([])

    render(<TeamList />)

    const noMembersText = screen.getByText(
      "There are no Team Members to display."
    )
    expect(noMembersText).toBeInTheDocument()
  })
})

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { render, act, renderHook } from "@testing-library/react"
// support shadow dom queries
// https://reactjsexample.com/an-extension-of-dom-testing-library-to-provide-hooks-into-the-shadow-dom/
import { screen } from "shadow-dom-testing-library"
import ShellLayout from "./ShellLayout"
import StoreProvider from "../StoreProvider"
import { MessagesProvider } from "messages-provider"

jest.mock("communicator")

test("renders app", async () => {
  await act(() =>
    render(
      <MessagesProvider>
        <StoreProvider>
          <ShellLayout />
        </StoreProvider>
      </MessagesProvider>
    )
  )

  let logoTitle = await screen.queryAllByShadowTitle(/Greenhouse/i)
  expect(logoTitle.length > 0).toBe(true)
})

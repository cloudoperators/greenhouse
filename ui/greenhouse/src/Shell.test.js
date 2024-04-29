/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { render, act } from "@testing-library/react"
// support shadow dom queries
// https://reactjsexample.com/an-extension-of-dom-testing-library-to-provide-hooks-into-the-shadow-dom/
import { screen } from "shadow-dom-testing-library"
import Shell from "./Shell"
import Auth from "./components/Auth"
import StoreProvider from "./components/StoreProvider"

jest.mock("communicator")
jest.mock("./components/Auth")

test("renders app", async () => {
  await act(() =>
    render(
      <StoreProvider>
        <Shell />
      </StoreProvider>
    )
  )

  expect(Auth).toHaveBeenCalled()
})

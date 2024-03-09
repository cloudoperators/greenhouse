import React from "react"
import { render } from "@testing-library/react"
// support shadow dom queries
// https://reactjsexample.com/an-extension-of-dom-testing-library-to-provide-hooks-into-the-shadow-dom/
import { screen } from "shadow-dom-testing-library"
import App from "./App"

test("renders app", () => {
  render(<App />)
  const loginTitle = screen.queryAllByShadowText(/Converged Cloud/i)
  expect(loginTitle.length > 0).toBe(true)
})

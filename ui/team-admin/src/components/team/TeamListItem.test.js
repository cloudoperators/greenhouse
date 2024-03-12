import React from "react"
import { render } from "@testing-library/react"
import TeamListItem from "./TeamListItem"

const teamMember = {
  id: "1",
  firstName: "fn",
  lastName: "ln",
  email: "mail",
}

test("create teamMember for table items", () => {
  const { getByText } = render(<TeamListItem teamMember={teamMember} />)
  const id = getByText(/1/i)
  const labelElement = getByText(/fn/i)
  expect(id.textContent).toMatch(/1/i)
  expect(labelElement.textContent).toMatch(/fn/i)
})

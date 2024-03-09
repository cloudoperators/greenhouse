import React from "react"
import { DataGridRow, DataGridCell } from "juno-ui-components"

const TeamListItem = ({ teamMember }) => {
  return (
    <DataGridRow>
      <DataGridCell>{teamMember.id}</DataGridCell>
      <DataGridCell>{teamMember.firstName}</DataGridCell>
      <DataGridCell>{teamMember.lastName}</DataGridCell>
      <DataGridCell>{teamMember.email}</DataGridCell>
    </DataGridRow>
  )
}
export default TeamListItem

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

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

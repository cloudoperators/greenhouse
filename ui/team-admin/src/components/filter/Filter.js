/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect } from "react"
import { Stack, Select, SelectOption } from "juno-ui-components"
import {
  useAuth,
  useDefaultTeam,
  useCurrentTeam,
  useTeamMemberships,
  useStoreActions,
} from "../StoreProvider"

const filtersStyles = `
  bg-theme-background-lvl-1
  py-2
  px-4
  my-px`

const Filter = ({ onTeamChange }) => {
  const auth = useAuth()
  const { setDefaultTeam } = useStoreActions()
  const defaultTeam = useDefaultTeam()
  const currentTeam = useCurrentTeam()
  const teamMemberships = useTeamMemberships()

  useEffect(() => {
    setDefaultTeam()
  }, [auth, teamMemberships])

  return (
    <Stack direction="vertical" gap="4" className={`filters ${filtersStyles}`}>
      <Select
        name="team"
        className="filter-label-select w-64 mb-0"
        label="Team"
        value={currentTeam || defaultTeam}
        onChange={onTeamChange}
      >
        {teamMemberships?.map((teamData, index) => (
          <SelectOption
            value={teamData?.metadata?.name}
            label={teamData?.metadata?.name}
            key={index}
          />
        ))}
      </Select>
    </Stack>
  )
}

export default Filter

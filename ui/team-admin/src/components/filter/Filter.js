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

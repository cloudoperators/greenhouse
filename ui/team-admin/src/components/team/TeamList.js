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

import React, { useMemo } from "react"
import {
  DataGrid,
  DataGridRow,
  DataGridCell,
  DataGridHeadCell,
} from "juno-ui-components"
import TeamListItem from "./TeamListItem"
import {
  useCurrentTeam,
  useDefaultTeam,
  useTeamMemberships,
} from "../StoreProvider"
import { useEndlessScrollList } from "utils"

const TeamList = () => {
  const currentTeam = useCurrentTeam()
  const defaultTeam = useDefaultTeam()
  const teamMemberships = useTeamMemberships()

  const selectedTeam = currentTeam || defaultTeam

  const team = useMemo(() => {
    if (!selectedTeam || !teamMemberships) return []

    const matchingTeam = teamMemberships.find(
      (item) => item.metadata.name === selectedTeam
    )

    return matchingTeam?.spec?.members || []
  }, [selectedTeam, teamMemberships])

  const { scrollListItems, iterator } = useEndlessScrollList(team, {
    loadingObject: (
      <DataGridRow>
        <DataGridCell colSpan={4}>
          <span>Loading ...</span>
        </DataGridCell>
      </DataGridRow>
    ),
    refFunction: (ref) => (
      <DataGridRow>
        <DataGridCell colSpan={4} className="border-b-0 py-0">
          <span ref={ref} />
        </DataGridCell>
      </DataGridRow>
    ),
  })

  return scrollListItems?.length > 0 ? (
    <>
      <DataGrid columns={4}>
        <DataGridRow>
          <DataGridHeadCell>ID</DataGridHeadCell>
          <DataGridHeadCell>Firstname</DataGridHeadCell>
          <DataGridHeadCell>Lastname</DataGridHeadCell>
          <DataGridHeadCell>E-Mail</DataGridHeadCell>
        </DataGridRow>

        {iterator.map((member) => (
          <TeamListItem key={member.id} teamMember={member} />
        ))}
      </DataGrid>
    </>
  ) : (
    <div>There are no Team Members to display.</div>
  )
}

export default TeamList

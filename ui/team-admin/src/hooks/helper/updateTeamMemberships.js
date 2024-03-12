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

import isEqual from "lodash/isEqual"

function isIterable(input) {
  if (input === null || input === undefined) {
    return false
  }

  return typeof input[Symbol.iterator] === "function"
}

function updateTeamMemberships(
  teamMemberships,
  setTeamMemberships,
  { added, modified, deleted }
) {
  // Create a new array to hold the updated teamMemberships
  let updatedMemberships = [...teamMemberships]

  function findIndex(teamName) {
    return teamMemberships.findIndex(function (membership) {
      return teamName === membership.metadata.name
    })
  }

  if (added) {
    if (!isIterable(added)) return

    // If added is iterable, handle added objects
    Object.values(added)?.forEach((newTeamMembership) => {
      // Check if newTeamMembership already exists in teamMemberships
      const entryExists = updatedMemberships.some(
        (teamMembership) =>
          newTeamMembership?.metadata?.name === teamMembership?.metadata?.name
      )

      // If it doesn't exist, add it to the updatedMemberships array
      if (!entryExists) {
        updatedMemberships.push(newTeamMembership)
      }
    })
  }

  if (modified) {
    if (!isIterable(modified)) return

    modified.forEach((modifiedMembership) => {
      let index = findIndex(modifiedMembership?.metadata?.name)
      if (index !== -1) {
        if (!isEqual(updatedMemberships[index], modifiedMembership)) {
          // Check for actual changes
          updatedMemberships[index] = modifiedMembership
        }
      }
    })
  }

  if (deleted) {
    if (!isIterable(deleted)) return

    deleted.forEach((deletedMembership) => {
      let index = findIndex(deletedMembership?.metadata?.name)
      if (index !== -1) {
        updatedMemberships = updatedMemberships.filter((_, i) => i !== index) // Remove deleted membership
      }
    })
  }

  // Update teamMemberships using the setter function only if there are actual changes
  if (!isEqual(updatedMemberships, teamMemberships)) {
    setTeamMemberships(updatedMemberships)
  }
}

export default updateTeamMemberships

import React, { useEffect, useState } from "react"
import { Container, Stack } from "juno-ui-components"
import { useAPI } from "./hooks/useAPI"
import TeamList from "./components/team/TeamList"
import Filter from "./components/filter/Filter"
import {
  useTeamMemberships,
  useLoggedIn,
  useIsMock,
  useEndpoint,
  useStoreActions,
} from "./components/StoreProvider"
import { Messages, useActions } from "messages-provider"
import { fetchProxy } from "utils"
import { parseError } from "./lib/helpers"

const AppContent = () => {
  const teamMemberships = useTeamMemberships()
  const loggedIn = useLoggedIn()
  const isMock = useIsMock()
  const endpoint = useEndpoint()
  const { watchTeamMembers } = useAPI()
  const { setCurrentTeam, setTeamMemberships } = useStoreActions()
  const { addMessage } = useActions()

  useEffect(() => {
    if (!watchTeamMembers || isMock) return
    const unwatch = watchTeamMembers()
    return unwatch
  }, [watchTeamMembers])

  useEffect(() => {
    if (isMock) {
      fetchProxy(`${endpoint}/teammemberships`, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        ...{ mock: true },
      })
        .then((response) => {
          if (!response.ok) {
            addMessage({
              variant: "error",
              text: parseError(e.message),
            })
          }
          return response.json()
        })
        .then((result) => {
          setTeamMemberships(result)
        })
    }
  }, [isMock])

  function onTeamChange(value) {
    setCurrentTeam(value)
  }

  return (
    <Container py px>
      <Messages className="pb-6" />
      {loggedIn ? (
        teamMemberships?.length > 0 && (
          <>
            <Filter onTeamChange={onTeamChange} />
            <TeamList />
          </>
        )
      ) : (
        <Stack
          alignment="center"
          distribution="center"
          direction="vertical"
          className="h-full"
        >
          <span>
            You are not logged in! To access the application, you need to have
            the Authentication Application and log in.
          </span>
        </Stack>
      )}
    </Container>
  )
}

export default AppContent

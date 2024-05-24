/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { Container, Message, Stack } from "juno-ui-components"
import { useEffect, useState } from "react"
import SecretEdit from "./components/SecretEdit"
import SecretList from "./components/SecretList"

import WelcomeView from "./components/WelcomeView"
import useClient from "./hooks/useClient"
import useNamespace from "./hooks/useNamespace"
import useStore from "./store"
import ResultMessageComponent, {
  ResultMessage,
} from "./components/SubmitResultMessage"
import useCheckAuthorized from "./hooks/useIsAuthorized"

const AppContent = () => {
  const secrets = useStore((state) => state.secrets)
  const showSecretEdit = useStore((state) => state.showSecretEdit)

  const auth = useStore((state) => state.auth)
  const authError = auth?.error
  const loggedIn = useStore((state) => state.loggedIn)

  const { namespace } = useNamespace()
  const { client: client } = useClient()
  const { canListSecrets } = useCheckAuthorized()

  const [authMessage, setAuthMessage] = useState<ResultMessage>({
    ok: false,
    message: "",
  })

  useEffect(() => {
    canListSecrets().then((res) => {
      console.log("canListSecrets", res)
      setAuthMessage(res)
    })
  }, [client, namespace])

  return (
    <Container>
      {authMessage.message && (
        <Stack>
          <ResultMessageComponent
            submitMessage={authMessage}
          ></ResultMessageComponent>
        </Stack>
      )}

      {loggedIn && !authError ? (
        <>
          {secrets.length > 0 && <SecretList secrets={secrets} />}
          {showSecretEdit && <SecretEdit />}
        </>
      ) : (
        <WelcomeView />
      )}
    </Container>
  )
}

export default AppContent

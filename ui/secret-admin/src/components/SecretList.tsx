/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  DataGrid,
  DataGridHeadCell,
  DataGridRow,
  Container,
  Button,
  DataGridToolbar,
  ButtonRow,
} from "juno-ui-components"
import React, { useEffect, useState } from "react"
import { Secret } from "../../../types/types"
import SecretListItem from "./SecretListItem"
import useStore from "../store"
import { initSecret } from "./initSecret"
import useCheckAuthorized from "../hooks/useIsAuthorized"
import ResultMessageComponent, { ResultMessage } from "./SubmitResultMessage"

interface SecretListProps {
  secrets: Secret[]
}

const SecretList: React.FC<SecretListProps> = (props: SecretListProps) => {
  const setShowSecretEdit = useStore((state) => state.setShowSecretEdit)
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const openEditSecret = () => {
    setShowSecretEdit(true)
    setSecretDetail(initSecret())
  }
  const { canListSecrets } = useCheckAuthorized()
  const auth = useStore((state) => state.auth)

  const [authMessage, setAuthMessage] = useState<ResultMessage>({
    ok: false,
    message: "",
  })

  useEffect(() => {
    canListSecrets().then((res) => {
      setAuthMessage(res)
    })
  }, [auth])
  return (
    <>
      <Container>
        <DataGridToolbar>
          <ButtonRow>
            <Button
              icon="addCircle"
              label="Add Secret"
              onClick={openEditSecret}
            />
          </ButtonRow>
        </DataGridToolbar>
        <DataGrid columns={2} className="secrets">
          <DataGridRow>
            <DataGridHeadCell>Name</DataGridHeadCell>
            <DataGridHeadCell>Keys</DataGridHeadCell>
          </DataGridRow>

          {props.secrets.map((secret) => (
            <SecretListItem key={secret.metadata!.name!} secret={secret} />
          ))}
        </DataGrid>
        {authMessage.message && (
          <ResultMessageComponent
            submitMessage={authMessage}
          ></ResultMessageComponent>
        )}
      </Container>
    </>
  )
}

export default SecretList

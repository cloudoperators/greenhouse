/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  ButtonRow,
  Container,
  DataGrid,
  DataGridHeadCell,
  DataGridRow,
  DataGridToolbar,
} from "@cloudoperators/juno-ui-components"
import { Messages } from "@cloudoperators/juno-messages-provider"
import React from "react"
import useStore from "../store"
import SecretListItem from "./SecretListItem"
import { initSecret } from "./secretUtils"

const SecretList: React.FC = () => {
  const setShowSecretEdit = useStore((state) => state.setShowSecretEdit)
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const openEditSecret = () => {
    setShowSecretEdit(true)
    setSecretDetail(initSecret())
  }

  const secrets = useStore((state) => state.secrets)

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

          {secrets.map((secret) => (
            <SecretListItem key={secret.metadata!.name!} secret={secret} />
          ))}
        </DataGrid>
        <Messages />
      </Container>
    </>
  )
}

export default SecretList

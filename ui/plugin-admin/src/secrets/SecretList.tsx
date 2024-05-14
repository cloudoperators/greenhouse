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
import React from "react"
import { Secret } from "../../../types/types"
import SecretListItem from "./SecretListItem"
import useStore from "../plugindefinitions/store"
import { initSecret } from "./initSecret"

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
      </Container>
    </>
  )
}

export default SecretList

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { DataGridCell, DataGridRow } from "juno-ui-components"
import React from "react"
import { Secret } from "../../../types/types"
import useStore from "../store"
import { base64DecodeSecretData } from "./secretUtils"

interface SecretListItemProps {
  secret: Secret
}

const SecretListItem: React.FC<SecretListItemProps> = (
  props: SecretListItemProps
) => {
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const setShowSecretEdit = useStore((state) => state.setShowSecretEdit)
  const setIsSecretEditMode = useStore((state) => state.setIsSecretEditMode)
  const openDetails = () => {
    // base64 decode secret data before showing it
    let secretDetail = base64DecodeSecretData(props.secret)
    setSecretDetail(secretDetail)
    setIsSecretEditMode(true)
    setShowSecretEdit(true)
  }
  return (
    <DataGridRow className={`cursor-pointer`} onClick={openDetails}>
      <DataGridCell>{props.secret.metadata!.name}</DataGridCell>
      <DataGridCell>
        {props.secret.data
          ? Object.keys(props.secret.data).join(", ")
          : "No keys found"}
      </DataGridCell>
    </DataGridRow>
  )
}
export default SecretListItem

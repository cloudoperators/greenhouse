/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { Stack, Spinner } from "@cloudoperators/juno-ui-components"

const HintLoading = ({ text }) => {
  return (
    <Stack alignment="center">
      <Spinner variant="primary" />
      {text ? <span>{text}</span> : <span>Loading...</span>}
    </Stack>
  )
}

export default HintLoading

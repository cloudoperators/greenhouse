/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useMemo } from "react"
import { Stack, Spinner } from "juno-ui-components"

const centeredProps = {
  alignment: "center",
  distribution: "center",
  direction: "vertical",
  className: "h-full",
}

const HintLoading = ({ text, centered }) => {
  const stackProps = useMemo(() => {
    return centered ? centeredProps : {}
  }, [centered])

  return (
    <Stack {...stackProps}>
      <Stack alignment="center">
        <Spinner variant="primary" />
        {text ? <span>{text}</span> : <span>Loading...</span>}
      </Stack>
    </Stack>
  )
}

export default HintLoading

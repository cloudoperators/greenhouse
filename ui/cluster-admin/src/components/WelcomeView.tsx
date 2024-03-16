/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Button, Spinner, Stack } from "juno-ui-components"
import React from "react"
import useStore from "../store"

const WelcomeView: React.FC<any> = () => {
  const authIsProcessing = useStore((state) => state.auth?.isProcessing)
  return (
    <Stack
      alignment="center"
      distribution="center"
      direction="vertical"
      className="my-[10vh]"
    >
      <p className="text-xl">Welcome to the Cluster Administration</p>
      {authIsProcessing ? (
        <Spinner />
      ) : (
        <>
          <p className="text-xl">Reload to login</p>
        </>
      )}
    </Stack>
  )
}

export default WelcomeView

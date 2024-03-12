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

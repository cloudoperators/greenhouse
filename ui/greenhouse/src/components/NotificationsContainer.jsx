/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { Messages } from "messages-provider"
import { useDemoMode } from "./StoreProvider"

const NotificationsContainer = () => {
  const demoMode = useDemoMode()

  return (
    <>
      {demoMode && (
        <div className="bg-theme-accent/30 py-2 px-4">
          Welcome to the Greenhouse demo system! We're glad you're here! Just a
          quick heads up: you won't find any live data here. Enjoy exploring!
        </div>
      )}
      {/* do not use a container here to align the messages to the ones coming from each plugin */}
      <Messages className="mx-4 mt-4" />
    </>
  )
}

export default NotificationsContainer

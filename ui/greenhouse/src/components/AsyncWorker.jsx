/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect } from "react"
import useUrlState from "../hooks/useUrlState"
import useCommunication from "../hooks/useCommunication"
import { useAuthData, useAuthLoggedIn } from "../components/StoreProvider"

const currentUrl = new URL(window.location.href)
let match = currentUrl.host.match(/^(.+)\.dashboard\..+/)
let orgName = match ? match[1] : currentUrl.searchParams.get("org")

const AsyncWorker = () => {
  const authData = useAuthData()
  const authLoggedIn = useAuthLoggedIn()

  useCommunication()
  useUrlState()

  // read org name from token and adjust url to contain the org name
  useEffect(() => {
    if (!authLoggedIn) return

    if (!orgName) {
      const orgString = authData?.raw?.groups?.find(
        (g) => g.indexOf("organization:") === 0
      )

      if (orgString) {
        const name = orgString.split(":")[1]
        let url = new URL(window.location.href)
        url.searchParams.set("org", name)
        window.history.replaceState(null, null, url.href)
      }
    }
  }, [authLoggedIn, authData])

  return null
}

export default AsyncWorker

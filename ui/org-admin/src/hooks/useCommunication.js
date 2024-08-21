/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect } from "react"
import { get, watch } from "@cloudoperators/juno-communicator"
import { useActions } from "../components/StoreProvider"

const useCommunication = () => {
  const { setAuthData: setAuthData } = useActions()
  const { setAuthAppLoaded: setAuthAppLoaded } = useActions()
  const CONSUMER_ID = "greenhouse-org-admin"

  useEffect(() => {
    if (!setAuthData || !setAuthAppLoaded) return
    get("AUTH_APP_LOADED", setAuthAppLoaded, {
      consumerID: CONSUMER_ID,
      debug: true,
    })
    const unwatchLoaded = watch("AUTH_APP_LOADED", setAuthAppLoaded, {
      debug: true,
      consumerID: CONSUMER_ID,
    })

    get("AUTH_GET_DATA", setAuthData, {
      consumerID: CONSUMER_ID,
      debug: true,
    })
    const unwatchUpdate = watch("AUTH_UPDATE_DATA", setAuthData, {
      debug: true,
      consumerID: CONSUMER_ID,
    })

    return () => {
      if (unwatchLoaded) unwatchLoaded()
      if (unwatchUpdate) unwatchUpdate()
    }
  }, [setAuthData, setAuthAppLoaded])
}

export default useCommunication

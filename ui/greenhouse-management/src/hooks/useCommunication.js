/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect } from "react"
import { get, watch } from "communicator"
import { useActions } from "../components/StoreProvider"

const useCommunication = () => {
  const { setAuthData: setAuthData } = useActions()
  const { setAuthAppLoaded: setAuthAppLoaded } = useActions()

  useEffect(() => {
    if (!setAuthData || !setAuthAppLoaded) return
    get("AUTH_APP_LOADED", setAuthAppLoaded, {
      consumerID: "greenhouse-management",
      debug: true,
    })
    const unwatchLoaded = watch("AUTH_APP_LOADED", setAuthAppLoaded, {
      debug: true,
      consumerID: "greenhouse-management",
    })

    get("AUTH_GET_DATA", setAuthData, {
      consumerID: "greenhouse-management",
      debug: true,
    })
    const unwatchUpdate = watch("AUTH_UPDATE_DATA", setAuthData, {
      debug: true,
      consumerID: "greenhouse-management",
    })

    return () => {
      if (unwatchLoaded) unwatchLoaded()
      if (unwatchUpdate) unwatchUpdate()
    }
  }, [setAuthData, setAuthAppLoaded])
}

export default useCommunication

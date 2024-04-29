/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect, useCallback } from "react"
import { broadcast, get, watch } from "communicator"
import {
  useAuthAppLoaded,
  useAuthIsProcessing,
  useAuthError,
  useAuthLoggedIn,
  useAuthLastAction,
  useAuthActions,
  useDemoMode,
  useDemoUserToken,
} from "../components/StoreProvider"

const useCommunication = () => {
  const authAppLoaded = useAuthAppLoaded()
  const authIsProcessing = useAuthIsProcessing()
  const authLastAction = useAuthLastAction()
  const { setData: authSetData, setAppLoaded: authSetAppLoaded } =
    useAuthActions()
  const demoMode = useDemoMode()
  const demoUserToken = useDemoUserToken()

  const setAuthData = useCallback(
    (data) => {
      // If we're in demo mode, we need to make sure the JWT is set to the demo user's JWT
      if (data?.auth && demoMode && demoUserToken) {
        data.auth.JWT = demoUserToken
      }
      if (data?.auth?.error)
        console.warn("Greenhouse: Auth error: ", data?.auth?.error)
      authSetData(data)
    },
    [authSetData, demoMode, demoUserToken]
  )

  useEffect(() => {
    if (!authAppLoaded || authIsProcessing) return
    if (authLastAction?.name === "signOn") {
      broadcast("AUTH_LOGIN", "greenhouse", {
        debug: true,
        consumerID: "greenhouse",
      })
    } else if (authLastAction?.name === "signOut") {
      broadcast("AUTH_LOGOUT", "greenhouse", {
        debug: true,
        consumerID: "greenhouse",
      })
    }
  }, [authAppLoaded, authIsProcessing, authLastAction])

  useEffect(() => {
    if (!authSetData || !authSetAppLoaded) return
    get("AUTH_APP_LOADED", authSetAppLoaded, {
      consumerID: "greenhouse",
      debug: true,
    })
    const unwatchLoaded = watch("AUTH_APP_LOADED", authSetAppLoaded, {
      debug: true,
      consumerID: "greenhouse",
    })

    get("AUTH_GET_DATA", setAuthData, { consumerID: "greenhouse", debug: true })
    const unwatchUpdate = watch("AUTH_UPDATE_DATA", setAuthData, {
      debug: true,
      consumerID: "greenhouse",
    })

    return () => {
      if (unwatchLoaded) unwatchLoaded()
      if (unwatchUpdate) unwatchUpdate()
    }
  }, [setAuthData, authSetAppLoaded])
}

export default useCommunication

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { broadcast, get, watch } from "communicator"
import { useCallback, useEffect } from "react"
import {
  useAuthActions,
  useAuthAppLoaded,
  useAuthIsProcessing,
  useAuthLastAction,
  useDemoMode,
  useDemoUserToken
} from "../components/StoreProvider"

const useCommunication = () => {
  const CONSUMER_ID = "greenhouse-dashboard"
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
        consumerID: CONSUMER_ID,
      })
    } else if (authLastAction?.name === "signOut") {
      broadcast("AUTH_LOGOUT", "greenhouse", {
        debug: true,
        consumerID: CONSUMER_ID,
      })
    }
  }, [authAppLoaded, authIsProcessing, authLastAction])

  useEffect(() => {
    if (!authSetData || !authSetAppLoaded) return
    get("AUTH_APP_LOADED", authSetAppLoaded, {
      consumerID: CONSUMER_ID,
      debug: true,
    })
    const unwatchLoaded = watch("AUTH_APP_LOADED", authSetAppLoaded, {
      debug: true,
      consumerID: CONSUMER_ID,
    })

    get("AUTH_GET_DATA", setAuthData, { consumerID: CONSUMER_ID, debug: true })
    const unwatchUpdate = watch("AUTH_UPDATE_DATA", setAuthData, {
      debug: true,
      consumerID: CONSUMER_ID,
    })

    return () => {
      if (unwatchLoaded) unwatchLoaded()
      if (unwatchUpdate) unwatchUpdate()
    }
  }, [setAuthData, authSetAppLoaded])
}

export default useCommunication

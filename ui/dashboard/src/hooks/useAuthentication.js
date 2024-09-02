import React, { useEffect, useMemo } from "react"
import { oidcSession, mockedSession } from "@cloudoperators/juno-oauth"
import { broadcast, watch, onGet } from "@cloudoperators/juno-communicator"

export const MOCKED_TOKEN = {
  iss: "https://auth.mock.greenhouse",
  sub: "3ksXP1FQq7j9125Q6ayY",
  aud: "greenhouse-dev-env",
  nonce: "MOCK",
  email: "jane.doe@sap.com",
  email_verified: true,
  groups: ["organization:test-org", "test-team-1"],
  name: "I123456",
  preferred_username: "Jane Doe",
}

const useAuthentication = ({
  issuerURL,
  clientID,
  mock,
  debug,
  requestParams,
  initialLogin,
}) => {
  const [authData, setAuthData] = React.useState()

  const oidc = useMemo(() => {
    if (mock === "true" || mock === true || typeof mock === "string") {
      let token

      try {
        token = JSON.parse(atob(mock))
      } catch (e) {
        try {
          token = JSON.parse(mock)
        } catch (e) {
          token = MOCKED_TOKEN
        }
      }

      return mockedSession({
        token,
        initialLogin: initialLogin,
        onUpdate: (authData) => {
          setAuthData(authData)
        },
      })
    }

    if (!clientID || !issuerURL) return

    return oidcSession({
      issuerURL,
      clientID,
      initialLogin: true,
      refresh: true,
      requestParams: requestParams,
      flowType: "code",
      onUpdate: (authData) => {
        setAuthData(authData)
      },
    })
  }, [setAuthData, clientID, issuerURL, mock, initialLogin, requestParams])

  const login = React.useCallback(oidc?.login, [oidc.login])
  const logout = React.useCallback(() => {
    if (!oidc) return
    oidc.logout({
      resetOIDCSession: true,
      silent: true,
    })
  }, [oidc.logout])

  useEffect(() => {
    // inform that the auth app has been loaded!
    broadcast("AUTH_APP_LOADED", true, { debug, consumerID: "auth" })
    onGet("AUTH_APP_LOADED", () => true, { debug, consumerID: "auth" })
  }, [])

  useEffect(() => {
    broadcast("AUTH_UPDATE_DATA", authData, { debug, consumerID: "auth" })
    const unwatchGet = onGet("AUTH_GET_DATA", () => authData, {
      debug,
      consumerID: "auth",
    })

    return unwatchGet
  }, [authData])

  useEffect(() => {
    const unwatchLogin = watch("AUTH_LOGIN", login, {
      debug,
      consumerID: "auth",
    })
    const unwatchLogout = watch("AUTH_LOGOUT", logout, {
      debug,
      consumerID: "auth",
    })
    // unregister on get listener when unmounting
    return () => {
      unwatchLogin()
      unwatchLogout()
    }
  }, [login, logout])
}

export { useAuthentication }

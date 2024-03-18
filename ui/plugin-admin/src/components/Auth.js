import React from "react"
import { useLoggedIn } from "./StoreProvider"

// Adds a loading screen while during auth
// Shows children when auth is complete

const Auth = ({ children }) => {
  const loggedIn = useLoggedIn()

  return (
    <>
      {!!loggedIn && children}
      {!loggedIn && null}
    </>
  )
}

export default Auth

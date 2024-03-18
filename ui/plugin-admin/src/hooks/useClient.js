import React, { useMemo } from "react"
import { createClient } from "sapcc-k8sclient"
import { useApiEndpoint } from "../components/StoreProvider"
import { useAuthData } from "../components/StoreProvider"

export const useClient = () => {
  const apiEndpoint = useApiEndpoint()
  const authData = useAuthData()

  const client = useMemo(() => {
    if (!apiEndpoint || !authData?.JWT) return null
    return createClient({ apiEndpoint, token: authData?.JWT })
  }, [apiEndpoint, authData?.JWT])

  return { client }
}
export default useClient

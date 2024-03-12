import { useMemo } from "react"
import { createClient } from "sapcc-k8sclient"
import { useEndpoint, useAuth } from "../components/StoreProvider"

export const useClient = () => {
  const apiEndpoint = useEndpoint()
  const authData = useAuth()

  const client = useMemo(() => {
    if (!apiEndpoint || !authData?.JWT) return null
    return createClient({ apiEndpoint, token: authData?.JWT })
  }, [apiEndpoint, authData?.JWT])

  return { client }
}
export default useClient

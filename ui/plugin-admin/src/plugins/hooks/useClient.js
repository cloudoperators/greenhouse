/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useMemo } from "react"
import { createClient } from "sapcc-k8sclient"
import { useApiEndpoint, useAuthData } from "../components/StoreProvider"

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

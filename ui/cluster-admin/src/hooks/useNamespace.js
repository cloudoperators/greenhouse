/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useMemo } from "react"
import useStore from "../store"

export const useNamespace = () => {
  const authData = useStore((state) => state.auth)

  const namespace = useMemo(() => {
    const orgString = authData?.parsed?.organizations
    if (!orgString) return null
    return orgString
  }, [authData?.parsed?.organizations])

  return { namespace }
}

export default useNamespace

/*
 * Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useEffect } from "react"
import { get, watch } from "communicator"
import { useStoreActions } from "../components/StoreProvider"

const useCommunication = () => {
  const { setAuthData } = useStoreActions()

  useEffect(() => {
    // get manually the current auth object in case the this app mist the first auth update message
    // this is the case this app is loaded after the Auth app.
    get(
      "AUTH_GET_DATA",
      (data) => {
        setAuthData(data)
      },
      { debug: true }
    )
    // watch for auth updates messages
    // with the watcher we get the auth object when this app is loaded before the Auth app
    const unwatch = watch(
      "AUTH_UPDATE_DATA",
      (data) => {
        setAuthData(data)
      },
      { debug: true }
    )
    return unwatch
  })
}

export default useCommunication

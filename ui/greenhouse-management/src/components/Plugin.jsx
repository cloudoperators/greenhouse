/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect, useMemo, useRef, useState } from "react"
import { useAppLoader } from "utils"
import { useAssetsUrl, usePluginActive } from "./StoreProvider"
import { Messages, useActions } from "messages-provider"
import { parseError } from "../lib/helpers"
import { Stack, Button } from "juno-ui-components"
import HintLoading from "./shared/HintLoading"

const Plugin = ({ config }) => {
  const { addMessage } = useActions()
  const assetsUrl = useAssetsUrl()
  const { mount } = useAppLoader(assetsUrl)
  const holder = useRef()
  const activePlugin = usePluginActive()

  // local state
  const [displayReload, setDisplayReload] = useState(false)
  const [reload, setReload] = useState(0)
  const [isMountedApp, setIsMountedApp] = useState(false)

  // element to mount the app
  const el = document.createElement("div")
  el.classList.add("inline")
  const app = useRef(el)

  // mount the app each time the component is reloaded losing the state
  useEffect(() => {
    if (!mount || !assetsUrl || !config) return
    // mount the app
    mount(app.current, config)
      .then((loaded) => {
        if (!loaded) return
        setIsMountedApp(true)
      })
      .catch((error) => {
        setDisplayReload(true)
        addMessage({
          variant: "error",
          text: `${config?.name}: ` + parseError(error),
        })
      })
  }, [mount, reload, config, assetsUrl])

  const displayPluging = useMemo(
    () => activePlugin === config?.name,
    [activePlugin, config]
  )

  useEffect(() => {
    // if assetsUrl still null when rendering for first time the component then mountApp also return null and we skip here
    if (!isMountedApp) return

    if (displayPluging) {
      // append to holder
      holder.current.appendChild(app.current)
    } else {
      // remove from holder
      if (holder.current.contains(app.current))
        holder.current.removeChild(app.current)
    }
  }, [isMountedApp, displayPluging])

  return (
    <div data-app={config?.name} ref={holder} className="inline">
      {displayPluging && (
        <>
          <Messages />
          {!isMountedApp && !displayReload && <HintLoading centered />}
          {displayReload && (
            <Stack
              alignment="center"
              distribution="center"
              direction="vertical"
              className="my-[10vh]"
            >
              <p className="text-xl">
                Uh-oh! Our plugin <b>{config?.label}</b> encountered a hiccup.{" "}
              </p>
              <p>
                No worries, just give it a little nudge by clicking the{" "}
                <strong>Reload</strong> button below.
              </p>
              <Button
                label="Reload"
                variant="primary"
                onClick={() => setReload(reload + 1)}
                className="mt-2"
              />
            </Stack>
          )}
        </>
      )}
    </div>
  )
}

export default Plugin

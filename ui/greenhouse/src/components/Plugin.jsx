/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect, useState, useMemo, useRef } from "react"
import { useAppLoader } from "utils"
import { usePlugin, useGlobalsAssetsHost } from "../components/StoreProvider"
import { Messages, useActions } from "messages-provider"
import { parseError } from "../lib/helpers"
import { Stack, Button } from "juno-ui-components"

const Plugin = ({ id }) => {
  const assetsHost = useGlobalsAssetsHost()
  const { mount } = useAppLoader(assetsHost)
  const holder = useRef()
  const config = usePlugin().config()
  const activeApps = usePlugin().active()
  const { addMessage } = useActions()

  const [displayReload, setDisplayReload] = useState(false)
  const [reload, setReload] = useState(0)
  const [isMounted, setIsMounted] = useState(false)

  // element to mount the app
  const el = document.createElement("div")
  el.classList.add("inline")
  const app = useRef(el)

  // mount the app each time the component is reloaded losing the state
  useEffect(() => {
    if (!mount || !assetsHost || !config) return
    // mount the app
    mount(app.current, {
      ...config[id],
      props: { ...config[id]?.props, embedded: true },
    })
      .then((loaded) => {
        if (!loaded) return
        setIsMounted(true)
      })
      .catch((error) => {
        setDisplayReload(true)
        addMessage({
          variant: "error",
          text: `${config?.name}: ` + parseError(error),
        })
      })
  }, [mount, reload, config, assetsHost])

  const displayPluging = useMemo(
    () => activeApps.indexOf(id) >= 0,
    [activeApps, config]
  )

  useEffect(() => {
    if (!config[id] || !isMounted) return

    if (displayPluging) {
      //  add to holder
      holder.current.appendChild(app.current)
    } else {
      // remove from holder
      if (holder.current.contains(app.current))
        holder.current.removeChild(app.current)
    }
  }, [isMounted, displayPluging])

  return (
    <div data-app={id} ref={holder} className="inline">
      {displayPluging && (
        <>
          <Messages className="mr-4" />
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

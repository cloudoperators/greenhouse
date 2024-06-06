/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import {
  SideNavigation,
  SideNavigationItem,
  Badge,
  Stack,
} from "juno-ui-components"
import { usePluginConfig, usePluginActive, useActions } from "./StoreProvider"

const SideNav = () => {
  const pluginConfig = usePluginConfig()
  const pluginActive = usePluginActive()
  const { setPluginActive } = useActions()

  return (
    <SideNavigation>
      {Object.keys(pluginConfig).map((key, index) => (
        <SideNavigationItem
          key={key}
          active={pluginConfig[key]?.name === pluginActive}
          onClick={() => setPluginActive(pluginConfig[key]?.name)}
        >
          <Stack>
            {pluginConfig[key]?.label}
            {pluginConfig[key]?.releaseState && (
              <Badge
                className="ml-1"
                text={pluginConfig[key]?.releaseState}
                variant="info"
              />
            )}
          </Stack>
        </SideNavigationItem>
      ))}
    </SideNavigation>
  )
}

export default SideNav

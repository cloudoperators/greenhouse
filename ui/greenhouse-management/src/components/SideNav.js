/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { SideNavigation, SideNavigationItem } from "juno-ui-components"
import { usePluginConfig, usePluginActive, useActions } from "./StoreProvider"

const SideNav = () => {
  const pluginConfig = usePluginConfig()
  const pluginActive = usePluginActive()
  const { setPluginActive } = useActions()

  return (
    <SideNavigation>
      {Object.keys(pluginConfig).map((key, index) => (
        <SideNavigationItem
          key={index}
          label={pluginConfig[key]?.label}
          active={pluginConfig[key]?.name === pluginActive}
          onClick={() => setPluginActive(pluginConfig[key]?.name)}
        />
      ))}
    </SideNavigation>
  )
}

export default SideNav

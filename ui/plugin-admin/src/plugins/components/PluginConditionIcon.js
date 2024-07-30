/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { Icon } from "@cloudoperators/juno-ui-components"

import { getResourceStatusFromKubernetesConditions } from "../../../../utils/resourceStatus"

// PluginConditionIcon renders an icon based on the plugin status
export const PluginConditionIcon = ({ plugin }) => {
  const readyStatus = plugin?.status?.statusConditions?.conditions
    ? getResourceStatusFromKubernetesConditions(
        plugin?.status?.statusConditions?.conditions
      )
    : null

  return (
    <Icon
      icon={plugin?.spec.disabled ? "error" : readyStatus?.icon}
      color={plugin?.spec.disabled ? "" : readyStatus?.color}
    />
  )
}

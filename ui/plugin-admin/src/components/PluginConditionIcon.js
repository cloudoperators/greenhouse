import React from "react"
import { Icon } from "juno-ui-components"

// PluginConditionIcon renders an icon based on the plugin status
export const PluginConditionIcon = ({ plugin }) => {
  return (
    <Icon
      icon={plugin?.disabled ? "error" : plugin?.readyStatus?.icon}
      color={plugin?.disabled ? "" : plugin?.readyStatus?.color}
    />
  )
}

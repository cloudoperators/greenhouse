/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { ClusterSelector, Plugin, PluginPreset } from "../../../types/types"

const initPluginPreset = (
  pluginPresetName: string,
  plugin: Plugin
): PluginPreset => {
  delete plugin.spec!.clusterName
  let pluginPreset: PluginPreset = {
    metadata: {
      name: pluginPresetName,
    },
    kind: "PluginPreset",
    apiVersion: "greenhouse.sap/v1alpha1",
    spec: {
      plugin: plugin.spec!,
      clusterSelector: {},
    },
  }
  return pluginPreset
}

export default initPluginPreset

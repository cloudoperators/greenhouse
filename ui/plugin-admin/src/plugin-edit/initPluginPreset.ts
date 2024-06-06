/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { PluginPreset } from "../../../types/types"
import { EditFormData } from "../plugindefinitions/store"

export const initPluginPreset = (formData: EditFormData): PluginPreset => {
  if (!!formData.spec?.clusterName) {
    delete formData.spec.clusterName
  }
  let pluginPreset: PluginPreset = {
    metadata: formData.metadata!,
    kind: "PluginPreset",
    apiVersion: "greenhouse.sap/v1alpha1",
    spec: {
      plugin: formData.spec!,
      clusterSelector: {
        matchLabels: formData.labelSelector!,
      },
    },
  }
  return pluginPreset
}

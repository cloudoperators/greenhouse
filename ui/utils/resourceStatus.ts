/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  KubernetesCondition,
  ResourceStatus,
  ResourceStatusCondition,
} from "../types/types"

const resourceStatusUnknown: ResourceStatus = {
  state: ResourceStatusCondition[ResourceStatusCondition.unkown],
  color: "text-theme-default",
  icon: "warning",
}

// Depends on a "Ready" condition within an kubernetes conditions array
const getResourceStatusFromKubernetesConditions = (
  conditions: KubernetesCondition[]
): ResourceStatus => {
  let message = ""
  let resourceStatus: ResourceStatus = conditions.some((condition) => {
    message = condition.message ?? ""
    return condition.type === "Ready" && condition.status === "True"
  })
    ? {
        state: ResourceStatusCondition[ResourceStatusCondition.ready],
        color: "text-theme-accent",
        icon: "success",
        message: message,
      }
    : conditions.some((condition) => {
        message = condition.message ?? ""
        return condition.type === "Ready" && condition.status === "False"
      })
    ? {
        state: ResourceStatusCondition[ResourceStatusCondition.notReady],
        color: "text-theme-danger",
        icon: "danger",
        message: message,
      }
    : resourceStatusUnknown

  return resourceStatus
}

export { getResourceStatusFromKubernetesConditions, resourceStatusUnknown }

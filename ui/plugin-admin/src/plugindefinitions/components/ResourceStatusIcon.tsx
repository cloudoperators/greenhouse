/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Icon } from "juno-ui-components"
import React from "react"
import { ResourceStatus } from "../types/types"

interface ResourceIconProps {
  status: ResourceStatus
}

const ResourceStatusIcon: React.FC<ResourceIconProps> = (
  props: ResourceIconProps
) => {
  return (
    props.status !== null && (
      <Icon icon={props.status.icon} color={props.status.color} />
    )
  )
}

export default ResourceStatusIcon

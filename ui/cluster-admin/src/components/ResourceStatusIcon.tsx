/*
 * Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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

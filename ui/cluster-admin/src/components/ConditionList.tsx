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

import React from "react"
import {
  Stack,
  Pill,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "juno-ui-components"
import { KubernetesCondition } from "../types/types"
import humanizedTimePeriodToNow from "../lib/utils/humanizedTimePeriodToNow"

interface ConditionPillListProps {
  conditionArray: KubernetesCondition[]
}
const ConditionList: React.FC<ConditionPillListProps> = (
  props: ConditionPillListProps
) => {
  return (
    <Stack gap="2" alignment="start" wrap={true}>
      {props.conditionArray.map((condition: KubernetesCondition) => {
        return (
          <Tooltip triggerEvent="hover">
            <TooltipTrigger>
              <Pill
                pillKeyLabel={condition.type}
                pillKey={condition.type}
                pillKeyValue={condition.status}
                pillValue={condition.status}
              />
            </TooltipTrigger>
            <TooltipContent>
              <ul>
                <li>
                  {humanizedTimePeriodToNow(condition.lastTransitionTime!)} ago
                </li>
                <li>{condition.message}</li>
              </ul>
            </TooltipContent>
          </Tooltip>
        )
      })}
    </Stack>
  )
}

export default ConditionList

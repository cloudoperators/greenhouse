/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import useNamespace from "../plugindefinitions/hooks/useNamespace"
import useClient from "../plugindefinitions/hooks/useClient"
import React, { useEffect } from "react"
import { Cluster } from "../../../types/types"
import { Select, SelectOption } from "juno-ui-components"

interface ClusterSelectProps {
  id?: string
  label?: string
  placeholder?: string
  defaultValue?: string
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void
}

const ClusterSelect: React.FC<ClusterSelectProps> = (
  props: ClusterSelectProps
) => {
  const { client: client } = useClient()
  const { namespace } = useNamespace()
  const [availableClusters, setAvailableClusters] = React.useState<Cluster[]>(
    []
  )
  useEffect(() => {
    client
      .get(`/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/clusters`, {})
      .then((res) => {
        if (res.kind !== "ClusterList") {
          console.log("ERROR: Failed to get Clusters")
        } else {
          setAvailableClusters(res.items as Cluster[])
        }
      })
  }, [client, namespace])

  const handleChange = (value: string): void => {
    let e = {
      target: {
        value: value,
        id: props.id,
        type: "cluster-select",
      },
    } as React.ChangeEvent<HTMLInputElement>
    if (props.onChange) {
      props.onChange!(e)
    }
  }

  return (
    <Select
      id={props.id}
      placeholder={props.placeholder}
      label={props.label}
      defaultValue={props.defaultValue}
      onChange={handleChange}
    >
      {availableClusters.map((cluster) => {
        return (
          <SelectOption
            key={cluster.metadata!.name}
            value={cluster.metadata!.name}
          />
        )
      })}
    </Select>
  )
}

export default ClusterSelect

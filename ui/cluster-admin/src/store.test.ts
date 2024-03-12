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

import useStore from "./store"
import { act, renderHook } from "@testing-library/react"
import { Cluster, UpdateClusterAction } from "./types/types"

let addItem = [
  {
    apiVersion: "greenhouse.sap/v1alpha1",
    kind: "TeamMembership",
    metadata: {
      name: "observability",
    },
  },
]

let testCluster: Cluster = {
  apiVersion: "greenhouse.sap/v1alpha1",
  kind: "Cluster",
  metadata: {
    name: "test-cluster",
    namespace: "test-namespace",
  },
}

describe("store tests", () => {
  afterEach(() => {
    const { result } = renderHook(() => useStore())
    act(() => {
      result.current.clusters = []
    })
  })

  describe("Add Clusters", () => {
    test("Should successfully add clusters", () => {
      const { result } = renderHook(() => useStore())
      act(() => {
        result.current.updateClusters({
          clusters: [testCluster],
          action: UpdateClusterAction.add,
        })
      })
      expect(result.current.clusters).toEqual([testCluster])
    })
    test("Should deny invalid clusters", () => {
      const { result } = renderHook(() => useStore())
      act(() => {
        result.current.updateClusters({
          clusters: [{}],
          action: UpdateClusterAction.add,
        })
      })
      expect(result.current.clusters).toHaveLength(0)
    })
    test("Should not duplicate cluster input", () => {
      const { result } = renderHook(() => useStore())
      act(() => {
        result.current.updateClusters({
          clusters: [testCluster],
          action: UpdateClusterAction.add,
        })
        result.current.updateClusters({
          clusters: [testCluster],
          action: UpdateClusterAction.add,
        })
      })
      expect(result.current.clusters).toHaveLength(1)
    })
  })

  describe("Modify Cluster", () => {
    const version = "greenhouse.sap/v1alpha1"

    test("check valid modification", () => {
      const { result } = renderHook(() => useStore())
      act(() => {
        result.current.updateClusters({
          clusters: [testCluster],
          action: UpdateClusterAction.add,
        })
        let updateTestCluster = { ...testCluster }
        updateTestCluster.metadata!.name = "updated-name"

        result.current.updateClusters({
          clusters: [updateTestCluster],
          action: UpdateClusterAction.add,
        })
      })

      expect(result.current.clusters[0].metadata!.name!).toEqual("updated-name")
      expect(result.current.clusters).toHaveLength(1)
    })
  })
  describe("Delete Cluster", () => {
    test("check valid deletion", () => {
      const { result } = renderHook(() => useStore())
      act(() => {
        result.current.updateClusters({
          clusters: [testCluster],
          action: UpdateClusterAction.add,
        })
      })
      expect(result.current.clusters).toHaveLength(1)
      act(() => {
        result.current.updateClusters({
          clusters: [testCluster],
          action: UpdateClusterAction.delete,
        })
      })
      expect(result.current.clusters).toHaveLength(0)
    })
  })
})

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  Container,
  DataGridToolbar,
  ButtonRow,
} from "juno-ui-components"
import ClusterDetail from "./components/ClusterDetail"
import ClusterList from "./components/ClusterList"
import DownloadKubeConfig from "./components/DownloadKubeConfig"
import OnBoardCluster from "./components/OnBoardCluster"
import WelcomeView from "./components/WelcomeView"
import useNamespace from "./hooks/useNamespace"
import useStore from "./store"

const AppContent = () => {
  const clusters = [
    {
      apiVersion: "greenhouse.sap/v1alpha1",
      kind: "Cluster",
      metadata: {
        creationTimestamp: "2024-02-07T10:23:23Z",
        finalizers: ["greenhouse.sap/cluster"],
        generation: 1,
        managedFields: [
          {
            apiVersion: "greenhouse.sap/v1alpha1",
            fieldsType: "FieldsV1",
            fieldsV1: {
              "f:metadata": {
                "f:finalizers": {
                  ".": {},
                  'v:"greenhouse.sap/cluster"': {},
                },
              },
              "f:spec": {
                ".": {},
                "f:accessMode": {},
              },
            },
            manager: "Go-http-client",
            operation: "Update",
            time: "2024-02-07T10:23:23Z",
          },
          {
            apiVersion: "greenhouse.sap/v1alpha1",
            fieldsType: "FieldsV1",
            fieldsV1: {
              "f:status": {
                ".": {},
                "f:statusConditions": {
                  ".": {},
                  "f:conditions": {
                    ".": {},
                    'k:{"type":"AllNodesReady"}': {
                      ".": {},
                      "f:type": {},
                    },
                    'k:{"type":"KubeConfigValid"}': {
                      ".": {},
                      "f:type": {},
                    },
                    'k:{"type":"Ready"}': {
                      ".": {},
                      "f:type": {},
                    },
                  },
                },
              },
            },
            manager: "Go-http-client",
            operation: "Update",
            subresource: "status",
            time: "2024-03-04T05:27:04Z",
          },
          {
            apiVersion: "greenhouse.sap/v1alpha1",
            fieldsType: "FieldsV1",
            fieldsV1: {
              "f:status": {
                "f:bearerTokenExpirationTimestamp": {},
                "f:kubernetesVersion": {},
                "f:nodes": {
                  ".": {},
                  "f:shoot--greenhouse--monitoring-worker-bsadm-z1-747d6-48k7t":
                    {
                      ".": {},
                      "f:ready": {},
                      "f:statusConditions": {
                        ".": {},
                        "f:conditions": {
                          ".": {},
                          'k:{"type":"ClusterNetworkProblem"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"CorruptDockerOverlay2"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"DiskPressure"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"FrequentContainerdRestart"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"FrequentDockerRestart"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"FrequentKubeletRestart"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"FrequentUnregisterNetDevice"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"HostNetworkProblem"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"KernelDeadlock"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"MemoryPressure"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"NetworkUnavailable"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"PIDPressure"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"ReadonlyFilesystem"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"Ready"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                        },
                      },
                    },
                  "f:shoot--greenhouse--monitoring-worker-bsadm-z1-747d6-f9gg8":
                    {
                      ".": {},
                      "f:ready": {},
                      "f:statusConditions": {
                        ".": {},
                        "f:conditions": {
                          ".": {},
                          'k:{"type":"ClusterNetworkProblem"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"CorruptDockerOverlay2"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"DiskPressure"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"FrequentContainerdRestart"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"FrequentDockerRestart"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"FrequentKubeletRestart"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"FrequentUnregisterNetDevice"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"HostNetworkProblem"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"KernelDeadlock"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"MemoryPressure"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"NetworkUnavailable"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"PIDPressure"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"ReadonlyFilesystem"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                          'k:{"type":"Ready"}': {
                            ".": {},
                            "f:lastTransitionTime": {},
                            "f:message": {},
                            "f:status": {},
                            "f:type": {},
                          },
                        },
                      },
                    },
                },
                "f:statusConditions": {
                  "f:conditions": {
                    'k:{"type":"AllNodesReady"}': {
                      "f:lastTransitionTime": {},
                      "f:status": {},
                    },
                    'k:{"type":"KubeConfigValid"}': {
                      "f:lastTransitionTime": {},
                      "f:status": {},
                    },
                    'k:{"type":"Ready"}': {
                      "f:lastTransitionTime": {},
                      "f:status": {},
                    },
                  },
                },
              },
            },
            manager: "greenhouse",
            operation: "Update",
            subresource: "status",
            time: "2024-04-09T12:10:31Z",
          },
        ],
        name: "monitoring",
        namespace: "ccloud",
        resourceVersion: "331828238",
        uid: "0db6e464-ec36-459e-8a05-4ad668b57f42",
      },
      spec: {
        accessMode: "direct",
      },
      status: {
        bearerTokenExpirationTimestamp: "2024-04-10T10:03:43Z",
        kubernetesVersion: "v1.27.11",
        nodes: {
          "shoot--greenhouse--monitoring-worker-bsadm-z1-747d6-48k7t": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-08T03:24:51Z",
                  message: "no cluster network problems",
                  status: "False",
                  type: "ClusterNetworkProblem",
                },
                {
                  lastTransitionTime: "2024-03-08T03:24:34Z",
                  message: "no host network problems",
                  status: "False",
                  type: "HostNetworkProblem",
                },
                {
                  lastTransitionTime: "2024-03-31T22:46:37Z",
                  message: "kubelet is functioning properly",
                  status: "False",
                  type: "FrequentKubeletRestart",
                },
                {
                  lastTransitionTime: "2024-03-31T22:46:37Z",
                  message: "docker is functioning properly",
                  status: "False",
                  type: "FrequentDockerRestart",
                },
                {
                  lastTransitionTime: "2024-03-31T22:46:37Z",
                  message: "containerd is functioning properly",
                  status: "False",
                  type: "FrequentContainerdRestart",
                },
                {
                  lastTransitionTime: "2024-03-31T22:46:37Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-31T22:46:37Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-31T22:46:37Z",
                  message: "docker overlay2 is functioning properly",
                  status: "False",
                  type: "CorruptDockerOverlay2",
                },
                {
                  lastTransitionTime: "2024-03-31T22:46:37Z",
                  message: "node is functioning properly",
                  status: "False",
                  type: "FrequentUnregisterNetDevice",
                },
                {
                  lastTransitionTime: "2024-03-08T03:23:39Z",
                  message: "Calico is running on this node",
                  status: "False",
                  type: "NetworkUnavailable",
                },
                {
                  lastTransitionTime: "2024-04-02T07:37:56Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-02T07:37:56Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-02T07:37:56Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T03:44:15Z",
                  message: "kubelet is posting ready status. AppArmor enabled",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "shoot--greenhouse--monitoring-worker-bsadm-z1-747d6-f9gg8": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-08T03:22:19Z",
                  message: "no cluster network problems",
                  status: "False",
                  type: "ClusterNetworkProblem",
                },
                {
                  lastTransitionTime: "2024-03-08T03:23:05Z",
                  message: "no host network problems",
                  status: "False",
                  type: "HostNetworkProblem",
                },
                {
                  lastTransitionTime: "2024-03-31T22:45:37Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-31T22:45:37Z",
                  message: "docker overlay2 is functioning properly",
                  status: "False",
                  type: "CorruptDockerOverlay2",
                },
                {
                  lastTransitionTime: "2024-03-31T22:45:37Z",
                  message: "node is functioning properly",
                  status: "False",
                  type: "FrequentUnregisterNetDevice",
                },
                {
                  lastTransitionTime: "2024-03-31T22:45:37Z",
                  message: "kubelet is functioning properly",
                  status: "False",
                  type: "FrequentKubeletRestart",
                },
                {
                  lastTransitionTime: "2024-03-31T22:45:37Z",
                  message: "docker is functioning properly",
                  status: "False",
                  type: "FrequentDockerRestart",
                },
                {
                  lastTransitionTime: "2024-03-31T22:45:37Z",
                  message: "containerd is functioning properly",
                  status: "False",
                  type: "FrequentContainerdRestart",
                },
                {
                  lastTransitionTime: "2024-03-31T22:45:37Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-08T03:21:12Z",
                  message: "Calico is running on this node",
                  status: "False",
                  type: "NetworkUnavailable",
                },
                {
                  lastTransitionTime: "2024-04-05T22:37:37Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-05T22:37:37Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-05T22:37:37Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-05T22:37:37Z",
                  message: "kubelet is posting ready status. AppArmor enabled",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
        },
        statusConditions: {
          conditions: [
            {
              lastTransitionTime: "2024-04-09T12:10:31Z",
              status: "True",
              type: "Ready",
            },
            {
              lastTransitionTime: "2024-04-09T12:10:31Z",
              status: "True",
              type: "AllNodesReady",
            },
            {
              lastTransitionTime: "2024-04-09T12:10:31Z",
              status: "True",
              type: "KubeConfigValid",
            },
          ],
        },
      },
    },
    {
      apiVersion: "greenhouse.sap/v1alpha1",
      kind: "Cluster",
      metadata: {
        creationTimestamp: "2023-09-25T18:47:23Z",
        finalizers: [
          "greenhouse.sap/cluster",
          "greenhouse.sap/propagatedResource",
        ],
        generation: 1,
        managedFields: [
          {
            apiVersion: "greenhouse.sap/v1alpha1",
            fieldsType: "FieldsV1",
            fieldsV1: {
              "f:metadata": {
                "f:finalizers": {
                  ".": {},
                  'v:"greenhouse.sap/cluster"': {},
                  'v:"greenhouse.sap/propagatedResource"': {},
                },
              },
              "f:spec": {
                ".": {},
                "f:accessMode": {},
              },
            },
            manager: "Go-http-client",
            operation: "Update",
            time: "2023-10-05T06:43:23Z",
          },
          {
            apiVersion: "greenhouse.sap/v1alpha1",
            fieldsType: "FieldsV1",
            fieldsV1: {
              "f:status": {
                ".": {},
                "f:conditions": {},
                "f:nodes": {
                  "f:minion-0125e7cb233b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-038544ee47fe.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-0911a1122cbc.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-0c7ac0bb9bcd.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-0ce43f1ff613.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-135d8bca8db7.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-1376dcfce507.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-18b530b4f68d.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-221674c81fdf.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-2c11a584506c.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-34f64193bc82.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-3b00b56c2adb.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-400fe456f8c6.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-4151f8398206.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-41efd4906869.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-435e4a02f494.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-47bfc4ca27a4.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-4ac1401f77a8.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-4c07d928741c.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-53417db1eea5.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-5ec99d0a2741.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-6004999e873b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-64d946b88cc9.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-66efa0071a04.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-677e09a723e1.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-77adc1a2c6c0.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-7921bb114af3.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-7952f3cc43b9.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-7a57c0410532.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-7ce0ffd024db.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-7d0d1110ad75.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-81c3466f8fc6.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-81c91e239890.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-81d1dfe89ed0.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-8610a3ecf564.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-975227539c95.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-992b204a0a91.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-99a18125dd00.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-9b883b211b43.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-9c22d136c32b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-9f209521a8ed.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-a2b11f55966a.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-a3a2a725c620.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-a945e3ae5846.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-aa87f32bcaae.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-b4c105b6163b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-b6d9c4e90021.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-b8833abc9739.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-bec719c96c33.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-c76a04ec8090.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-c814acc9c181.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-cbc581edd7b3.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-cf982cc463aa.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-d057dc2f0f1b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-d4946cf53f0c.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-daf2618f931b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-dea87a2d0a31.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-dfe7ad68265b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-e42320920335.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-e7dc2e947a5a.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-e8776ec6368c.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-e8a5c7ae01c4.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-e9a5ebd643e1.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-eb21ef91d496.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-ebbc4a1fd8b0.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-ed9217ae7550.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-f12fcd310089.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-f36f95ff380c.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-f4c4d833fcd6.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-f50e79ee3ded.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-f66dde0c2cf6.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-f972e6f6377e.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-f9a591978dfb.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-f9bdc527e71f.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion-fe80400a2e3a.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-1cd957f1553e.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-1d7c43e98c52.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-40197a4d85fe.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-67676d32a38b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-7445cec911d0.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-779a50941998.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-902cc0d7b8d9.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-91dc22c869ad.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-aac9ba651ef3.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-b4114553166b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-c9da5cd554f3.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-f1da450603d8.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage-f796c463050b.cc.qa-de-1.cloud.sap": {
                    "f:conditions": {},
                  },
                },
                "f:statusConditions": {
                  ".": {},
                  "f:conditions": {
                    ".": {},
                    'k:{"type":"AllNodesReady"}': {
                      ".": {},
                      "f:type": {},
                    },
                    'k:{"type":"KubeConfigValid"}': {
                      ".": {},
                      "f:type": {},
                    },
                    'k:{"type":"Ready"}': {
                      ".": {},
                      "f:type": {},
                    },
                  },
                },
              },
            },
            manager: "Go-http-client",
            operation: "Update",
            subresource: "status",
            time: "2024-03-04T06:47:04Z",
          },
          {
            apiVersion: "greenhouse.sap/v1alpha1",
            fieldsType: "FieldsV1",
            fieldsV1: {
              "f:status": {
                "f:bearerTokenExpirationTimestamp": {},
                "f:kubernetesVersion": {},
                "f:nodes": {
                  ".": {},
                  "f:minion-0125e7cb233b.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-038544ee47fe.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-0911a1122cbc.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-0c7ac0bb9bcd.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-0ce43f1ff613.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-135d8bca8db7.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-1376dcfce507.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-18b530b4f68d.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-221674c81fdf.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-34f64193bc82.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-3b00b56c2adb.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-400fe456f8c6.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-4151f8398206.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-41efd4906869.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-435e4a02f494.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-4ac1401f77a8.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-4c07d928741c.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-53417db1eea5.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-5ec99d0a2741.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-6004999e873b.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-64d946b88cc9.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-66efa0071a04.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-677e09a723e1.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-77adc1a2c6c0.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-7921bb114af3.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-7952f3cc43b9.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-7a57c0410532.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-7ce0ffd024db.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-7d0d1110ad75.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-81c3466f8fc6.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-81c91e239890.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-81d1dfe89ed0.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-8610a3ecf564.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-975227539c95.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-992b204a0a91.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-99a18125dd00.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-9b883b211b43.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-9c22d136c32b.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-9f209521a8ed.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-a2b11f55966a.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-a3a2a725c620.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-a945e3ae5846.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-aa87f32bcaae.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-b4c105b6163b.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-b6d9c4e90021.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-b8833abc9739.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-c76a04ec8090.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-c814acc9c181.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-cbc581edd7b3.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-cf982cc463aa.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-d057dc2f0f1b.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-d4946cf53f0c.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-daf2618f931b.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-dea87a2d0a31.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-dfe7ad68265b.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-e42320920335.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-e7dc2e947a5a.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-e8a5c7ae01c4.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-e9a5ebd643e1.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-eb21ef91d496.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-ebbc4a1fd8b0.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-ed9217ae7550.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-f12fcd310089.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-f36f95ff380c.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-f50e79ee3ded.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-f66dde0c2cf6.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-f9a591978dfb.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-f9bdc527e71f.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:minion-fe80400a2e3a.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-1cd957f1553e.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-40197a4d85fe.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-67676d32a38b.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-7445cec911d0.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-902cc0d7b8d9.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"NetworkUnavailable"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-91dc22c869ad.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-aac9ba651ef3.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-b4114553166b.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"NetworkUnavailable"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-c9da5cd554f3.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-dd95225511f7.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                  "f:storage-f1da450603d8.cc.qa-de-1.cloud.sap": {
                    ".": {},
                    "f:ready": {},
                    "f:statusConditions": {
                      ".": {},
                      "f:conditions": {
                        ".": {},
                        'k:{"type":"BridgeFilterVLANTagged"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"DiskPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"KernelDeadlock"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"MemoryPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"PIDPressure"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"ReadonlyFilesystem"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                        'k:{"type":"Ready"}': {
                          ".": {},
                          "f:lastTransitionTime": {},
                          "f:message": {},
                          "f:status": {},
                          "f:type": {},
                        },
                      },
                    },
                  },
                },
                "f:statusConditions": {
                  "f:conditions": {
                    'k:{"type":"AllNodesReady"}': {
                      "f:lastTransitionTime": {},
                      "f:message": {},
                      "f:status": {},
                    },
                    'k:{"type":"KubeConfigValid"}': {
                      "f:lastTransitionTime": {},
                      "f:status": {},
                    },
                    'k:{"type":"Ready"}': {
                      "f:lastTransitionTime": {},
                      "f:status": {},
                    },
                  },
                },
              },
            },
            manager: "greenhouse",
            operation: "Update",
            subresource: "status",
            time: "2024-04-09T12:24:36Z",
          },
        ],
        name: "qa-de-1",
        namespace: "ccloud",
        resourceVersion: "331827031",
        uid: "f99b680f-ed98-4e1d-acc9-b987f1c8b0ca",
      },
      spec: {
        accessMode: "direct",
      },
      status: {
        bearerTokenExpirationTimestamp: "2024-04-10T10:01:43Z",
        kubernetesVersion: "v1.28.8",
        nodes: {
          "minion-0125e7cb233b.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:06Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:06Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:06Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-28T12:12:48Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:12:48Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:12:48Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:12:48Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-038544ee47fe.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T12:46:55Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:46:55Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:46:55Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:46:55Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-0911a1122cbc.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:10Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:10Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:10Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T10:17:17Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:17:17Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:17:17Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:17:17Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-0c7ac0bb9bcd.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:06Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:06Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:06Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T10:17:19Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:17:19Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:17:19Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:17:19Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-0ce43f1ff613.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:22:58Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:22:58Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:22:58Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-01T23:25:08Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:25:08Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:25:08Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:25:08Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-135d8bca8db7.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-1376dcfce507.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:22Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:22Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:22Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-28T12:17:08Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:17:08Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:17:08Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:17:08Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-18b530b4f68d.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:33:48Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:33:48Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:33:48Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-29T10:25:00Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:25:00Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:25:00Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:25:00Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-221674c81fdf.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:19Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:19Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:19Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T12:52:10Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:52:10Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:52:10Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:52:10Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-34f64193bc82.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:11Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:11Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:11Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T12:22:11Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:22:11Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:22:11Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:22:11Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-3b00b56c2adb.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:13Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:13Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:13Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-400fe456f8c6.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:28Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:28Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:28Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T12:28:30Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:28:30Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:28:30Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:28:30Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-4151f8398206.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-28T12:12:48Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:12:48Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:12:48Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:12:48Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-41efd4906869.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:38:31Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:38:31Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:38:31Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-06T05:15:27Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-06T05:15:27Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-06T05:15:27Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-06T05:15:37Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-435e4a02f494.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:12Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:12Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:12Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-4ac1401f77a8.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:11Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:11Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:11Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-29T08:47:57Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T19:04:19Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T19:04:19Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T07:22:32Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-4c07d928741c.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:03Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:03Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:03Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T13:14:37Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:14:37Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:14:37Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:14:37Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-53417db1eea5.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:14Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:14Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:14Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T13:19:18Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:19:18Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:19:18Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:19:18Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-5ec99d0a2741.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:29:54Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:29:54Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:29:54Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-08T22:38:54Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T00:19:59Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T00:19:59Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T00:19:59Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-6004999e873b.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:03Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:03Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:03Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:49Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:49Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:49Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:49Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-64d946b88cc9.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:03Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:03Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:03Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T12:54:41Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:54:41Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:54:41Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:54:41Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-66efa0071a04.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:32Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:32Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:32Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:50Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:50Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:50Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:50Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-677e09a723e1.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T13:47:32Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:47:32Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:47:32Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:47:32Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-77adc1a2c6c0.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:27Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:27Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:27Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T10:30:35Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:30:35Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:30:35Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:30:35Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-7921bb114af3.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:20Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:20Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:20Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T12:34:14Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:34:14Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:34:14Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:34:14Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-7952f3cc43b9.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:27:59Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:27:59Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:27:59Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:51Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:51Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:51Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:51Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-7a57c0410532.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:09Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:09Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:09Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T12:29:00Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:29:00Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:29:00Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:29:00Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-7ce0ffd024db.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:23Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:23Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:23Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T12:22:01Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:22:01Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:22:01Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:22:01Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-7d0d1110ad75.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:05Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:05Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:05Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T12:57:36Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:57:36Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:57:36Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:57:36Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-81c3466f8fc6.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:31Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:31Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:31Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-81c91e239890.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:02Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:02Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:02Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T13:33:03Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:33:03Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:33:03Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:33:03Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-81d1dfe89ed0.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:17:23Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:17:23Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:17:23Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:36Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:36Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:36Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:36Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-8610a3ecf564.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:22:59Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:59Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:59Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-975227539c95.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:02Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:02Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:02Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T13:02:45Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:02:45Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:02:45Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:02:45Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-992b204a0a91.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:44:59Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:44:59Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:44:59Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-29T10:28:21Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:28:21Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:28:21Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:28:21Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-99a18125dd00.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T13:36:25Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T13:36:25Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T13:36:25Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:47Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:47Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:47Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:47Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-9b883b211b43.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:34:53Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:34:53Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:34:53Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-06T00:19:23Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:39Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:39Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:39Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-9c22d136c32b.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:23:07Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:23:07Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:23:07Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:22:28Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T14:22:28Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T14:22:28Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T14:22:28Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-9f209521a8ed.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:57Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:57Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-a2b11f55966a.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:32:53Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:32:53Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:32:53Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-29T10:13:53Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:13:53Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:13:53Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:13:53Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-a3a2a725c620.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:23:32Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:23:32Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:23:32Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-a945e3ae5846.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:14Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:14Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:14Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-29T11:07:17Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T11:07:17Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T11:07:17Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T11:07:17Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-aa87f32bcaae.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:08Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:08Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:08Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T13:09:44Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:09:44Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:09:44Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:09:44Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-b4c105b6163b.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:28:43Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:28:43Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:28:43Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:54Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:54Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:54Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:54Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-b6d9c4e90021.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:40:02Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:40:02Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:40:02Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-07T17:38:53Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-07T17:38:53Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-07T17:38:53Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-07T17:38:53Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-b8833abc9739.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:08Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:08Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:08Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T12:48:24Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:48:24Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:48:24Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:48:24Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-c76a04ec8090.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:30Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:30Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:30Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:48Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:48Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:48Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:48Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-c814acc9c181.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:18:21Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:18:21Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:18:21Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-01T23:24:49Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:24:49Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:24:49Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:24:49Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-cbc581edd7b3.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:48:22Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:48:22Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:48:22Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-27T14:45:57Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-27T14:45:57Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-27T14:45:57Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-27T14:45:57Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-cf982cc463aa.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:29Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:29Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:29Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-d057dc2f0f1b.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:20Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:20Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:20Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:52Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:52Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:52Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:52Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-d4946cf53f0c.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:11Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:11Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:11Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:32Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:32Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:32Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:32Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-daf2618f931b.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:32Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:32Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:32Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:52Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:52Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:52Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:52Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-dea87a2d0a31.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:15Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:15Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:15Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-28T23:27:27Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:48Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:48Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:26:48Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-dfe7ad68265b.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:43:46Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:43:46Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:43:46Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-01T23:25:05Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:25:05Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:25:05Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:25:05Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-e42320920335.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:07Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:07Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:07Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:37Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:37Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:37Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:37Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-e7dc2e947a5a.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:29Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:29Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:29Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-28T12:27:02Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:27:02Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:27:02Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:27:02Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-e8a5c7ae01c4.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:06Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:06Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:06Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T12:43:05Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:43:05Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:43:05Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T12:43:05Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-e9a5ebd643e1.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T13:54:16Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:54:16Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:54:16Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-21T13:54:16Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-eb21ef91d496.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:50:25Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:50:25Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:50:25Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-07T17:38:51Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-07T17:38:51Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-07T17:38:51Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-07T17:38:51Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-ebbc4a1fd8b0.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:23Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:23Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:23Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T10:30:36Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:30:36Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:30:36Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T10:30:36Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-ed9217ae7550.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:26Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:26Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:26Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:47Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:47Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:47Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T05:28:47Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-f12fcd310089.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:38:16Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:38:16Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:38:16Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:57Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:57Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:57Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:24:57Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-f36f95ff380c.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:17:44Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:17:44Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:17:44Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:34Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:34Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:34Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:34Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-f50e79ee3ded.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:45:49Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-21T14:45:49Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:45:49Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-29T10:28:49Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:28:49Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:28:49Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:28:49Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-f66dde0c2cf6.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:37Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:37Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:37Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:37Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-f9a591978dfb.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:32Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:32Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:32Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-f9bdc527e71f.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-21T14:50:12Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-21T14:50:12Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-21T14:50:12Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-29T10:14:28Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:14:28Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:14:28Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-29T10:14:28Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "minion-fe80400a2e3a.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-27T15:59:50Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-27T15:59:50Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-27T15:59:50Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-27T16:00:00Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-1cd957f1553e.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:26Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:26Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:26Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-22T08:41:51Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T08:41:51Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T08:41:51Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T08:41:51Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-40197a4d85fe.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:04Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:04Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:04Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-22T10:24:36Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T10:24:36Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T10:24:36Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T10:24:36Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-67676d32a38b.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-04T06:04:41Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-7445cec911d0.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:20Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:20Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:20Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-03-22T10:13:52Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T10:13:52Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T10:13:52Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T10:13:52Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-902cc0d7b8d9.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-22T11:17:09Z",
                  message: "Calico is running on this node",
                  status: "False",
                  type: "NetworkUnavailable",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:00Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-01T23:24:58Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:24:58Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:24:58Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-04-01T23:24:58Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-91dc22c869ad.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:29:10Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-aac9ba651ef3.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:17Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-22T10:46:09Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T10:46:09Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T10:46:09Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T10:46:09Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-b4114553166b.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-03-22T11:29:25Z",
                  message: "Calico is running on this node",
                  status: "False",
                  type: "NetworkUnavailable",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:22:56Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-22T11:29:15Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T11:29:15Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T11:29:15Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T11:29:15Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-c9da5cd554f3.cc.qa-de-1.cloud.sap": {
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-02-22T09:05:45Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-02-22T09:05:45Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-02-22T09:05:45Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-14T18:47:03Z",
                  message: "Kubelet stopped posting node status.",
                  status: "Unknown",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-14T18:47:03Z",
                  message: "Kubelet stopped posting node status.",
                  status: "Unknown",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-14T18:47:03Z",
                  message: "Kubelet stopped posting node status.",
                  status: "Unknown",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-14T18:47:03Z",
                  message: "Kubelet stopped posting node status.",
                  status: "Unknown",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-dd95225511f7.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:22Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:22Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:22Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-03-28T12:17:12Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:17:12Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:17:12Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-28T12:17:12Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
          "storage-f1da450603d8.cc.qa-de-1.cloud.sap": {
            ready: true,
            statusConditions: {
              conditions: [
                {
                  lastTransitionTime: "2024-04-09T12:23:27Z",
                  message:
                    "Don’t pass bridged VLAN-tagged ARP/IP traffic to ARPtables/IPtables",
                  status: "False",
                  type: "BridgeFilterVLANTagged",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:27Z",
                  message: "kernel has no deadlock",
                  status: "False",
                  type: "KernelDeadlock",
                },
                {
                  lastTransitionTime: "2024-04-09T12:23:27Z",
                  message: "Filesystem is not read-only",
                  status: "False",
                  type: "ReadonlyFilesystem",
                },
                {
                  lastTransitionTime: "2024-03-22T09:11:07Z",
                  message: "kubelet has sufficient memory available",
                  status: "False",
                  type: "MemoryPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T09:11:07Z",
                  message: "kubelet has no disk pressure",
                  status: "False",
                  type: "DiskPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T09:11:07Z",
                  message: "kubelet has sufficient PID available",
                  status: "False",
                  type: "PIDPressure",
                },
                {
                  lastTransitionTime: "2024-03-22T09:11:07Z",
                  message: "kubelet is posting ready status",
                  status: "True",
                  type: "Ready",
                },
              ],
            },
          },
        },
        statusConditions: {
          conditions: [
            {
              lastTransitionTime: "2024-03-18T12:55:57Z",
              status: "True",
              type: "Ready",
            },
            {
              lastTransitionTime: "2024-03-18T12:55:57Z",
              message: "storage-c9da5cd554f3.cc.qa-de-1.cloud.sap not ready",
              status: "False",
              type: "AllNodesReady",
            },
            {
              lastTransitionTime: "2024-03-18T12:55:57Z",
              status: "True",
              type: "KubeConfigValid",
            },
          ],
        },
      },
    },
    {
      apiVersion: "greenhouse.sap/v1alpha1",
      kind: "Cluster",
      metadata: {
        creationTimestamp: "2023-09-27T18:42:37Z",
        finalizers: [
          "greenhouse.sap/cluster",
          "greenhouse.sap/propagatedResource",
        ],
        generation: 1,
        managedFields: [
          {
            apiVersion: "greenhouse.sap/v1alpha1",
            fieldsType: "FieldsV1",
            fieldsV1: {
              "f:metadata": {
                "f:finalizers": {
                  ".": {},
                  'v:"greenhouse.sap/cluster"': {},
                  'v:"greenhouse.sap/propagatedResource"': {},
                },
              },
              "f:spec": {
                ".": {},
                "f:accessMode": {},
              },
            },
            manager: "Go-http-client",
            operation: "Update",
            time: "2023-10-05T06:43:27Z",
          },
          {
            apiVersion: "greenhouse.sap/v1alpha1",
            fieldsType: "FieldsV1",
            fieldsV1: {
              "f:status": {
                ".": {},
                "f:conditions": {},
                "f:headScaleStatus": {},
                "f:nodes": {
                  "f:minion0.cc.qa-de-2.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion1.cc.qa-de-2.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:minion2.cc.qa-de-2.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage0.cc.qa-de-2.cloud.sap": {
                    "f:conditions": {},
                  },
                  "f:storage1.cc.qa-de-2.cloud.sap": {
                    "f:conditions": {},
                  },
                },
                "f:statusConditions": {
                  ".": {},
                  "f:conditions": {
                    ".": {},
                    'k:{"type":"AllNodesReady"}': {
                      ".": {},
                      "f:type": {},
                    },
                    'k:{"type":"HeadscaleReady"}': {
                      ".": {},
                      "f:type": {},
                    },
                    'k:{"type":"KubeConfigValid"}': {
                      ".": {},
                      "f:type": {},
                    },
                    'k:{"type":"Ready"}': {
                      ".": {},
                      "f:type": {},
                    },
                  },
                },
              },
            },
            manager: "Go-http-client",
            operation: "Update",
            subresource: "status",
            time: "2024-03-04T07:58:09Z",
          },
          {
            apiVersion: "greenhouse.sap/v1alpha1",
            fieldsType: "FieldsV1",
            fieldsV1: {
              "f:status": {
                "f:bearerTokenExpirationTimestamp": {},
                "f:kubernetesVersion": {},
                "f:statusConditions": {
                  "f:conditions": {
                    'k:{"type":"AllNodesReady"}': {
                      "f:lastTransitionTime": {},
                      "f:message": {},
                      "f:status": {},
                    },
                    'k:{"type":"HeadscaleReady"}': {
                      "f:lastTransitionTime": {},
                      "f:message": {},
                      "f:status": {},
                    },
                    'k:{"type":"KubeConfigValid"}': {
                      "f:lastTransitionTime": {},
                      "f:message": {},
                      "f:status": {},
                    },
                    'k:{"type":"Ready"}': {
                      "f:lastTransitionTime": {},
                      "f:message": {},
                      "f:status": {},
                    },
                  },
                },
              },
            },
            manager: "greenhouse",
            operation: "Update",
            subresource: "status",
            time: "2024-04-09T13:17:42Z",
          },
        ],
        name: "qa-de-2",
        namespace: "ccloud",
        resourceVersion: "331826437",
        uid: "a0746265-90c9-4a60-ac44-7b12746ae450",
      },
      spec: {
        accessMode: "headscale",
      },
      status: {
        bearerTokenExpirationTimestamp: "2024-03-18T13:22:34Z",
        headScaleStatus: {},
        kubernetesVersion: "unknown",
        statusConditions: {
          conditions: [
            {
              lastTransitionTime: "2024-04-09T13:17:42Z",
              message: "no headscale machine found",
              status: "False",
              type: "HeadscaleReady",
            },
            {
              lastTransitionTime: "2024-03-17T15:10:27Z",
              message: "Headscale connection not ready",
              status: "False",
              type: "Ready",
            },
            {
              lastTransitionTime: "2024-03-17T15:10:27Z",
              message: "kubeconfig not valid - cannot know node status",
              status: "Unknown",
              type: "AllNodesReady",
            },
            {
              lastTransitionTime: "2024-03-17T15:10:22Z",
              message:
                'Get "https://100.126.0.3/version?timeout=32s": proxyconnect tcp: dial tcp 100.110.75.28:1055: connect: connection refused',
              status: "False",
              type: "KubeConfigValid",
            },
          ],
        },
      },
    },
  ]
  const clusterDetails = useStore((state) => state.clusterDetails)
  const showClusterDetails = useStore((state) => state.showClusterDetails)
  const showOnBoardCluster = useStore((state) => state.showOnBoardCluster)
  const showDownloadKubeConfig = useStore(
    (state) => state.showDownloadKubeConfig
  )
  const auth = useStore((state) => state.auth)
  const authError = auth?.error
  const expiryTimestamp = auth?.parsed.expiresAt
  const { namespace } = useNamespace()
  const apiEndpoint = useStore((state) => state.endpoint)
  const loggedIn = useStore((state) => state.loggedIn)
  const setShowOnBoardCluster = useStore((state) => state.setShowOnBoardCluster)
  const setShowClusterDetails = useStore((state) => state.setShowClusterDetails)
  const setShowDownloadKubeConfig = useStore(
    (state) => state.setShowDownloadKubeConfig
  )

  const openOnBoardCluster = () => {
    setShowOnBoardCluster(true)
    setShowClusterDetails(false)
    setShowDownloadKubeConfig(false)
  }

  const openShowDownloadKubeConfig = () => {
    setShowOnBoardCluster(false)
    setShowClusterDetails(false)
    setShowDownloadKubeConfig(true)
  }

  return (
    <Container>
      {loggedIn && !authError ? (
        <>
          <DataGridToolbar>
            <ButtonRow>
              <Button
                icon="openInBrowser"
                label="Access greenhouse cluster"
                onClick={() => openShowDownloadKubeConfig()}
              />
              <Button
                icon="addCircle"
                label="Onboard Cluster"
                onClick={() => openOnBoardCluster()}
              />
            </ButtonRow>
          </DataGridToolbar>

          {showOnBoardCluster && <OnBoardCluster />}
          {showDownloadKubeConfig && (
            <DownloadKubeConfig
              namespace={namespace}
              token={auth?.JWT}
              endpoint={apiEndpoint}
              expiry={expiryTimestamp}
            />
          )}
          {clusters.length > 0 && <ClusterList clusters={clusters} />}
          {showClusterDetails && clusterDetails.cluster && <ClusterDetail />}
        </>
      ) : (
        <WelcomeView />
      )}
    </Container>
  )
}

export default AppContent

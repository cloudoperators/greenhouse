/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

// test of useAPI hook
import { describe, expect, test } from "@jest/globals"
import { createPluginConfig, buildExternalServicesUrls } from "./useAPI"

describe("createPluginConfig", () => {
  // checks if all fields are created correctly
  test("createPluginConfig with all important fields", () => {
    const items = db1
    const result = createPluginConfig(items)
    expect(result).toEqual(res1)
  })

  // checks if the function works with just metadata name
  test("createPluginConfig with just metadata name", () => {
    const items = db2
    const result = createPluginConfig(items)
    expect(result).toEqual(res2)
  })
})

describe("buildExternalServicesUrls", () => {
  // checks if the function works with no external services
  test("buildExternalServicesUrls with no external services", () => {
    const items = undefined
    const result = buildExternalServicesUrls(items)
    expect(result).toEqual(null)
  })

  // checks if the function works with URLs with and without a name in Data
  test("buildExternalServicesUrls with external services", () => {
    const items = {
      "https://example.com": {
        name: "exposed-service",
        port: 80,
        namespace: "default",
      },
      "https://example.org": {
        a: "b",
      },
    }

    const result = buildExternalServicesUrls(items)
    expect(result).toEqual([
      {
        url: "https://example.com",
        name: "exposed-service",
      },
      {
        url: "https://example.org",
        name: "https://example.org",
      },
    ])
  })
})

// mock data

const db1 = [
  {
    metadata: {
      name: "test",
    },
    spec: {
      clusterName: "c1",
      disabled: false,
      displayName: "Test",
      optionValues: [
        { name: "value1", value: true },
        {
          name: "greenhouse.value2",
          value: "hidden",
        },
      ],
    },
    status: {
      exposedServices: {
        "https://example.com": {
          name: "exposed-service",
        },
        "https://example.org": {
          a: "b",
        },
      },
      statusConditions: {
        conditions: [
          {
            message: "ready",
            status: "True",
            type: "Ready",
          },
          {
            status: "True",
            type: "ClusterAccessReady",
          },
        ],
      },
      version: "1.6.0",
    },
  },
]

const res1 = [
  {
    id: "test",
    name: "Test",
    version: "1.6.0",
    clusterName: "c1",
    externalServicesUrls: [
      {
        url: "https://example.com",
        name: "exposed-service",
      },
      {
        url: "https://example.org",
        name: "https://example.org",
      },
    ],
    statusConditions: [
      {
        message: "ready",
        status: "True",
        type: "Ready",
      },
      {
        status: "True",
        type: "ClusterAccessReady",
      },
    ],
    readyStatus: {
      state: "ready",
      color: "text-theme-accent",
      icon: "success",
      message: "ready",
    },
    optionValues: [
      { name: "value1", value: true },
      { name: "greenhouse.value2", value: "hidden" },
    ],
    disabled: false,
    raw: {
      metadata: { name: "test" },
      spec: {
        clusterName: "c1",
        disabled: false,
        displayName: "Test",
        optionValues: [
          { name: "value1", value: true },
          { name: "greenhouse.value2", value: "hidden" },
        ],
      },
      status: {
        exposedServices: {
          "https://example.com": {
            name: "exposed-service",
          },

          "https://example.org": {
            a: "b",
          },
        },
        statusConditions: {
          conditions: [
            {
              message: "ready",
              status: "True",
              type: "Ready",
            },
            {
              status: "True",
              type: "ClusterAccessReady",
            },
          ],
        },
        version: "1.6.0",
      },
    },
  },
]

const db2 = [
  {
    metadata: {
      name: "test",
    },
  },
]

const res2 = [
  {
    clusterName: undefined,
    externalServicesUrls: null,
    id: "test",
    name: "test",
    optionValues: undefined,
    disabled: undefined,
    raw: {
      metadata: {
        name: "test",
      },
    },
    readyStatus: null,
    statusConditions: undefined,
    version: undefined,
  },
]

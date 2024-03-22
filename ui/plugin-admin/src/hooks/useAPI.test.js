// test of useAPI hook
import { describe, expect, test } from "@jest/globals"
import { createPluginConfig } from "./useAPI"

describe("useAPI", () => {
  test("createPluginConfig with all important fields", () => {
    const items = db1
    const result = createPluginConfig(items)
    expect(result).toEqual(res1)
  })
})

describe("useAPI", () => {
  test("createPluginConfig with just metadata name", () => {
    const items = db2
    const result = createPluginConfig(items)
    expect(result).toEqual(res2)
  })
})

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

// test of useAPI hook
import { describe, expect, test } from "@jest/globals"
import { createPluginConfig } from "./useAPI"

describe("useAPI", () => {
  test("createPluginConfig", () => {
    const items = db1

    const result = createPluginConfig(items)
    expect(result).toEqual(res1)
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

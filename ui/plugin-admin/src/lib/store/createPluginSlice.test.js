/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import * as React from "react"
import { renderHook, act } from "@testing-library/react"
import StoreProvider, {
  usePluginActions,
  usePluginConfig,
} from "../../components/StoreProvider"

describe("createPluginSlice", () => {
  describe("setPluginConfig", () => {
    it("test if pluginConfig are saved sorted correctly", () => {
      const wrapper = ({ children }) => (
        <StoreProvider>{children}</StoreProvider>
      )
      const store = renderHook(
        () => ({
          pluginActions: usePluginActions(),
          pluginConfig: usePluginConfig(),
        }),
        { wrapper }
      )

      act(() => {
        store.result.current.pluginActions.setPluginConfig([
          { id: "onePlugin", disabled: false },
          { id: "alert", disabled: false },
          { id: "example app", disabled: true },
        ])
      })

      expect(store.result.current.pluginConfig).toEqual([
        { id: "alert", disabled: false },
        { id: "onePlugin", disabled: false },
        { id: "example app", disabled: true },
      ])
    })
    it("test if pluginConfig is saved correctly without disabled field", () => {
      const wrapper = ({ children }) => (
        <StoreProvider>{children}</StoreProvider>
      )
      const store = renderHook(
        () => ({
          pluginActions: usePluginActions(),
          pluginConfig: usePluginConfig(),
        }),
        { wrapper }
      )

      act(() => {
        store.result.current.pluginActions.setPluginConfig([
          { id: "onePlugin", disabled: false },
          { id: "alert" },
          { id: "example app", disabled: true },
        ])
      })

      expect(store.result.current.pluginConfig).toEqual([
        { id: "alert" },
        { id: "onePlugin", disabled: false },
        { id: "example app", disabled: true },
      ])
    })
  })
})

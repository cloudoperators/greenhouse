/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import * as React from "react"
import { createPluginConfig, NAV_TYPES } from "./plugin"
import StoreProvider, { usePlugin } from "../components/StoreProvider"
import { renderHook, act } from "@testing-library/react"

describe("Plugin", () => {
  describe("createPluginConfig", () => {
    it("requires at least an id and name", () => {
      const spy = jest.spyOn(console, "warn").mockImplementation(() => {})

      createPluginConfig()
      createPluginConfig({ id: "test" })
      createPluginConfig({ name: "test" })

      expect(spy).toHaveBeenCalledTimes(3)
      expect(spy).toHaveBeenCalledWith(
        expect.stringContaining(
          "[greenhouse]::createPluginConfig: id and name are required."
        )
      )
      spy.mockRestore()
    })

    it("maps name to displayName if missing", () => {
      expect(createPluginConfig({ id: "id_test", name: "name_test" })).toEqual(
        expect.objectContaining({ displayName: "name_test" })
      )
    })

    it("sets weight to default 0 if missing", () => {
      expect(createPluginConfig({ id: "id_test", name: "name_test" })).toEqual(
        expect.objectContaining({ weight: 0 })
      )
    })

    it("sets version to latest if missing", () => {
      expect(createPluginConfig({ id: "id_test", name: "name_test" })).toEqual(
        expect.objectContaining({ version: "latest" })
      )
    })

    it("sets navigable to true if missing", () => {
      expect(createPluginConfig({ id: "id_test", name: "name_test" })).toEqual(
        expect.objectContaining({ navigable: true })
      )
    })

    it("sets navigation type to app", () => {
      expect(createPluginConfig({ id: "id_test", name: "name_test" })).toEqual(
        expect.objectContaining({ navType: NAV_TYPES.APP })
      )
    })

    it("adds id to the props", () => {
      expect(createPluginConfig({ id: "id_test", name: "name_test" })).toEqual(
        expect.objectContaining({
          props: expect.objectContaining({ id: "id_test" }),
        })
      )
    })

    it("does not save not known keys", () => {
      expect(
        createPluginConfig({
          id: "id_test",
          name: "name_test",
          miau: "bup",
        })
      ).not.toEqual(
        expect.objectContaining({
          miau: "bup",
        })
      )
    })

    it("creates a plugin", () => {
      const config = {
        id: "id_test",
        name: "name_test",
        displayName: "displayName_Test",
        version: "1.2.3",
        url: "http://miau.bup",
        weight: 9,
        navigable: false,
        navType: NAV_TYPES.MNG,
        props: {
          test1: "test1",
          test2: "test2",
        },
      }
      expect(createPluginConfig(config)).toEqual({
        ...config,
        props: { ...config.props, id: config.id },
      })
    })
  })

  describe("savePlugin", () => {
    describe("set active plugin", () => {
      it("keeps active plugin if existing in the config", () => {
        const wrapper = ({ children }) => (
          <StoreProvider>{children}</StoreProvider>
        )

        const store = renderHook(
          () => ({
            setActive: usePlugin().setActive,
            receiveConfig: usePlugin().receiveConfig,
            active: usePlugin().active(),
          }),
          { wrapper }
        )

        const configs = {
          plugin1: createPluginConfig({
            id: "plugin1",
            name: "plugin1",
            weight: 9,
          }),
          plugin2: createPluginConfig({
            id: "plugin2",
            name: "plugin2",
            weight: 0,
          }),
        }

        act(() => store.result.current.setActive(["plugin1"]))
        act(() => store.result.current.receiveConfig(configs))
        expect(store.result.current.active).toEqual(["plugin1"])
      })
      it("sets a new active plugin (from apps and not from mng) with the lowest weight", () => {
        const wrapper = ({ children }) => (
          <StoreProvider>{children}</StoreProvider>
        )

        const store = renderHook(
          () => ({
            receiveConfig: usePlugin().receiveConfig,
            active: usePlugin().active(),
          }),
          { wrapper }
        )

        const configs = {
          plugin0: createPluginConfig({
            id: "plugin0",
            name: "plugin0",
            weight: 0,
            navType: NAV_TYPES.MNG,
          }),
          plugin1: createPluginConfig({
            id: "plugin1",
            name: "plugin1",
            weight: 9,
          }),
          plugin2: createPluginConfig({
            id: "plugin2",
            name: "plugin2",
            weight: 1,
          }),
        }

        act(() => store.result.current.receiveConfig(configs))
        expect(store.result.current.active).toEqual(["plugin2"])
      })
    })
  })
})

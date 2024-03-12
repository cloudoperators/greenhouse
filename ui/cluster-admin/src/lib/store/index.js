import { createStore } from "zustand"
import { devtools } from "zustand/middleware"
import createGlobalsSlice from "./createGlobalsSlice"

export default () =>
  createStore(
    devtools((set, get) => ({
      ...createGlobalsSlice(set, get),
    }))
  )

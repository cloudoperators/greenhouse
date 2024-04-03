import { createContext, useContext } from "react"
import { useStore as create } from "zustand"
import createStore from "../lib/store"

const StoreContext = createContext()
const StoreProvider = ({ children }) => (
  <StoreContext.Provider value={createStore()}>
    {children}
  </StoreContext.Provider>
)

const useAppStore = (selector) => create(useContext(StoreContext), selector)

export const useGlobalsUrlStateKey = () =>
  useAppStore((state) => state.globals.urlStateKey)
export const useGlobalsActions = () =>
  useAppStore((state) => state.globals.actions)

export default StoreProvider

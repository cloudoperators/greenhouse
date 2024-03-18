import React, { useEffect } from "react"
import { Container } from "juno-ui-components"
import useAPI from "./hooks/useAPI"
import { usePluginConfig } from "./components/StoreProvider"
import PluginList from "./components/PluginList"
import PluginDetail from "./components/PluginDetail"

const AppContent = (props) => {
  const { getPlugins } = useAPI()
  const pluginConfig = usePluginConfig()

  useEffect(() => {
    if (!getPlugins) return
    const plugins = getPlugins()
    console.log("getPlugins", plugins)
  }, [getPlugins])

  return (
    <Container>
      <PluginDetail />
      <PluginList />
    </Container>
  )
}

export default AppContent

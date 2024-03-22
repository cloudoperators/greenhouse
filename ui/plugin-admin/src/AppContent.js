import React, { useEffect } from "react"
import { Container } from "juno-ui-components"
import useAPI from "./hooks/useAPI"
import PluginList from "./components/PluginList"
import PluginDetail from "./components/PluginDetail"

const AppContent = () => {
  const { getPlugins } = useAPI()

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

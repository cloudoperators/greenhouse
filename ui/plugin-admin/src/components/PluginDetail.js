import React, { useEffect, useState } from "react"
import {
  useShowDetailsFor,
  useGlobalsActions,
  usePluginConfig,
} from "./StoreProvider"
import {
  CodeBlock,
  Container,
  DataGrid,
  DataGridRow,
  DataGridCell,
  DataGridHeadCell,
  JsonViewer,
  Panel,
  Pill,
  PanelBody,
  Stack,
  Tabs,
  TabList,
  Tab,
  TabPanel,
} from "juno-ui-components"

const PluginDetail = () => {
  const pluginConfig = usePluginConfig()
  const { setShowDetailsFor } = useGlobalsActions()
  const showDetailsFor = useShowDetailsFor()
  const [plugin, setPlugin] = useState(null)

  useEffect(() => {
    if (!showDetailsFor || !pluginConfig) {
      return
    }
    setPlugin(pluginConfig.find((p) => p.id === showDetailsFor))
  }, [showDetailsFor, pluginConfig])

  const onPanelClose = () => {
    setShowDetailsFor(null)
  }

  return (
    <Panel opened={!!showDetailsFor} onClose={onPanelClose} size="large">
      <PanelBody>
        <Tabs>
          <TabList>
            <Tab>Details</Tab>
            <Tab>Raw Data</Tab>
          </TabList>
          <TabPanel>
            <Container px={false} py>
              <DataGrid
                columns={2}
                cellVerticalAlignment="top"
                gridColumnTemplate="35% auto"
              >
                <DataGridRow>
                  <DataGridHeadCell>Name</DataGridHeadCell>
                  <DataGridCell>{plugin?.name}</DataGridCell>
                </DataGridRow>

                <DataGridRow>
                  <DataGridHeadCell>Version</DataGridHeadCell>
                  <DataGridCell>{plugin?.version}</DataGridCell>
                </DataGridRow>

                <DataGridRow>
                  <DataGridHeadCell>Cluster</DataGridHeadCell>
                  <DataGridCell>{plugin?.clusterName}</DataGridCell>
                </DataGridRow>

                <DataGridRow>
                  <DataGridHeadCell>External Links</DataGridHeadCell>
                  <DataGridCell>
                    {plugin?.externalServicesUrls?.map((url) => {
                      return (
                        <p>
                          <a href={url.url} target="_blank" rel="noreferrer">
                            {url.name}
                          </a>
                        </p>
                      )
                    })}{" "}
                  </DataGridCell>
                </DataGridRow>

                <DataGridRow>
                  <DataGridHeadCell>Conditions</DataGridHeadCell>
                  <DataGridCell>
                    <Stack gap="2" alignment="start" wrap={true}>
                      {plugin?.statusConditions?.map((condition) => {
                        return (
                          <Pill
                            pillKey={condition.type}
                            pillKeyLabel={condition.type}
                            pillValue={condition.status}
                            pillValueLabel={condition.status}
                          />
                        )
                      })}
                    </Stack>
                  </DataGridCell>
                </DataGridRow>
                {plugin?.optionValues?.map((option) => {
                  if (option?.name.startsWith("greenhouse.")) return null

                  return (
                    <DataGridRow>
                      <DataGridHeadCell style={{ overflowWrap: "break-word" }}>
                        {option?.name}
                      </DataGridHeadCell>
                      <DataGridCell>
                        <CodeBlock>
                          <JsonViewer data={option?.value} expanded={true} />
                        </CodeBlock>
                      </DataGridCell>
                    </DataGridRow>
                  )
                })}
              </DataGrid>
            </Container>
          </TabPanel>

          <TabPanel>
            <Container px={false} py>
              <CodeBlock>
                <JsonViewer data={plugin?.raw} expanded={true} />
              </CodeBlock>
            </Container>
          </TabPanel>
        </Tabs>
      </PanelBody>
    </Panel>
  )
}

export default PluginDetail

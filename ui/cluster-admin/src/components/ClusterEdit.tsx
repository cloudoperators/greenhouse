import cluster from "cluster"
import useStore from "../store"

import {
  Container,
  Panel,
  PanelBody,
  TextInput,
  Form,
  FormRow,
  FormSection,
  Stack,
  Button,
} from "juno-ui-components"
import KeyValueInput from "./KeyValueInput"
import ResultMessageComponent, { ResultMessage } from "./ResultMessage"
import React from "react"
import { useClusterApi } from "../hooks/useClusterApi"

const ClusterEdit: React.FC<any> = () => {
  const clusterInEdit = useStore((state) => state.clusterInEdit)
  clusterInEdit?.spec
  const setClusterInEdit = useStore((state) => state.setClusterInEdit)

  const [submitMessage, setSubmitResultMessage] = React.useState<ResultMessage>(
    { message: "", ok: false }
  )

  const { updateCluster } = useClusterApi()

  const onPanelClose = () => {
    setClusterInEdit(undefined)
  }

  const setLabels = (labels: { [key: string]: string }) => {
    setClusterInEdit({
      ...clusterInEdit,
      metadata: {
        ...clusterInEdit?.metadata,
        labels: labels,
      },
    })
  }

  const onSubmit = async () => {
    let clusterUpdatePromise = updateCluster(clusterInEdit!)
    clusterUpdatePromise.then((response) => {
      setSubmitResultMessage(response)
    })
  }

  return (
    <Panel
      heading={clusterInEdit!.metadata?.name! || "Not found"}
      opened={!!clusterInEdit}
      onClose={onPanelClose}
      size="large"
    >
      <PanelBody>
        <Container px={false} py>
          <Form>
            <FormSection title="General">
              <FormRow>
                <TextInput
                  id="metadata.name"
                  label="Name"
                  placeholder="Name of this Cluster"
                  value={clusterInEdit!.metadata?.name}
                  disabled={true}
                />
              </FormRow>
              <FormRow>
                <TextInput
                  id="spec.accessmode"
                  label="Accessmode"
                  placeholder="Accessmode of this Cluster"
                  value={clusterInEdit!.spec?.accessMode}
                  disabled={true}
                />
              </FormRow>
              <FormRow>
                <KeyValueInput
                  data={clusterInEdit!.metadata?.labels}
                  setData={setLabels}
                  title="Labels"
                  dataName="Label"
                ></KeyValueInput>
              </FormRow>
            </FormSection>
            <FormSection>
              <Stack distribution="end" gap="2">
                {submitMessage.message != "" && (
                  <ResultMessageComponent submitMessage={submitMessage} />
                )}
                <Button onClick={onSubmit} variant="primary">
                  Update Cluster
                </Button>
              </Stack>
            </FormSection>
          </Form>
        </Container>
      </PanelBody>
    </Panel>
  )
}

export default ClusterEdit

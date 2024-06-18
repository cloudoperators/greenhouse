import { Button, Stack } from "juno-ui-components"
import { useActions } from "messages-provider"
import useSecretApi from "../hooks/useSecretApi"
import useStore from "../store"
import { Secret } from "../../../types/types"
import { base64DecodeSecretData, base64EncodeSecretData } from "./secretUtils"

const SecretFormButtons: React.FC = () => {
  const { createSecret, updateSecret, deleteSecret } = useSecretApi()
  const secretDetail = useStore((state) => state.secretDetail)
  const isSecretEditMode = useStore((state) => state.isSecretEditMode)
  const setShowSecretEdit = useStore((state) => state.setShowSecretEdit)
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const setIsSecretEditMode = useStore((state) => state.setIsSecretEditMode)
  const { addMessage } = useActions()

  const onDelete = async () => {
    let res = await deleteSecret(secretDetail!)
    addMessage({
      variant: res.ok ? "success" : "error",
      text: res.message,
    })
    // TODO: bind panel close to message dismiss once this is possible
    // close panel after 3sec if delete was successful
    if (res.ok) {
      setTimeout(() => {
        setShowSecretEdit(false)
        setSecretDetail(undefined)
        setIsSecretEditMode(false)
      }, 3000)
    }
  }
  const onSubmit = () => {
    let base64EncodedSecret = base64EncodeSecretData(secretDetail!)
    let secretCreatePromise = isSecretEditMode
      ? updateSecret(base64EncodedSecret)
      : createSecret(base64EncodedSecret)

    secretCreatePromise.then(async (res) => {
      addMessage({
        variant: res.ok ? "success" : "error",
        text: res.message,
      })
      if (res.ok) {
        setIsSecretEditMode(true)
        if (res.response) {
          setSecretDetail(base64DecodeSecretData(res.response))
        }
      }
    })
  }

  return (
    <Stack distribution="between">
      <Button onClick={onDelete} variant="primary-danger">
        Delete Secret
      </Button>

      <Button onClick={onSubmit} variant="primary">
        {isSecretEditMode ? "Update Secret" : "Create Secret"}
      </Button>
    </Stack>
  )
}

export default SecretFormButtons

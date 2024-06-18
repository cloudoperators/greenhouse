import { Messages } from "messages-provider"
const SecretFormHeader: React.FC = () => {
  return <Messages onDismiss={() => console.log("dismissed!")} />
}

export default SecretFormHeader

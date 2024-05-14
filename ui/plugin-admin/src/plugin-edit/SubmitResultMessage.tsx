/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Message } from "juno-ui-components"
export type SubmitMessage = {
  message: string
  ok: boolean
}

interface SubmitResultMessageProps {
  submitMessage: {
    message: string
    ok: boolean
  }
  onMessageDismiss?: () => void
}

const SubmitResultMessage: React.FC<SubmitResultMessageProps> = (
  props: SubmitResultMessageProps
) => {
  return (
    <Message
      autoDismissTimeout={3000}
      autoDismiss={props.submitMessage.ok}
      onDismiss={props.onMessageDismiss}
      variant={props.submitMessage.ok ? "success" : "error"}
      text={props.submitMessage.message}
    />
  )
}

export default SubmitResultMessage

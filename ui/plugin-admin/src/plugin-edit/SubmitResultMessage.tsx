/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Message } from "juno-ui-components"
export type SubmitMessage = {
  message: string
  ok: boolean
  variant?: "warning" | "success" | "error"
}

interface SubmitResultMessageProps {
  submitMessage: SubmitMessage
  onMessageDismiss?: () => void
}

const SubmitResultMessage: React.FC<SubmitResultMessageProps> = (
  props: SubmitResultMessageProps
) => {
  // if variant is not set, we deduct from ok
  if (!props.submitMessage.variant)
    props.submitMessage.variant = props.submitMessage.ok ? "success" : "error"
  return (
    <Message
      dismissible
      onDismiss={props.onMessageDismiss}
      variant={props.submitMessage.variant}
      text={props.submitMessage.message}
    />
  )
}

export default SubmitResultMessage

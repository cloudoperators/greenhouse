/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Messages } from "messages-provider"
const SecretFormHeader: React.FC = () => {
  return <Messages onDismiss={() => console.log("dismissed!")} />
}

export default SecretFormHeader

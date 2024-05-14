/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import useStore from "../plugindefinitions/store"

export const useSecretEditForm = () => {
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const secretDetail = useStore((state) => state.secretDetail)

  const handleSecretFormChange = (key, value: string) => {
    // key is in format name or dataKey.key or dataValue.key
    let keyInfo = key.split(".")
    let keyIdentifier = keyInfo[0]
    let keyData = keyInfo[1]

    switch (keyIdentifier) {
      case "name":
        setSecretDetail({
          ...secretDetail,
          metadata: {
            ...secretDetail?.metadata,
            name: value,
          },
        })
        break
      case "dataKey":
        // remove entry with old key and add new entry with new key
        let data = { ...secretDetail?.data }
        let dataValue = data[keyData]
        delete data[keyData]
        data[value] = dataValue
        setSecretDetail({
          ...secretDetail,
          data: data,
        })
        break
      case "dataValue":
        setSecretDetail({
          ...secretDetail,
          data: {
            ...secretDetail?.data,
            [keyData]: btoa(value),
          },
        })
        break
      default:
        console.log("keyIdentifier not found")
        break
    }
  }

  const deleteDataEntry = (key: string) => {
    let data = { ...secretDetail?.data }
    delete data[key]
    setSecretDetail({
      ...secretDetail,
      data: data,
    })
  }

  return { handleSecretFormChange, deleteDataEntry }
}

export default useSecretEditForm

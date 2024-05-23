/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect, useState } from "react"
import { registerConsumer } from "url-state-provider"
import useStore from "../store"

const DEFAULT_KEY = "greenhouse-secrets-admin"
const SECRET_DETAIL = "sd"
const SHOW_SECRET_EDIT = "sse"
const IS_SECRET_EDIT_MODE = "isem"

const useUrlState = (key: string): void => {
  const [isURLRead, setIsURLRead] = useState(false)
  const urlStateManager = registerConsumer(key || DEFAULT_KEY)

  // auth
  const loggedIn = useStore((state) => state.loggedIn)

  // globals
  const showSecretEdit = useStore((state) => state.showSecretEdit)
  const setShowSecretEdit = useStore((state) => state.setShowSecretEdit)
  const secretDetail = useStore((state) => state.secretDetail)
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const isSecretEditMode = useStore((state) => state.isSecretEditMode)
  const setIsSecretEditMode = useStore((state) => state.setIsSecretEditMode)

  // Set initial state from URL (on login)
  useEffect(() => {
    /* !!!IMPORTANT!!!
      don't read the url if we are already read it or if we are not logged in!!!!!
    */
    if (isURLRead || !loggedIn) return
    console.log(
      `${DEFAULT_KEY}: (${key || DEFAULT_KEY}) setting up state from url:`,
      urlStateManager.currentState()
    )

    // READ the url state and set the state
    const newSecretEdit = urlStateManager.currentState()?.[SHOW_SECRET_EDIT]
    const newSecretDetail = urlStateManager.currentState()?.[SECRET_DETAIL]
    const newIsSecretEditMode =
      urlStateManager.currentState()?.[IS_SECRET_EDIT_MODE]
    // SAVE the state
    if (newSecretEdit) setShowSecretEdit(newSecretEdit)
    if (newSecretDetail) setSecretDetail(newSecretDetail)
    if (newIsSecretEditMode) setIsSecretEditMode(newIsSecretEditMode)
    setIsURLRead(true)
  }, [loggedIn, setShowSecretEdit])

  // SYNC states to URL state
  useEffect(() => {
    // don't sync if we are not logged in OR URL ist not yet read
    if (!isURLRead || !loggedIn) return
    urlStateManager.push({
      [SHOW_SECRET_EDIT]: showSecretEdit,
      [SECRET_DETAIL]: secretDetail,
      [IS_SECRET_EDIT_MODE]: isSecretEditMode,
    })
  }, [loggedIn, showSecretEdit])
}

export default useUrlState

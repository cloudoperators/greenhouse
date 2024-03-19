import React from "react"
import useCommunication from "../hooks/useCommunication"
import useUrlState from "../hooks/useUrlState"

const AsyncWorker = () => {
  useCommunication()
  useUrlState()
  return null
}

export default AsyncWorker

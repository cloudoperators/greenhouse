import React from "react"
import useUrlState from "../hooks/useUrlState"

const AsyncWorker = ({ consumerId }) => {
  useUrlState(consumerId)
  return null
}

export default AsyncWorker

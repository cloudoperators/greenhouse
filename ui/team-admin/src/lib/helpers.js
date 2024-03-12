export const parseError = (error) => {
  let errMsg = error

  // check if error is JSON containing message or just string
  if (typeof error === "string") {
    errMsg = parseMessage(error)
  }

  // check if the error is a object containing message
  if (typeof error === "object") {
    console.log("Error parsing error message::object")
    if (error?.message) {
      errMsg = parseMessage(error?.message)
    }
  }
  return errMsg
}

const parseMessage = (message) => {
  let newMsg = message
  try {
    newMsg = JSON.parse(message)
    if (newMsg?.message) {
      newMsg = (newMsg?.code ? `${newMsg.code}, ` : "") + newMsg?.message
    }
  } catch (error) {}

  return newMsg
}

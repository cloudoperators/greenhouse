const path = require("path")
const fs = require("fs")

module.exports = ({ appPath = "" } = {}) => {
  const pkg = require(path.resolve(appPath, "package.json"))
  let secrets
  try {
    if (fs.existsSync(path.resolve(appPath, "secretProps.js"))) {
      secrets = require(path.resolve(appPath, "secretProps.js"))
    } else {
      secrets = require(path.resolve(appPath, "secretProps.json"))
    }
  } catch (e) {
    secrets = {}
  }

  const appProps = pkg.appProps || {}
  const props = {}
  for (let propName in appProps) {
    let value = appProps[propName]
    if (typeof value !== "string") value = appProps[propName].value
    props[propName] = value
  }

  for (let propName in secrets) {
    if (!props.hasOwnProperty(propName))
      throw Error(
        `Secret property ${propName} is not defined in package.json -> appProps`
      )
    props[propName] = secrets[propName]
  }

  return props
}

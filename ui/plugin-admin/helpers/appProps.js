/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

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

  const pkgAppProps = pkg.appProps || {}
  const pkgDependencyProps = pkg.appDependencies || {}
  const appProps = {}
  const dependencyProps = {}
  for (let propName in pkgAppProps) {
    // skip appDependencies
    if (propName === "appDependencies") return
    let value = pkgAppProps[propName]
    if (typeof value !== "string") value = pkgAppProps[propName].value
    appProps[propName] = value
  }

  // map pkg app props with the secret props
  for (let propName in secrets) {
    if (propName === "appDependencies") continue
    if (!appProps.hasOwnProperty(propName))
      throw Error(
        `Secret property ${propName} is not defined in package.json -> appProps`
      )
    appProps[propName] = secrets[propName]
  }

  if (secrets.appDependencies) {
    for (let propName in secrets.appDependencies) {
      if (!pkgDependencyProps.hasOwnProperty(propName))
        throw Error(
          `Secret property ${propName} is not defined in package.json -> appDependencies`
        )
      dependencyProps[propName] = secrets.appDependencies[propName]
    }
  }

  return { appProps, dependencyProps }
}

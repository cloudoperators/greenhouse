/*
 * Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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

/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

// opacity helper to make custom colors work with opacity
function withOpacity(variableName) {
  return ({ opacityVariable, opacityValue }) => {
    if (opacityValue !== undefined) {
      return `rgba(var(${variableName}), ${opacityValue})`
    }
    if (opacityVariable !== undefined) {
      return `rgba(var(${variableName}), var(${opacityVariable}, 1))`
    }
    return `rgb(var(${variableName}))`
  }
}

module.exports = {
  presets: [
    require("juno-ui-components/build/lib/tailwind.config"), // important, do not change
  ],
  prefix: "", // important, do not change
  content: ["./src/**/*.{js,jsx,ts,tsx}", "./public/index.html"],
  corePlugins: {
    preflight: false, // important, do not change
  },
  theme: {},
  plugins: [],
}

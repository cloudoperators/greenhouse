/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

module.exports = {
  transform: { "\\.[jt]sx?$": "babel-jest" },
  testEnvironment: "jsdom",
  setupFilesAfterEnv: ["<rootDir>/setupTests.js"],
  transformIgnorePatterns: [
    "node_modules/(?!(@cloudoperators/juno-ui-components|@cloudoperators/juno-url-state-router|@cloudoperators/juno-communicator|@cloudoperators/juno-oauth|@cloudoperators/juno-url-state-provider-v1|@cloudoperators/juno-messages-provider)/)",
  ],
  moduleNameMapper: {
    // Jest currently doesn't support resources with query parameters.
    // Therefore we add the optional query parameter matcher at the end
    // https://github.com/facebook/jest/issues/4181
    "\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)(\\?.+)?$":
      require.resolve("./__mocks__/fileMock"),
    "\\.(css|less|scss)$": require.resolve("./__mocks__/styleMock"),
  },
}

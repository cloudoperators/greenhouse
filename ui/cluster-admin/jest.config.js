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

module.exports = {
  transform: { "\\.[jt]sx?$": "babel-jest" },
  testEnvironment: "jsdom",
  setupFilesAfterEnv: ["<rootDir>/setupTests.js"],
  transformIgnorePatterns: [
    "node_modules/(?!(juno-ui-components|url-state-router|communicator|oauth|url-state-provider|messages-provider|policy-engine)/)",
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

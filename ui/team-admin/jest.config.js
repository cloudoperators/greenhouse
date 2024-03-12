module.exports = {
  transform: { "\\.[jt]sx?$": "babel-jest" },
  testEnvironment: "jsdom",
  setupFilesAfterEnv: ["<rootDir>/setupTests.js"],
  transformIgnorePatterns: [
    "node_modules/(?!(juno-ui-components|url-state-router|communicator|oauth|url-state-provider|messages-provider|policy-engine|utils)/)",
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

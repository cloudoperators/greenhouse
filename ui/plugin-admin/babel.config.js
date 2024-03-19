module.exports = {
  env: {
    test: {
      presets: ["@babel/preset-env", ["@babel/preset-react", {"runtime": "automatic"}],'@babel/preset-typescript'],
      plugins: [["babel-plugin-transform-import-meta", { module: "ES6" }]],
    },
  },
}

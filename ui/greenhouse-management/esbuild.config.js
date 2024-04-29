/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

const esbuild = require("esbuild")
const fs = require("node:fs/promises")
const pkg = require("./package.json")
const postcss = require("postcss")
const sass = require("sass")
const { transform } = require("@svgr/core")
const url = require("postcss-url")
// this function generates app props based on package.json and propSecrets.json
const appProps = require("../../helpers/appProps")

if (!/.+\/.+\.js/.test(pkg.module))
  throw new Error(
    "module value is incorrect, use DIR/FILE.js like build/index.js"
  )

const isProduction = process.env.NODE_ENV === "production"
// If the jspm server fails and we cannot use external packages
// in our import map then IGNORE_EXTERNALS (global env variable)
// should be set to true
const IGNORE_EXTERNALS = process.env.IGNORE_EXTERNALS === "true"
// in dev environment we prefix output file with public
let outfile = `${isProduction ? "" : "public/"}${pkg.main || pkg.module}`
// get output from outputfile
let outdir = outfile.slice(0, outfile.lastIndexOf("/"))
const args = process.argv.slice(2)
const watch = args.indexOf("--watch") >= 0
const serve = args.indexOf("--serve") >= 0

// helpers for console log
const green = "\x1b[32m%s\x1b[0m"
const yellow = "\x1b[33m%s\x1b[0m"
const clear = "\033c"

const build = async () => {
  // delete build folder and re-create it as an empty folder
  await fs.rm(outdir, { recursive: true, force: true })
  await fs.mkdir(outdir, { recursive: true })

  // build app
  let ctx = await esbuild.context({
    bundle: true,
    minify: isProduction,
    // target: ["es2020"],
    target: ["es2020"], //["chrome64", "firefox67", "safari11.1", "edge79"],
    format: "esm",
    platform: "browser",
    // built-in loaders: js, jsx, ts, tsx, css, json, text, base64, dataurl, file, binary
    loader: { ".js": "jsx" },
    sourcemap: !isProduction,
    // here we exclude package from bundle which are defined in peerDependencies
    // our importmap generator uses also the peerDependencies to create the importmap
    // it means all packages defined in peerDependencies are in browser available via the importmap
    external:
      isProduction && !IGNORE_EXTERNALS
        ? Object.keys(pkg.peerDependencies || {})
        : [],
    entryPoints: [pkg.source],
    outdir,
    // this step is important for performance reason.
    // the main file (index.js) contains minimal code needed to
    // load the app via dynamic import (splitting: true)
    splitting: true,
    // we suport only esm!
    format: "esm",
    plugins: [
      // minimal plugin to log the recompiling process.
      {
        name: "start/end",
        setup(build) {
          build.onStart(() => {
            console.log(clear)
            console.log(yellow, "Compiling...")
          })
          build.onEnd(() => console.log(green, "Done!"))
        },
      },

      // this custom plugin rewrites SVG imports to
      // dataurls, paths or react components based on the
      // search param and size
      {
        name: "svg-loader",
        setup(build) {
          build.onLoad(
            // consider only .svg files
            { filter: /.\.(svg)$/, namespace: "file" },
            async (args) => {
              let contents = await fs.readFile(args.path)
              // built-in loaders: js, jsx, ts, tsx, css, json, text, base64, dataurl, file, binary
              let loader = "text"
              if (args.suffix === "?url") {
                // as URL
                const maxSize = 10240 // 10Kb
                // use dataurl loader for small files and file loader for big files!
                loader = contents.length <= maxSize ? "dataurl" : "file"
              } else {
                // as react component
                // use react component loader (jsx)
                loader = "jsx"
                contents = await transform(contents, {
                  plugins: ["@svgr/plugin-jsx"],
                })
              }

              return { contents, loader }
            }
          )
        },
      },

      // this custom plugin rewrites image imports to
      // dataurls or urls based on the size
      {
        name: "image-loader",
        setup(build) {
          build.onLoad(
            // consider only .svg files
            { filter: /.\.(png|jpg|jpeg|gif)$/, namespace: "file" },
            async (args) => {
              let contents = await fs.readFile(args.path)
              const maxSize = 10240 // 10Kb
              // built-in loaders: js, jsx, ts, tsx, css, json, text, base64, dataurl, file, binary
              // use dataurl loader for small files and file loader for big files!
              loader = contents.length <= maxSize ? "dataurl" : "file"

              return { contents, loader }
            }
          )
        },
      },

      // this custom plugin parses the style files
      {
        name: "parse-styles",
        setup(build) {
          build.onLoad(
            // consider only .scss and .css files
            { filter: /.\.(css|scss)$/, namespace: "file" },
            async (args) => {
              let content
              // handle scss, convert to css
              if (args.path.endsWith(".scss")) {
                const result = sass.renderSync({ file: args.path })
                content = result.css
              } else {
                // read file content
                content = await fs.readFile(args.path)
              }

              // postcss plugins
              const plugins = [
                require("tailwindcss"),
                require("autoprefixer"),
                // rewrite urls inside css
                url({
                  url: "inline",
                  // maxSize: 10, // use dataurls if files are smaller than 10k
                  // fallback: "copy", // if files are bigger use copy method
                  // assetsPath: "./build/assets",
                  // useHash: true,
                  // optimizeSvgEncode: true,
                }),
              ]

              const { css } = await postcss(plugins).process(content, {
                from: args.path,
                to: outdir,
              })
              // built-in loaders: js, jsx, ts, tsx, css, json, text, base64, dataurl, file, binary
              return { contents: css, loader: "text" }
            }
          )
        },
      },
    ],
  })

  // watch and serve
  if (watch || serve) {
    if (watch) await ctx.watch()
    if (serve) {
      // generate app props based on package.json and secretProps.json
      await fs.writeFile(
        `./${outdir}/appProps.js`,
        `export default ${JSON.stringify(appProps())}`
      )

      let { host, port } = await ctx.serve({
        host: "0.0.0.0",
        port: parseInt(process.env.APP_PORT || process.env.PORT || 3000),
        servedir: "public",
      })
      console.log("serve on", `${host}:${port}`)
    }
  } else {
    await ctx.rebuild()
    await ctx.dispose()
  }
}

build()

// Bundles the web server into dist/server.mjs, beside the renderer Vite
// built in dist/ssr and the browser's files in dist/client. A release ships
// dist as it is, with its own Node, and needs nothing installed.

import * as esbuild from "esbuild";

await esbuild.build({
  entryPoints: ["server/server.ts"],
  outfile: "dist/server.mjs",
  bundle: true,
  platform: "node",
  format: "esm",
  target: "node24",
  minify: false,
  legalComments: "none",
  define: { __RELEASE__: JSON.stringify(process.env.ORIEL_RELEASE ?? "dev") },
  logLevel: "warning",
});

/// <reference types="vitest/config" />
import { fileURLToPath, URL } from "node:url";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

// One configuration builds both halves of the site: the browser's bundle in
// dist/client, and the server renderer in dist/ssr. scripts/bundle-server.mjs
// then wraps the renderer in the small Node server that ships in a release.
export default defineConfig(({ isSsrBuild }) => ({
  plugins: [vue()],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  define: {
    __RELEASE__: JSON.stringify(process.env.ORIEL_RELEASE ?? "dev"),
  },
  build: {
    outDir: isSsrBuild ? "dist/ssr" : "dist/client",
    // The server reads which scripts and styles each page's components
    // need, so it can ask the browser for them early.
    ssrManifest: !isSsrBuild,
    manifest: !isSsrBuild,
    sourcemap: false,
    // Pictures and fonts stay files with fingerprinted names, never inlined.
    assetsInlineLimit: 0,
  },
  // The renderer carries every package it uses, so a release needs no
  // node_modules on the server.
  ssr: { noExternal: true },
  test: {
    environment: "happy-dom",
    include: ["src/**/*.test.ts", "server/**/*.test.ts"],
    restoreMocks: true,
  },
}));

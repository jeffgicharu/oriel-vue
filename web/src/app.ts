import { createPinia } from "pinia";
import { createSSRApp } from "vue";
import { apiKey, type Api } from "./api/client";
import App from "./App.vue";
import { makeRouter } from "./router";
import { installDeskGuards } from "./router/guards";

/**
 * Makes one Oriel app: its own router, its own stores and its own API
 * client. The server makes a fresh one for every request, so nothing one
 * visitor's page holds can reach another's.
 */
export function createOriel(api: Api, ssr: boolean) {
  const app = createSSRApp(App);
  const pinia = createPinia();
  const router = makeRouter(ssr);
  // Before the router starts, so the first page is guarded too.
  installDeskGuards(router, pinia);
  app.provide(apiKey, api);
  app.use(pinia);
  app.use(router);
  return { app, pinia, router };
}

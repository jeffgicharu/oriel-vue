import { createHead } from "@unhead/vue/client";
import { parse } from "devalue";
import { createApi } from "./api/client";
import { createOriel } from "./app";
import { recordArrival } from "./lib/leadSource";
import "./styles/site.css";

const api = createApi({ baseUrl: window.location.origin });
const { app, pinia, router } = createOriel(api, false);
app.use(createHead());

// The stores as the server left them, so the page picks up where the
// server's render ended instead of asking the API again.
const state = document.getElementById("oriel-state")?.textContent;
if (state) pinia.state.value = parse(state);

recordArrival(new URL(window.location.href), document.referrer);

void router.isReady().then(() => app.mount("#app"));

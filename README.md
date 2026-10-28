# Oriel, in Vue

The sales website for Oriel, an apartment tower of 86 homes sold off-plan in Riverside, Nairobi, built with Vue 3 and a Go API. Buyers turn the building in 3D and pick a home floor by floor, filter the homes, open each one's plan, light study and payment schedule, shortlist and compare them, book viewings, ask questions by form or on WhatsApp, and reserve a home, whose page then changes live when the sales team decides. Every page is rendered on the server.

It is a Vue 3 version of [Oriel](https://github.com/jeffgicharu/oriel). The development, its company and its sales team are invented, and every home, price and sale is sample data.

Live at **https://oriel-vue.jeffgicharu.com**.

## How it fits together

```
browser ──► nginx ──► web server (Node, Vue rendered on the server) ──► API (Go) ──► SQLite
                 └──► /api ────────────────────────────────────────────►┘
```

- **The web server** renders every page with Vue on the server and sends it whole, with its title, description, canonical address, share tags and structured data, so crawlers and link previews need no script. The browser then takes over as a single-page app. It holds no data and no secrets, and talks to its own API on 127.0.0.1.
- **The API** is a REST JSON API in Go. It owns the homes, prices, payment plans, viewings and enquiries, the PDFs and the construction programme. Every visitor reads a shared sample until they first send something; from then on they have a private copy of the database, so nothing one visitor does is seen by another.
- **The contract** between them is [`api/openapi.yaml`](api/openapi.yaml). The TypeScript client's types are generated from it, and a Go test checks every answer the API gives against it, so the two cannot drift apart.

## The web app

Vue 3 with `<script setup>` and the Composition API, strict TypeScript, Pinia, Vue Router and Vite. Server rendering is set up directly on Vite rather than through a framework, so the pieces are in plain view:

| Part | Where |
|---|---|
| A fresh app, router, Pinia and head for every request | [`src/app.ts`](web/src/app.ts), [`src/entry-server.ts`](web/src/entry-server.ts) |
| The web server: cookie allow-list, time limits, real 404 and 503 pages, no-store on every page | [`server/server.ts`](web/server/server.ts), [`server/http.ts`](web/server/http.ts) |
| Typed routes, scroll behaviour, a redirect in a guard | [`src/router/`](web/src/router) |
| Pinia setup stores per domain | [`src/stores/`](web/src/stores) |
| The typed API client and the state of each read | [`src/api/`](web/src/api) |
| Filters kept in the address | [`src/composables/useHomeFilters.ts`](web/src/composables/useHomeFilters.ts), [`src/lib/filters.ts`](web/src/lib/filters.ts) |
| Forms: the browser's checks, the API's 422 field errors beside their fields | [`src/composables/useForm.ts`](web/src/composables/useForm.ts) |
| Loading, empty, error and retry for every read | [`src/components/RequestView.vue`](web/src/components/RequestView.vue) |
| Floor plans drawn by Vue from the API's geometry | [`src/lib/floorplan.ts`](web/src/lib/floorplan.ts), [`src/components/FloorPlan.vue`](web/src/components/FloorPlan.vue) |
| Where each visitor came from (campaign and referrer) | [`src/lib/leadSource.ts`](web/src/lib/leadSource.ts) |
| WhatsApp enquiries that name the home | [`src/lib/whatsapp.ts`](web/src/lib/whatsapp.ts) |

The stores' state is carried from the server's render to the browser with `devalue`, in a JSON data block escaped so nothing in it can close its element. No inline script is allowed by the pages' Content Security Policy.

### Where enquiries come from

The browser keeps two notes of how a visitor arrived: their first visit, kept for 90 days, and the visit that led to what they sent. Each holds the campaign named in the link (`utm_source`, `utm_medium`, `utm_campaign`), the site that sent them and the page they landed on, and nothing that identifies them. The notes travel with every enquiry, viewing and WhatsApp tap. The API works out the channel itself (search, paid search, social, paid social, property portal, email, WhatsApp, another website or direct) and keeps it with the lead for the sales team.

### WhatsApp

Each home's page and the enquiry page offer a WhatsApp link with a message already written, naming the home, its price and its address. The site has no WhatsApp number of its own, so the link carries none and WhatsApp asks whom to send it to; a business would set its own in `WHATSAPP_NUMBER`. Each tap is recorded with where the visitor came from.

### The building in 3D

The explorer and the home page's opening draw the tower with three.js from the same model the API uses for the plans and prices. Before any 3D code is fetched, the page asks the browser for its graphics renderer: a device without a graphics chip (SwiftShader, llvmpipe) or with data saving turned on gets the flat drawing instead and never downloads three.js. The scene draws only when something moves, and its geometry, textures and listeners are disposed of when the page is left ([`src/three/`](web/src/three)).

### Reservations

A buyer can ask to reserve an available home. The home is held for 72 hours, and the buyer's reservation page listens to the API's server-sent events ([`useLiveEvents`](web/src/composables/useLiveEvents.ts)): when the sales team confirms or declines, the page changes without a reload, and the letter of offer can be downloaded as a PDF.

## The API

Go, with the standard library's router, SQLite through the pure-Go `modernc.org/sqlite`, and `gopdf` for the price list. The domain packages come from Oriel: the building's geometry, the floor plans, the light studies, the payment plans, the construction programme, the pricing and the sample sales history.

- Errors are problem documents (RFC 9457); a 422 names each field that needs another look.
- Forms are accepted only from the site's own pages (`http.CrossOriginProtection`), as JSON of a bounded size, with unknown fields refused.
- Forms and new private copies are limited per visitor address; the address is believed from a header only when the request comes from this machine.
- Live updates are server-sent events, with a heartbeat and a cap on how many are open at once.
- The sample is rebuilt every hour with its dates moved on, so today's viewings and this month's sales are always current.

## Tests

- **Go:** the domain packages' tests, and a contract test that calls every operation with good and bad input and checks each answer against `openapi.yaml`.
- **Vitest:** stores, composables and the pure modules, the web server's decisions, and server rendering itself: two visitors' pages rendered at the same moment, fifty times, never show each other's data.
- **Playwright:** the main journeys in Chromium, Firefox, WebKit and a phone: filters in the address, a home's plan and payment plans, booking and cancelling a viewing, asking a question, the API's field errors, the shortlist and comparison, and pages read with scripts turned off.
- **axe:** every page checked against WCAG 2.2 AA, with no failures allowed.
- **The release gate** (`api/internal/check`) runs on the server against a new release before any visitor reaches it, and again through nginx after: every page, its head tags, its noindex and no-store headers, its files and links, and the API's journeys in a throwaway copy.

## Running it

```sh
# The API
cd api
go test ./...
ORIEL_API_PORT=8702 ORIEL_DATA=./data ORIEL_SITE_URL=http://127.0.0.1:8700 \
  ORIEL_SECRET=$(openssl rand -hex 32) ORIEL_DESK_KEY=$(openssl rand -hex 16) \
  go run ./cmd/oriel-vue-api serve

# The web app (in another terminal)
cd web
pnpm install
go run ../api/cmd/oriel-vue-api drawings src/generated   # the drawings, made from the building
pnpm test && pnpm build
ORIEL_WEB_PORT=8700 ORIEL_API_PORT=8702 ORIEL_SITE_URL=http://127.0.0.1:8700 ORIEL_PROXY_API=1 node dist/server.mjs
```

`pnpm api-types` regenerates the client's types after a change to `openapi.yaml`.

## Deploying

Every push to `main` is tested, built and walked in CI, then uploaded and switched in by blue/green slots: a release starts beside the live one, passes the gate there, and only then does nginx send visitors to it. If the check through nginx fails, nginx goes back. The release carries its own Node, so the server needs nothing installed. The server files are in [`deploy/`](deploy).

## Layout

```
api/
  cmd/oriel-vue-api/   the API, the release gate, the desk links and the drawings
  internal/api/        the HTTP handlers, and the contract test
  internal/attribution where a visitor came from, as a channel
  internal/check/      the release gate
  internal/...         the domain: building, floor plans, light, plans, programme, pricing, sample, store, PDFs
  openapi.yaml         the contract
web/
  server/              the Node web server
  src/                 the Vue app
  e2e/                 the Playwright journeys and the accessibility checks
deploy/                provisioning, systemd units, nginx and the activate script
```

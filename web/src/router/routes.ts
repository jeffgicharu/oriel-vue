import type { RouteRecordInfo, RouteRecordRaw } from "vue-router";

// Every page is its own chunk, fetched when it is first visited. The server
// renders the page that was asked for and the browser picks it up from there.

export const routes: RouteRecordRaw[] = [
  { path: "/", name: "home", component: () => import("@/pages/HomePage.vue") },
  { path: "/residences", name: "residences", component: () => import("@/pages/ResidencesPage.vue"), meta: { nav: "residences" } },
  // A home's number is its level and position: four digits.
  { path: "/residences/:id(\\d{4})", name: "home-detail", component: () => import("@/pages/HomeDetailPage.vue"), meta: { nav: "residences" } },
  {
    path: "/residences/types/:code([A-Da-dPp])",
    name: "home-type",
    component: () => import("@/pages/TypePage.vue"),
    meta: { nav: "residences" },
    // A type is always written in capitals: /residences/types/b moves to /B.
    beforeEnter: (to) => {
      const code = "code" in to.params ? String(to.params.code) : "";
      if (code !== code.toUpperCase()) return { name: "home-type", params: { code: code.toUpperCase() }, replace: true };
    },
  },
  { path: "/shortlist", name: "shortlist", component: () => import("@/pages/ShortlistPage.vue") },
  { path: "/compare", name: "compare", component: () => import("@/pages/ComparePage.vue") },
  { path: "/building", name: "building", component: () => import("@/pages/BuildingPage.vue"), meta: { nav: "building" } },
  { path: "/location", name: "location", component: () => import("@/pages/LocationPage.vue"), meta: { nav: "location" } },
  { path: "/progress", name: "progress", component: () => import("@/pages/ProgressPage.vue"), meta: { nav: "progress" } },
  { path: "/buying", name: "buying", component: () => import("@/pages/BuyingPage.vue"), meta: { nav: "buying" } },
  { path: "/viewings/new", name: "viewing-new", component: () => import("@/pages/ViewingNewPage.vue") },
  { path: "/viewings/:ref([A-Za-z0-9-]{3,20})", name: "viewing", component: () => import("@/pages/ViewingPage.vue") },
  { path: "/enquire", name: "enquire", component: () => import("@/pages/EnquirePage.vue") },
  { path: "/reserve/:id(\\d{4})", name: "reserve", component: () => import("@/pages/ReservePage.vue") },
  { path: "/reservations/:ref([A-Za-z0-9-]{3,20})", name: "reservation", component: () => import("@/pages/ReservationPage.vue") },
  { path: "/privacy", name: "privacy", component: () => import("@/pages/PrivacyPage.vue") },
  { path: "/terms", name: "terms", component: () => import("@/pages/TermsPage.vue") },
  { path: "/:pathMatch(.*)*", name: "not-found", component: () => import("@/pages/NotFoundPage.vue") },
];

// The routes by name, with their parameters, so a link to a home that leaves
// out its number, or to a route that does not exist, fails to compile.
type NoParams = Record<never, never>;
export interface RouteNamedMap {
  home: RouteRecordInfo<"home", "/", NoParams, NoParams>;
  residences: RouteRecordInfo<"residences", "/residences", NoParams, NoParams>;
  "home-detail": RouteRecordInfo<"home-detail", "/residences/:id", { id: string }, { id: string }>;
  "home-type": RouteRecordInfo<"home-type", "/residences/types/:code", { code: string }, { code: string }>;
  shortlist: RouteRecordInfo<"shortlist", "/shortlist", NoParams, NoParams>;
  compare: RouteRecordInfo<"compare", "/compare", NoParams, NoParams>;
  building: RouteRecordInfo<"building", "/building", NoParams, NoParams>;
  location: RouteRecordInfo<"location", "/location", NoParams, NoParams>;
  progress: RouteRecordInfo<"progress", "/progress", NoParams, NoParams>;
  buying: RouteRecordInfo<"buying", "/buying", NoParams, NoParams>;
  "viewing-new": RouteRecordInfo<"viewing-new", "/viewings/new", NoParams, NoParams>;
  viewing: RouteRecordInfo<"viewing", "/viewings/:ref", { ref: string }, { ref: string }>;
  enquire: RouteRecordInfo<"enquire", "/enquire", NoParams, NoParams>;
  reserve: RouteRecordInfo<"reserve", "/reserve/:id", { id: string }, { id: string }>;
  reservation: RouteRecordInfo<"reservation", "/reservations/:ref", { ref: string }, { ref: string }>;
  privacy: RouteRecordInfo<"privacy", "/privacy", NoParams, NoParams>;
  terms: RouteRecordInfo<"terms", "/terms", NoParams, NoParams>;
  "not-found": RouteRecordInfo<"not-found", "/:pathMatch(.*)*", { pathMatch: string | string[] }, { pathMatch: string | string[] }>;
}

declare module "vue-router" {
  interface TypesConfig {
    RouteNamedMap: RouteNamedMap;
  }
  interface RouteMeta {
    /** The main navigation item the page belongs to. */
    nav?: string;
  }
}

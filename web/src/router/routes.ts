import type { RouteRecordInfo, RouteRecordRaw } from "vue-router";

// Every page is its own chunk, fetched when it is first visited. The server
// renders the page that was asked for and the browser picks it up from there.

export const routes: RouteRecordRaw[] = [
  { path: "/", name: "home", component: () => import("@/pages/HomePage.vue"), meta: { headerOver: true } },
  { path: "/explore", name: "explore", component: () => import("@/pages/ExplorePage.vue"), meta: { nav: "explore" } },
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
  // The sales desk: the team's own pages, behind its sign-in.
  { path: "/desk/sign-in", name: "desk-sign-in", component: () => import("@/pages/desk/SignInPage.vue"), meta: { desk: true } },
  { path: "/desk", name: "desk", component: () => import("@/pages/desk/DashboardPage.vue"), meta: { desk: true } },
  { path: "/desk/stack", name: "desk-stack", component: () => import("@/pages/desk/StackPage.vue"), meta: { desk: true } },
  // A home's drawer opens on the stacking plan; the activity log links to it by the home.
  {
    path: "/desk/homes/:id(\\d{4})",
    redirect: (to) => ({ name: "desk-stack", query: { home: "id" in to.params ? String(to.params.id) : "" } }),
  },
  { path: "/desk/leads", name: "desk-leads", component: () => import("@/pages/desk/LeadsPage.vue"), meta: { desk: true } },
  { path: "/desk/leads/new", name: "desk-lead-new", component: () => import("@/pages/desk/NewLeadPage.vue"), meta: { desk: true } },
  { path: "/desk/leads/:id(\\d{1,9})", name: "desk-lead", component: () => import("@/pages/desk/LeadPage.vue"), meta: { desk: true } },
  { path: "/desk/viewings", name: "desk-viewings", component: () => import("@/pages/desk/ViewingsPage.vue"), meta: { desk: true } },
  { path: "/desk/reservations", name: "desk-reservations", component: () => import("@/pages/desk/ReservationsPage.vue"), meta: { desk: true } },
  { path: "/desk/pricing", name: "desk-pricing", component: () => import("@/pages/desk/PricingPage.vue"), meta: { desk: true, manager: true } },
  { path: "/desk/activity", name: "desk-activity", component: () => import("@/pages/desk/ActivityPage.vue"), meta: { desk: true } },
  { path: "/:pathMatch(.*)*", name: "not-found", component: () => import("@/pages/NotFoundPage.vue") },
];

// The routes by name, with their parameters, so a link to a home that leaves
// out its number, or to a route that does not exist, fails to compile.
type NoParams = Record<never, never>;
export interface RouteNamedMap {
  home: RouteRecordInfo<"home", "/", NoParams, NoParams>;
  explore: RouteRecordInfo<"explore", "/explore", NoParams, NoParams>;
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
  "desk-sign-in": RouteRecordInfo<"desk-sign-in", "/desk/sign-in", NoParams, NoParams>;
  desk: RouteRecordInfo<"desk", "/desk", NoParams, NoParams>;
  "desk-stack": RouteRecordInfo<"desk-stack", "/desk/stack", NoParams, NoParams>;
  "desk-leads": RouteRecordInfo<"desk-leads", "/desk/leads", NoParams, NoParams>;
  "desk-lead-new": RouteRecordInfo<"desk-lead-new", "/desk/leads/new", NoParams, NoParams>;
  "desk-lead": RouteRecordInfo<"desk-lead", "/desk/leads/:id", { id: string }, { id: string }>;
  "desk-viewings": RouteRecordInfo<"desk-viewings", "/desk/viewings", NoParams, NoParams>;
  "desk-reservations": RouteRecordInfo<"desk-reservations", "/desk/reservations", NoParams, NoParams>;
  "desk-pricing": RouteRecordInfo<"desk-pricing", "/desk/pricing", NoParams, NoParams>;
  "desk-activity": RouteRecordInfo<"desk-activity", "/desk/activity", NoParams, NoParams>;
  "not-found": RouteRecordInfo<"not-found", "/:pathMatch(.*)*", { pathMatch: string | string[] }, { pathMatch: string | string[] }>;
}

declare module "vue-router" {
  interface TypesConfig {
    RouteNamedMap: RouteNamedMap;
  }
  interface RouteMeta {
    /** The main navigation item the page belongs to. */
    nav?: string;
    /** The header lies over the page's opening picture. */
    headerOver?: boolean;
    /** A page of the sales desk: it has the desk's frame and needs a signed-in member of staff. */
    desk?: boolean;
    /** Only the sales manager may open it. */
    manager?: boolean;
  }
}

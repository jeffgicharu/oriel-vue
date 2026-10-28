// The residences list's filters. They live in the address, so a filtered list
// can be shared, bookmarked and reached with the back button. Anything in the
// address that is not a known filter is dropped.

import type { LocationQuery, LocationQueryRaw } from "vue-router";

export type Floors = "" | "lower" | "middle" | "upper";
export type Side = "" | "river" | "garden";
export type Sort = "price" | "price-desc" | "floor" | "size";

export interface HomeFilter {
  beds: number; // 0 for any
  max: number; // highest price in KES, 0 for any
  floors: Floors;
  side: Side;
  all: boolean; // include homes no longer available
  sort: Sort;
}

export const defaultFilter: Readonly<HomeFilter> = Object.freeze({ beds: 0, max: 0, floors: "", side: "", all: false, sort: "price" });

/** The choices for the highest price. */
export const priceSteps = [12_000_000, 15_000_000, 20_000_000, 25_000_000, 30_000_000, 40_000_000, 60_000_000];

export const sortNames: Record<Sort, string> = {
  price: "Price, low to high",
  "price-desc": "Price, high to low",
  floor: "Floor, highest first",
  size: "Size, largest first",
};

export const floorNames: Record<Exclude<Floors, "">, string> = {
  lower: "Levels 2 to 6",
  middle: "Levels 7 to 11",
  upper: "Levels 12 to 16",
};

function first(v: LocationQuery[string] | undefined): string {
  return (Array.isArray(v) ? v[0] : v) ?? "";
}

export function parseFilter(query: LocationQuery): HomeFilter {
  const f: HomeFilter = { ...defaultFilter };
  const beds = Number(first(query.beds));
  if (Number.isInteger(beds) && beds >= 1 && beds <= 4) f.beds = beds;
  const max = Number(first(query.max));
  if (Number.isInteger(max) && max > 0) f.max = max;
  const floors = first(query.floors);
  if (floors === "lower" || floors === "middle" || floors === "upper") f.floors = floors;
  const side = first(query.side);
  if (side === "river" || side === "garden") f.side = side;
  f.all = first(query.show) === "all";
  const sort = first(query.sort);
  if (sort === "price-desc" || sort === "floor" || sort === "size") f.sort = sort;
  return f;
}

/** The filters as the API's query, leaving out the defaults. */
export function filterQuery(f: HomeFilter) {
  return {
    ...(f.beds ? { beds: f.beds } : {}),
    ...(f.max ? { max: f.max } : {}),
    ...(f.floors ? { floors: f.floors } : {}),
    ...(f.side ? { side: f.side } : {}),
    ...(f.all ? { show: "all" as const } : {}),
    ...(f.sort !== "price" ? { sort: f.sort } : {}),
  };
}

/** The filters as the address's query. */
export function filterRouteQuery(f: HomeFilter): LocationQueryRaw {
  const out: LocationQueryRaw = {};
  for (const [k, v] of Object.entries(filterQuery(f))) out[k] = String(v);
  return out;
}

/** Does any filter narrow the list? */
export function isNarrowed(f: HomeFilter): boolean {
  return f.beds > 0 || f.max > 0 || f.floors !== "" || f.side !== "" || f.all;
}

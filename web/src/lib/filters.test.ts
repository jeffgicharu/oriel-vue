import { describe, expect, it } from "vitest";
import { filterQuery, filterRouteQuery, isNarrowed, parseFilter } from "./filters";

describe("the residences filters in the address", () => {
  it("reads what it knows and drops the rest", () => {
    expect(parseFilter({ beds: "2", max: "25000000", floors: "upper", side: "river", show: "all", sort: "floor", junk: "x" })).toEqual({
      beds: 2,
      max: 25_000_000,
      floors: "upper",
      side: "river",
      all: true,
      sort: "floor",
    });
    expect(parseFilter({ beds: "9", max: "-4", floors: "attic", sort: "random" })).toEqual({ beds: 0, max: 0, floors: "", side: "", all: false, sort: "price" });
  });

  it("writes back only what differs from the defaults", () => {
    expect(filterRouteQuery(parseFilter({ beds: "3" }))).toEqual({ beds: "3" });
    expect(filterQuery(parseFilter({}))).toEqual({});
  });

  it("knows when the list is narrowed", () => {
    expect(isNarrowed(parseFilter({ sort: "size" }))).toBe(false);
    expect(isNarrowed(parseFilter({ side: "garden" }))).toBe(true);
  });
});

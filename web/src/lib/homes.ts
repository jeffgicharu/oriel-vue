import type { Home, Status } from "@/api/client";

const statusNames: Record<Status, string> = {
  available: "Available",
  held: "On hold",
  reserved: "Reserved",
  sold: "Sold",
  withheld: "Not yet released",
};

/** A status as a buyer reads it. */
export function statusName(status: Status): string {
  return statusNames[status] ?? status;
}

/** Does the public site show this home's price? */
export function showsPrice(h: Pick<Home, "status">): boolean {
  return h.status === "available" || h.status === "held";
}

/** "North and west" from "north and west". */
export function capitalise(s: string): string {
  return s ? s[0]!.toUpperCase() + s.slice(1) : s;
}

/** A home's number carries across from a list into its own page. */
export function homeTransition(id: string): string {
  return `view-transition-name: home-${id}`;
}

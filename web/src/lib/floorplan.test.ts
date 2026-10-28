import { describe, expect, it } from "vitest";
import type { Plan } from "@/api/client";
import { bounds, drawPlan, plainWidth } from "./floorplan";

const plan: Plan = {
  type: "A",
  w: 6,
  d: 10,
  rooms: [
    { name: "Living", kind: "living", rects: [[0, 0, 6, 5]], area: 30, size: "6.0 × 5.0 m" },
    { name: "Bath", kind: "wet", rects: [[0, 5, 2.5, 7.5]], area: 6.3, size: "2.5 × 2.5 m" },
  ],
  doors: [{ x: 0.5, y: 5, width: 0.9, vertical: false, into: 1, hingeAtEnd: false, entrance: false, service: false }],
  openings: [],
  windows: [[1, 0, 3, 0]],
  bay: [4, -1.2, 6, 0],
  bayCorner: false,
  outdoor: [0, 10, 3, 11.5],
  outdoorName: "Balcony",
  outdoorMore: [],
  items: [{ kind: "bed", r: [3, 6, 5, 8] }, { kind: "sofa", r: [0.2, 1, 2.2, 1.9] }],
};

describe("a floor plan drawn from its geometry", () => {
  it("frames the plan with everything outside its walls", () => {
    expect(bounds(plan)).toEqual([0, -1.2, 6, 11.5]);
    expect(plainWidth(plan)).toBeCloseTo(6 + 0.3 + 1.15);
  });

  it("draws in layers, furniture and dimensions only when not plain", () => {
    const full = drawPlan(plan).layers.map((l) => l.name);
    expect(full).toEqual(["outside", "floors", "walls", "glazing", "doors", "items", "dims", "labels", "north"]);
    expect(drawPlan(plan, { plain: true }).layers.map((l) => l.name)).toEqual(["outside", "floors", "walls", "glazing", "doors", "labels", "north"]);
  });

  it("hatches wet rooms and names every room", () => {
    const d = drawPlan(plan, { id: "t" });
    const floors = d.layers.find((l) => l.name === "floors")!.shapes;
    expect(floors.filter((s) => s.attrs.fill === "url(#t-wet)")).toHaveLength(1);
    const names = d.layers.find((l) => l.name === "labels")!.shapes.filter((s) => s.attrs.class === "fp-name").map((s) => s.text);
    expect(names).toEqual(["Living", "Bath"]);
  });

  it("writes lengths in centimetres", () => {
    const walls = drawPlan(plan).layers.find((l) => l.name === "walls")!.shapes;
    expect(walls.at(-1)!.attrs).toMatchObject({ class: "fp-outer", width: "600", height: "1000" });
  });
});

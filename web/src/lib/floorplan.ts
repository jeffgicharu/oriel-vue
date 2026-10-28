// A home's floor plan, worked out as SVG shapes from the plan's geometry in
// metres: floors, walls, glazing, doors, furniture, dimensions, room names and
// the north arrow, in layers a page can bring in one after another. The
// FloorPlan component renders the shapes; nothing here touches the page.
//
// The drawing is in centimetres, as an architect's would be.

import type { Plan, Rect } from "@/api/client";

export interface Shape {
  tag: "rect" | "path" | "circle" | "ellipse" | "text";
  attrs: Record<string, string>;
  text?: string;
}

export interface Layer {
  name: "outside" | "floors" | "walls" | "glazing" | "doors" | "items" | "dims" | "labels" | "north";
  shapes: Shape[];
}

export interface Drawing {
  viewBox: string;
  id: string;
  layers: Layer[];
}

export interface PlanOptions {
  /** The direction of north on the drawing, in degrees clockwise from up. */
  north?: number;
  /** Prefixes the drawing's internal ids, so several plans can share a page. */
  id?: string;
  /** Leaves out furniture and dimensions, for small drawings. */
  plain?: boolean;
}

const U = 100;

/** A length in metres as centimetres, to one decimal, trimmed. */
function trim(v: number): string {
  let s = (Math.round(v * 10) / 10).toFixed(1);
  if (s.endsWith(".0")) s = s.slice(0, -2);
  return s === "-0" ? "0" : s;
}

function c(v: number): string {
  return trim(v * U);
}

const x0 = (r: Rect) => r[0];
const y0 = (r: Rect) => r[1];
const x1 = (r: Rect) => r[2];
const y1 = (r: Rect) => r[3];
const w = (r: Rect) => r[2] - r[0];
const h = (r: Rect) => r[3] - r[1];
const area = (r: Rect) => w(r) * h(r);

function rect(cls: string, r: Rect, extra: Record<string, string> = {}): Shape {
  return { tag: "rect", attrs: { class: cls, x: c(x0(r)), y: c(y0(r)), width: c(w(r)), height: c(h(r)), ...extra } };
}

function line(cls: string, ax: number, ay: number, bx: number, by: number): Shape {
  return { tag: "path", attrs: { class: cls, d: `M${c(ax)} ${c(ay)}L${c(bx)} ${c(by)}` } };
}

/** The rectangle the plan and everything outside it occupy. */
export function bounds(p: Plan): Rect {
  const b: Rect = [0, 0, p.w, p.d];
  const grow = (r: Rect) => {
    if (w(r) <= 0 || h(r) <= 0) return;
    b[0] = Math.min(b[0], r[0]);
    b[1] = Math.min(b[1], r[1]);
    b[2] = Math.max(b[2], r[2]);
    b[3] = Math.max(b[3], r[3]);
  };
  grow(p.bay);
  grow(p.outdoor);
  p.outdoorMore.forEach(grow);
  return b;
}

/** The width of a plain drawing, margins included, in metres: compared plans share one scale by it. */
export function plainWidth(p: Plan): number {
  return w(bounds(p)) + 0.3 + 1.15;
}

function normal(l: Rect): [number, number] {
  return l[0] === l[2] ? [1, 0] : [0, 1];
}

/** Paints over a wall where an opening, window or door goes. */
function cut(l: Rect, thickness: number): Shape {
  const ax = Math.min(l[0], l[2]);
  const bx = Math.max(l[0], l[2]);
  const ay = Math.min(l[1], l[3]);
  const by = Math.max(l[1], l[3]);
  if (ax === bx) return rect("fp-cut", [ax - thickness / 2, ay, ax + thickness / 2, by]);
  return rect("fp-cut", [ax, ay - thickness / 2, bx, ay + thickness / 2]);
}

function bayGlass(p: Plan): Shape[] {
  const r = p.bay;
  const faces: Rect[] = [];
  if (r[1] < 0) faces.push([r[0], r[1], r[2], r[1]]);
  if (r[3] > p.d) faces.push([r[0], r[3], r[2], r[3]]);
  if (r[0] < 0) faces.push([r[0], r[1], r[0], r[3]]);
  if (r[2] > p.w) faces.push([r[2], r[1], r[2], r[3]]);
  // A bay on a face shows glass on its three outward sides.
  if (!p.bayCorner) faces.push([r[0], r[1], r[0], r[3]], [r[2], r[1], r[2], r[3]]);
  const out: Shape[] = [];
  for (const f of faces) {
    const [dx, dy] = normal(f);
    out.push(line("fp-bayglass", f[0] + dx * 0.04, f[1] + dy * 0.04, f[2] + dx * 0.04, f[3] + dy * 0.04));
    out.push(line("fp-bayglass", f[0] - dx * 0.04, f[1] - dy * 0.04, f[2] - dx * 0.04, f[3] - dy * 0.04));
  }
  return out;
}

function door(d: Plan["doors"][number]): Shape[] {
  const l: Rect = d.vertical ? [d.x, d.y, d.x, d.y + d.width] : [d.x, d.y, d.x + d.width, d.y];
  const out: Shape[] = [cut(l, 0.3)];
  // The hinge, and the leaf swung open at right angles to the wall.
  let [hx, hy, ex, ey] = [l[0], l[1], l[2], l[3]];
  if (d.hingeAtEnd) [hx, hy, ex, ey] = [l[2], l[3], l[0], l[1]];
  const [lx, ly] = d.vertical ? [hx + d.into * d.width, hy] : [hx, hy + d.into * d.width];
  const main = d.entrance || d.service;
  out.push(line(main ? "fp-door fp-door--main" : "fp-door", hx, hy, lx, ly));
  // The arc from the open leaf back to the closed position.
  const sweep = (lx - hx) * (ey - hy) - (ly - hy) * (ex - hx) > 0 ? 1 : 0;
  out.push({ tag: "path", attrs: { class: "fp-swing", d: `M${c(lx)} ${c(ly)}A${c(d.width)} ${c(d.width)} 0 0 ${sweep} ${c(ex)} ${c(ey)}` } });
  if (main) {
    const label = d.service ? "Service door" : "Entrance";
    const mx = (l[0] + l[2]) / 2;
    const my = (l[1] + l[3]) / 2;
    if (d.vertical) {
      out.push({ tag: "text", attrs: { class: "fp-note", x: c(mx + 0.42), y: c(my), "text-anchor": "middle", transform: `rotate(90 ${c(mx + 0.42)} ${c(my)})` }, text: label });
    } else {
      out.push({ tag: "path", attrs: { class: "fp-arrow", d: `M${c(mx)} ${c(my + 0.24)}l-9 -16h18z` } });
      out.push({ tag: "text", attrs: { class: "fp-note", x: c(mx), y: c(my + 0.55), "text-anchor": "middle" }, text: label });
    }
  }
  return out;
}

function roomAt(p: Plan, x: number, y: number): Rect | null {
  for (const room of p.rooms) {
    for (const r of room.rects) {
      if (x >= r[0] && x <= r[2] && y >= r[1] && y <= r[3]) return r;
    }
  }
  return null;
}

function item(p: Plan, it: Plan["items"][number]): Shape[] {
  const r = it.r;
  const room = roomAt(p, (r[0] + r[2]) / 2, (r[1] + r[3]) / 2) ?? r;
  // The side of the piece against a wall: a bed's head, a sofa's back.
  let side = "top";
  let best = Math.abs(r[1] - room[1]);
  for (const [name, dist] of [
    ["bottom", Math.abs(room[3] - r[3])],
    ["left", Math.abs(r[0] - room[0])],
    ["right", Math.abs(room[2] - r[2])],
  ] as const) {
    if (dist < best) [side, best] = [name, dist];
  }
  const out: Shape[] = [];
  switch (it.kind) {
    case "bed": {
      out.push(rect("fp-item", r, { rx: "6" }));
      // Two pillows at the head, and the turned-down sheet.
      const d = 0.4;
      let p1: Rect;
      let p2: Rect;
      if (side === "top" || side === "bottom") {
        const top = side === "bottom" ? r[3] - 0.08 - d : r[1] + 0.08;
        const mid = (r[0] + r[2]) / 2;
        p1 = [r[0] + 0.1, top, mid - 0.05, top + d];
        p2 = [mid + 0.05, top, r[2] - 0.1, top + d];
        const fy = side === "bottom" ? top - 0.25 : top + d + 0.25;
        out.push(line("fp-item", r[0], fy, r[2], fy));
      } else {
        const left = side === "right" ? r[2] - 0.08 - d : r[0] + 0.08;
        const mid = (r[1] + r[3]) / 2;
        p1 = [left, r[1] + 0.1, left + d, mid - 0.05];
        p2 = [left, mid + 0.05, left + d, r[3] - 0.1];
        const fx = side === "right" ? left - 0.25 : left + d + 0.25;
        out.push(line("fp-item", fx, r[1], fx, r[3]));
      }
      out.push(rect("fp-item", p1, { rx: "5" }), rect("fp-item", p2, { rx: "5" }));
      break;
    }
    case "sofa": {
      out.push(rect("fp-item", r, { rx: "8" }));
      const back = 0.22;
      if (side === "top") out.push(line("fp-item", r[0], r[1] + back, r[2], r[1] + back));
      else if (side === "bottom") out.push(line("fp-item", r[0], r[3] - back, r[2], r[3] - back));
      else if (side === "left") out.push(line("fp-item", r[0] + back, r[1], r[0] + back, r[3]));
      else out.push(line("fp-item", r[2] - back, r[1], r[2] - back, r[3]));
      break;
    }
    case "table":
    case "desk":
    case "island":
    case "bedside":
      out.push(rect("fp-item", r, { rx: it.kind === "table" ? "10" : "4" }));
      break;
    case "counter": {
      out.push(rect("fp-item", r));
      // A hob of four rings and a sink on a long run.
      const long = w(r) > h(r);
      if (Math.max(w(r), h(r)) >= 1.8) {
        for (let i = 0; i < 4; i++) {
          let ox = 0.25 + (i % 2) * 0.3;
          let oy = 0.15 + Math.floor(i / 2) * 0.27;
          if (!long) [ox, oy] = [0.15 + (i % 2) * 0.27, 0.25 + Math.floor(i / 2) * 0.3];
          const cx = long ? r[2] - ox : r[0] + ox;
          out.push({ tag: "circle", attrs: { class: "fp-item", cx: c(cx), cy: c(r[1] + oy), r: "8" } });
        }
        const sx = (r[0] + r[2]) / 2;
        const sy = (r[1] + r[3]) / 2;
        out.push({ tag: "rect", attrs: { class: "fp-item", x: c(sx - 0.22), y: c(sy - 0.17), width: c(0.44), height: c(0.34), rx: "6" } });
      }
      break;
    }
    case "wardrobe":
    case "shelves":
      out.push(rect("fp-item", r));
      if (w(r) > h(r)) out.push({ tag: "path", attrs: { class: "fp-item fp-dash", d: `M${c(r[0] + 0.05)} ${c((r[1] + r[3]) / 2)}H${c(r[2] - 0.05)}` } });
      else out.push({ tag: "path", attrs: { class: "fp-item fp-dash", d: `M${c((r[0] + r[2]) / 2)} ${c(r[1] + 0.05)}V${c(r[3] - 0.05)}` } });
      break;
    case "shower":
      out.push(rect("fp-item", r), line("fp-item", r[0], r[1], r[2], r[3]), line("fp-item", r[2], r[1], r[0], r[3]));
      break;
    case "bath":
      out.push(rect("fp-item", r, { rx: "8" }));
      out.push({ tag: "rect", attrs: { class: "fp-item", x: c(r[0] + 0.08), y: c(r[1] + 0.08), width: c(w(r) - 0.16), height: c(h(r) - 0.16), rx: "14" } });
      break;
    case "wc": {
      const cx = (r[0] + r[2]) / 2;
      const cy = (r[1] + r[3]) / 2;
      out.push({ tag: "ellipse", attrs: { class: "fp-item", cx: c(cx), cy: c(cy), rx: c(Math.min(w(r), h(r)) * 0.4), ry: c(Math.max(w(r), h(r)) * 0.32) } });
      out.push(rect("fp-item", r, { rx: "5" }));
      break;
    }
    case "basin": {
      out.push(rect("fp-item", r, { rx: "4" }));
      out.push({ tag: "ellipse", attrs: { class: "fp-item", cx: c((r[0] + r[2]) / 2), cy: c((r[1] + r[3]) / 2), rx: c(w(r) * 0.32), ry: c(h(r) * 0.32) } });
      break;
    }
    case "washer":
      out.push(rect("fp-item", r, { rx: "4" }));
      out.push({ tag: "circle", attrs: { class: "fp-item", cx: c((r[0] + r[2]) / 2), cy: c((r[1] + r[3]) / 2), r: c(Math.min(w(r), h(r)) * 0.32) } });
      break;
    default:
      out.push(rect("fp-item", r));
  }
  return out;
}

function metres(v: number): string {
  return (Math.round(v * 10) / 10).toFixed(1);
}

function dimensions(p: Plan, b: Rect): Shape[] {
  const out: Shape[] = [];
  const y = b[1] - 0.55;
  out.push(line("fp-dim", 0, y, p.w, y), line("fp-dim", 0, y - 0.12, 0, y + 0.12), line("fp-dim", p.w, y - 0.12, p.w, y + 0.12));
  out.push(line("fp-dim fp-ext", 0, y + 0.12, 0, b[1] - 0.08), line("fp-dim fp-ext", p.w, y + 0.12, p.w, b[1] - 0.08));
  out.push({ tag: "text", attrs: { class: "fp-dimtext", x: c(p.w / 2), y: c(y - 0.14), "text-anchor": "middle" }, text: `${metres(p.w)} m` });
  const x = b[2] + 0.5;
  out.push(line("fp-dim", x, 0, x, p.d), line("fp-dim", x - 0.12, 0, x + 0.12, 0), line("fp-dim", x - 0.12, p.d, x + 0.12, p.d));
  out.push({
    tag: "text",
    attrs: { class: "fp-dimtext", x: c(x + 0.16), y: c(p.d / 2), "text-anchor": "middle", transform: `rotate(90 ${c(x + 0.16)} ${c(p.d / 2)})` },
    text: `${metres(p.d)} m`,
  });
  return out;
}

/** Names each room and gives its size, inside its largest rectangle. */
function labels(p: Plan, plain: boolean): Shape[] {
  const out: Shape[] = [];
  const textW = (s: string, fs: number) => [...s].length * fs * 0.0058;
  for (const room of p.rooms) {
    let best = room.rects[0];
    if (!best) continue;
    for (const r of room.rects.slice(1)) if (area(r) > area(best)) best = r;
    const cx = (best[0] + best[2]) / 2;
    let cy = (best[1] + best[3]) / 2;
    let size = 25;
    if (textW(room.name, size) > w(best) - 0.2) size = 19;
    let rotate = false;
    if (textW(room.name, size) > w(best) - 0.15) {
      if (h(best) > w(best) && textW(room.name, size) < h(best) - 0.2) rotate = true;
      else continue;
    }
    const showSize = !plain && room.kind !== "circulation" && h(best) > 1.15 && textW(room.size, 17) < w(best) - 0.15 && !rotate;
    if (showSize) cy -= 0.12;
    const attrs: Record<string, string> = { class: "fp-name", x: c(cx), y: c(cy), "text-anchor": "middle", "dominant-baseline": "middle", "font-size": trim(size) };
    if (rotate) attrs.transform = `rotate(-90 ${c(cx)} ${c(cy)})`;
    out.push({ tag: "text", attrs, text: room.name });
    if (showSize) out.push({ tag: "text", attrs: { class: "fp-size", x: c(cx), y: c(cy + 0.3), "text-anchor": "middle", "dominant-baseline": "middle" }, text: room.size });
  }
  if (w(p.bay) > 0) out.push({ tag: "text", attrs: { class: "fp-note fp-note--bay", x: c((p.bay[0] + p.bay[2]) / 2), y: c(p.bay[1] - 0.12), "text-anchor": "middle" }, text: "Oriel bay" });
  if (w(p.outdoor) > 1.5 && h(p.outdoor) > 0.6) {
    out.push({
      tag: "text",
      attrs: { class: "fp-note", x: c((p.outdoor[0] + p.outdoor[2]) / 2), y: c((p.outdoor[1] + p.outdoor[3]) / 2), "text-anchor": "middle", "dominant-baseline": "middle" },
      text: p.outdoorName,
    });
  }
  return out;
}

/** Works out the whole drawing. */
export function drawPlan(p: Plan, opt: PlanOptions = {}): Drawing {
  const id = opt.id || "fp-" + p.type.toLowerCase();
  const b = bounds(p);
  // The north arrow stands in the right margin of a plain drawing, clear of the plan.
  const [left, top, right, bottom] = opt.plain ? [0.3, 0.3, 1.15, 0.6] : [0.7, 1.35, 1.25, 0.9];
  const viewBox = [c(b[0] - left), c(b[1] - top), c(w(b) + left + right), c(h(b) + top + bottom)].join(" ");

  const outside: Shape[] = [];
  for (const r of [p.outdoor, ...p.outdoorMore]) if (w(r) > 0 && h(r) > 0) outside.push(rect("fp-outdoor", r, { fill: `url(#${id}-out)` }));
  if (w(p.bay) > 0) {
    const inset = 0.14;
    outside.push(rect("fp-bay", p.bay), rect("fp-seat", [p.bay[0] + inset, p.bay[1] + inset, p.bay[2] - inset, p.bay[3] - inset]));
  }

  const floors: Shape[] = [];
  for (const room of p.rooms) {
    for (const r of room.rects) {
      floors.push(rect("fp-floor", r));
      if (room.kind === "wet") floors.push(rect("fp-floor", r, { fill: `url(#${id}-wet)` }));
    }
  }

  // Rooms outlined, then the outer walls heavier.
  const walls: Shape[] = [];
  for (const room of p.rooms) for (const r of room.rects) walls.push(rect("fp-wall", r, { pathLength: "1" }));
  walls.push(rect("fp-outer", [0, 0, p.w, p.d], { pathLength: "1" }));

  const glazing: Shape[] = [];
  for (const o of p.openings) glazing.push(cut(o, 0.3), line("fp-open", o[0], o[1], o[2], o[3]));
  for (const win of p.windows) {
    glazing.push(cut(win, 0.3));
    const [dx, dy] = normal(win);
    for (const off of [-0.08, 0, 0.08]) {
      glazing.push(line(off === 0 ? "fp-glass" : "fp-window", win[0] + dx * off, win[1] + dy * off, win[2] + dx * off, win[3] + dy * off));
    }
  }
  if (w(p.bay) > 0) glazing.push(...bayGlass(p));

  const doors = p.doors.flatMap(door);
  const layers: Layer[] = [
    { name: "outside", shapes: outside },
    { name: "floors", shapes: floors },
    { name: "walls", shapes: walls },
    { name: "glazing", shapes: glazing },
    { name: "doors", shapes: doors },
  ];
  if (!opt.plain) {
    layers.push({ name: "items", shapes: p.items.flatMap((it) => item(p, it)) });
    layers.push({ name: "dims", shapes: dimensions(p, b) });
  }
  layers.push({ name: "labels", shapes: labels(p, !!opt.plain) });

  const north = opt.north ?? 0;
  const [nx, ny] = opt.plain ? [b[2] + 0.6, b[1] + 0.7] : [b[2] + 0.55, b[1] - 0.85];
  layers.push({
    name: "north",
    shapes: [
      { tag: "circle", attrs: { r: "34", transform: `translate(${c(nx)} ${c(ny)}) rotate(${trim(north)})` } },
      { tag: "path", attrs: { d: "M0 -30L11 14L0 6L-11 14Z", transform: `translate(${c(nx)} ${c(ny)}) rotate(${trim(north)})` } },
      { tag: "text", attrs: { y: "-40", "text-anchor": "middle", transform: `translate(${c(nx)} ${c(ny)}) rotate(${trim(north)}) rotate(${trim(-north)} 0 -48)` }, text: "N" },
    ],
  });
  return { viewBox, id, layers };
}

// Photographs, served at three widths in AVIF with WebP for browsers that
// lack it. The files carry fingerprints in their names, found here at build
// time; a picture missing from the build is left out of the page.

const files = import.meta.glob<string>("../assets/photos/*.{avif,webp}", { eager: true, query: "?url", import: "default" });

function url(name: string, width: number, format: "avif" | "webp"): string | undefined {
  return files[`../assets/photos/${name}-${width}.${format}`];
}

/** Each picture's width and height as made. */
const shapes: Record<string, [number, number]> = {
  "living-bay-b": [2400, 1600], "living-bay-a": [2400, 1600], "kitchen-b": [2400, 1600],
  "bedroom-principal": [2400, 1600], "bathroom-ensuite": [1856, 2304], "study-d": [2400, 1600],
  "dining-river": [2400, 1350], "penthouse-terrace": [2400, 1350], "penthouse-living": [2400, 1350],
  "rooftop-pool": [2400, 1350], gym: [2400, 1600], "lounge-workroom": [2400, 1350],
  lobby: [1856, 2304], "balcony-morning": [2400, 1600], materials: [2400, 1600],
  "river-walk": [2400, 1350], "forest-path": [1856, 2304], "neighbourhood-street": [2400, 1350],
  "sales-suite": [2400, 1600], "facade-detail-day": [1856, 2304], "facade-bays-dusk": [1856, 2304],
  "construction-1": [2400, 1600], "construction-2": [2400, 1600], "construction-3": [2400, 1600],
};

export interface PhotoSource {
  avif: string;
  webp: string;
  fallback: string;
  width: number;
  height: number;
}

/** A picture's files, or null when it is not in the build. */
export function photo(name: string): PhotoSource | null {
  const shape = shapes[name];
  if (!shape) return null;
  // Every picture has 800 and 1400 in both formats; landscapes also have a
  // 2000 AVIF for wide, sharp screens.
  const avif: string[] = [];
  const webp: string[] = [];
  for (const w of [800, 1400, 2000]) {
    const a = url(name, w, "avif");
    const b = url(name, w, "webp");
    if (w <= 1400 && (!a || !b)) return null;
    if (a) avif.push(`${a} ${w}w`);
    if (b) webp.push(`${b} ${w}w`);
  }
  return { avif: avif.join(", "), webp: webp.join(", "), fallback: url(name, 1400, "webp")!, width: shape[0], height: shape[1] };
}

export interface Shot {
  name: string;
  alt: string;
  caption: string;
}

export function isPortrait(name: string): boolean {
  const s = shapes[name];
  return !!s && s[1] > s[0];
}

/** The interiors shown for each type of home, in order. */
const typeShots: Record<string, Shot[]> = {
  A: [
    { name: "living-bay-a", alt: "A living room with a three-sided glazed bay, an oak window seat and a cobalt throw, with gardens and jacarandas outside", caption: "The living room and its bay, which looks three ways." },
    { name: "bathroom-ensuite", alt: "A bathroom with a walk-in shower behind a glass panel, terrazzo floors and an oak vanity", caption: "The bathroom: terrazzo, oak and a walk-in shower." },
  ],
  B: [
    { name: "living-bay-b", alt: "A woman reading on an oak window seat in a corner bay, with a cobalt cushion and treetops outside", caption: "The corner bay of the living room, on a February morning." },
    { name: "kitchen-b", alt: "A couple cooking at a pale stone island in an open kitchen with oak cabinets and tall windows", caption: "The kitchen, open to the living room." },
  ],
  C: [
    { name: "dining-river", alt: "A family at an oak dining table in a long living room with tall windows onto trees", caption: "The long living and dining room, on the river side." },
    { name: "bedroom-principal", alt: "A principal bedroom with a low oak bed, linen curtains and a door to the balcony", caption: "The principal bedroom, with a tall door to the outside." },
  ],
  D: [
    { name: "study-d", alt: "A study behind a graphite-framed glass partition, with oak shelves and a desk at the window", caption: "The study, behind a glass partition off the living room." },
    { name: "balcony-morning", alt: "Hands around a coffee cup on a balcony with a glass balustrade, above green treetops", caption: "Morning on the balcony, above the treetops." },
  ],
  P: [
    { name: "penthouse-living", alt: "A penthouse living room with glass on two sides, a corner bay, a low sofa and a cobalt chair", caption: "The penthouse living room, glazed on two sides." },
    { name: "penthouse-terrace", alt: "A couple sitting on a timber terrace with planters and a glass balustrade at sunset", caption: "The terrace that wraps the top floor." },
  ],
};

/** The type's pictures that are in the build. */
export function shotsFor(code: string): Shot[] {
  return (typeShots[code] ?? []).filter((s) => photo(s.name) !== null);
}

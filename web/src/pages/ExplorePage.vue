<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import ArrowIcon from "@/components/ArrowIcon.vue";
import FloorPlan from "@/components/FloorPlan.vue";
import RequestView from "@/components/RequestView.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import type { Home, Status } from "@/api/client";
import { useHomeFilters } from "@/composables/useHomeFilters";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { priceSteps } from "@/lib/filters";
import { kes, kesShort, levelName, monthName, oneDecimal } from "@/lib/format";
import { capitalise } from "@/lib/homes";
import { useFrameStore } from "@/stores/frame";
import { useHomesStore } from "@/stores/homes";
import { useSiteStore } from "@/stores/site";
import { plateSVG, statusName, type PlateHome } from "@/three/explorer/plate";
import type { BuildingModel } from "@/three/model/data";

const route = useRoute("explore");
const router = useRouter();
const site = useSiteStore();
const homes = useHomesStore();
const frame = useFrameStore();
const { filter, update } = useHomeFilters();

usePageMeta({
  title: "Explore the building in 3D",
  description: "Turn Oriel in 3D, open up any floor to see which homes are still available, and move the sun through the day to see the light reach each one.",
  path: "/explore",
  sheet: { number: "A-100", name: "Explore" },
  nav: "explore",
  bodyClass: "is-explorer",
});

// Every home, whatever its status, and the tower's geometry.
const everyHome = { ...filter.value, beds: 0, max: 0, all: true, floors: "" as const, side: "" as const, sort: "floor" as const };
usePageData(() => Promise.all([site.loadModel(), homes.loadList(everyHome)]));

const plateHomes = computed<PlateHome[]>(() =>
  (homes.list.data?.homes ?? []).map((h: Home) => ({
    id: h.id,
    l: h.level,
    p: h.number,
    t: h.type,
    b: h.bedrooms,
    m: h.internal,
    s: h.status,
    ...(h.price !== null ? { pr: h.price } : {}),
    f: h.facings,
    h: h.sun,
    ...(h.riverView ? { r: true } : {}),
  })),
);
const byId = computed(() => new Map(plateHomes.value.map((h) => [h.id, h])));

const matches = (h: PlateHome): boolean => {
  const f = filter.value;
  if (!f.all && h.s !== "available") return false;
  if (f.beds && h.b !== f.beds) return false;
  if (f.max && (!h.pr || h.pr > f.max)) return false;
  return true;
};

// The floors, from the top down: a small stacking plan to choose from.
const levels = computed(() => {
  const rows = new Map<number, PlateHome[]>();
  for (const h of plateHomes.value) rows.set(h.l, [...(rows.get(h.l) ?? []), h]);
  return [...rows.entries()]
    .sort(([a], [b]) => b - a)
    .map(([level, list]) => {
      const cells = list.sort((x, y) => x.p - y.p);
      return { level, cells, shown: cells.filter(matches).length, available: cells.filter((h) => h.s === "available").length };
    });
});

const level = ref<number | null>(null);
const picked = ref<string | null>(null);
const onLevel = computed(() => (level.value === null ? [] : plateHomes.value.filter((h) => h.l === level.value).sort((a, b) => a.p - b.p)));
const card = computed(() => (picked.value ? homes.detail(picked.value) : null));

// What the 3D scene is told when the panel changes; it sets these when it loads.
const hooks: { onPick?: (id: string | null) => void; onLevel?: (l: number | null) => void; onFilter?: () => void } = {};

function chooseLevel(l: number | null) {
  level.value = l;
  hooks.onLevel?.(l);
}

function pick(id: string) {
  const h = byId.value.get(id);
  if (!h) return;
  if (level.value !== h.l) chooseLevel(h.l);
  picked.value = id;
  void homes.loadHome(id);
  void router.replace({ query: { ...route.query, home: id } });
  hooks.onPick?.(id);
}

function closeCard() {
  picked.value = null;
  const query = { ...route.query };
  delete query.home;
  void router.replace({ query });
  hooks.onPick?.(null);
}

function toggleLevel(l: number) {
  if (picked.value) closeCard();
  chooseLevel(level.value === l ? null : l);
}

function reset() {
  if (picked.value) closeCard();
  chooseLevel(null);
}

watch(filter, () => hooks.onFilter?.());

function chooseFilter(event: Event) {
  const el = event.target as HTMLInputElement | HTMLSelectElement;
  if (el.name === "beds") void update({ beds: Number(el.value) || 0 });
  else if (el.name === "max") void update({ max: Number(el.value) || 0 });
  else if (el.name === "show") void update({ all: (el as HTMLInputElement).checked });
}

// ---------- The building: in 3D where it can be drawn, a flat plan otherwise ----------

const root = ref<HTMLElement | null>(null);
const stageEl = ref<HTMLElement | null>(null);
const tip = ref<HTMLElement | null>(null);
const loadingEl = ref<HTMLElement | null>(null);
const sunTime = ref<HTMLInputElement | null>(null);
const sunOut = ref<HTMLOutputElement | null>(null);
const sunMonth = ref<HTMLSelectElement | null>(null);
const flat = ref(false);
const plate = computed(() => {
  const m = site.model.data as unknown as BuildingModel | null;
  if (!flat.value || !m) return "";
  const l = level.value ?? (picked.value ? byId.value.get(picked.value)?.l : undefined) ?? 7;
  return `<p class="label">Level ${String(l).padStart(2, "0")} from above</p>` + plateSVG(m, plateHomes.value, l, matches, picked.value);
});
// The scene is three.js: it is kept out of Vue's reactivity, which would
// otherwise wrap every vertex of it.
const dispose = shallowRef<(() => void) | null>(null);

function showFlat() {
  flat.value = true;
  root.value?.classList.add("is-flat");
  loadingEl.value?.remove();
}

function onPlate(event: Event) {
  const g = (event.target as Element).closest<SVGGElement>("[data-home]");
  if (!g) return;
  if (event instanceof KeyboardEvent) {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
  }
  pick(g.dataset.home!);
}

// The month starts at this month in Nairobi.
const month = computed(() => Number(frame.today.slice(5, 7)) || 1);

onMounted(async () => {
  await Promise.all([site.loadModel(), homes.loadList(everyHome)]);
  await nextTick();
  const start = typeof route.query.home === "string" && byId.value.has(route.query.home) ? route.query.home : undefined;
  const { canDraw3D, isCapture } = await import("@/three/model/capable");
  const model = site.model.data as unknown as BuildingModel | null;
  if (!model || !root.value || !stageEl.value || !canDraw3D()) {
    showFlat();
    if (start) pick(start);
    else chooseLevel(7);
    return;
  }
  try {
    const { mountScene } = await import("@/three/explorer/scene");
    dispose.value = mountScene({
      root: root.value,
      stageEl: stageEl.value,
      tip: tip.value!,
      loading: loadingEl.value,
      sunTime: sunTime.value!,
      sunOut: sunOut.value!,
      sunMonth: sunMonth.value!,
      model,
      homes: byId.value,
      ...(start ? { start } : {}),
      capture: isCapture(),
      picked: () => picked.value,
      matches,
      chooseLevel,
      pick,
      closeCard,
      hooks,
    });
  } catch {
    showFlat();
  }
});

onBeforeUnmount(() => dispose.value?.());

const legend: Status[] = ["available", "held", "reserved", "sold"];
</script>

<template>
  <section ref="root" class="explorer" aria-label="The building in 3D">
    <div ref="stageEl" class="explorer__stage">
      <!-- The plan is drawn from the building's own geometry by our code. -->
      <div v-show="flat" class="explorer__plate" tabindex="-1" @click="onPlate" @keydown="onPlate" v-html="plate" />
      <p ref="loadingEl" class="explorer__loading label">Drawing the building…</p>
      <div ref="tip" class="explorer__tip" hidden />
    </div>
    <div class="explorer__hud">
      <div class="sunbar">
        <label class="sunbar__time">
          <span class="label">Time</span>
          <input ref="sunTime" type="range" min="360" max="1170" step="5" value="970" aria-label="Time of day" />
          <output ref="sunOut" class="mono">16:10</output>
        </label>
        <label class="sunbar__month">
          <span class="label">Month</span>
          <select ref="sunMonth" class="select select--small" aria-label="Month">
            <option v-for="m in 12" :key="m" :value="m" :selected="m === month">{{ monthName(m) }}</option>
          </select>
        </label>
        <button class="btn btn--line btn--small" type="button" @click="reset">Whole building</button>
      </div>
    </div>
    <aside class="explorer__panel" aria-label="Floors and filters">
      <div class="explorer__head">
        <p class="sheet-tag label">A-100 · Explore</p>
        <h1 class="h3">Explore the building</h1>
        <p class="muted">Choose a floor to open it up, then a home to see its plan.</p>
      </div>
      <form class="explorer__filters" @change="chooseFilter" @submit.prevent>
        <fieldset class="filter">
          <legend class="label">Bedrooms</legend>
          <div class="chips">
            <label class="chip"><input type="radio" name="beds" value="" :checked="filter.beds === 0" /><span>Any</span></label>
            <label v-for="n in 4" :key="n" class="chip"><input type="radio" name="beds" :value="n" :checked="filter.beds === n" /><span>{{ n }}</span></label>
          </div>
        </fieldset>
        <div class="explorer__filterrow">
          <label class="filter">
            <span class="label">Up to</span>
            <select class="select select--small" name="max">
              <option value="" :selected="filter.max === 0">Any price</option>
              <option v-for="step in priceSteps" :key="step" :value="step" :selected="filter.max === step">{{ kesShort(step) }}</option>
            </select>
          </label>
          <label class="filter filter--toggle">
            <input type="checkbox" name="show" value="all" :checked="filter.all" />
            <span>Show sold</span>
          </label>
        </div>
      </form>
      <RequestView :state="homes.list" :retry="() => homes.loadList(everyHome, true)" loading-label="Loading the floors…">
        <template #default>
          <ol class="levels" role="list">
            <li v-for="row in levels" :key="row.level">
              <button class="level" :class="{ 'is-empty': row.shown === 0 }" type="button" :aria-pressed="level === row.level" @click="toggleLevel(row.level)">
                <span class="level__name mono">{{ levelName(row.level) }}</span>
                <span class="level__plan" aria-hidden="true">
                  <i v-for="h in row.cells" :key="h.id" :class="['cell', `cell--${h.s}`, { 'is-dim': !matches(h) }]" />
                </span>
                <span class="level__count">{{ row.shown > 0 ? `${row.shown} ${filter.all ? "shown" : "free"}` : "—" }}</span>
                <span class="visually-hidden">, {{ row.available }} of {{ row.cells.length }} homes available</span>
              </button>
            </li>
          </ol>
        </template>
      </RequestView>
      <div v-if="level !== null" class="explorer__homes">
        <p class="label">Level {{ String(level).padStart(2, "0") }} · {{ onLevel.length }} homes</p>
        <ul role="list">
          <li v-for="h in onLevel" :key="h.id">
            <button type="button" :class="['lhome', { 'is-dim': !matches(h), 'is-picked': picked === h.id }]" @click="pick(h.id)">
              <span class="mono lhome__id">{{ h.id }}</span>
              <span>{{ h.b === 4 ? "Penthouse" : `${h.b} bed` }} · {{ h.f }}</span>
              <span class="lhome__right">
                <span v-if="h.pr" class="tnum">KES {{ (h.pr / 1e6).toFixed(2) }}M</span>
                <span :class="`status status--${h.s}`"><span class="status__swatch" aria-hidden="true" />{{ statusName(h.s) }}</span>
              </span>
            </button>
          </li>
        </ul>
      </div>
      <ul class="explorer__legend" role="list">
        <li v-for="s in legend" :key="s"><StatusBadge :status="s" /></li>
      </ul>
    </aside>
    <div v-if="card" class="explorer__card" aria-live="polite">
      <RequestView :state="card" :retry="() => picked && homes.loadHome(picked, true)" loading-label="Loading the home…">
        <template #default="{ data: d }">
          <article class="hcard">
            <button class="hcard__close" type="button" aria-label="Close" @click="closeCard">
              <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M3 3l10 10M13 3L3 13" /></svg>
            </button>
            <header class="hcard__head">
              <p class="hcard__id">{{ d.home.id }}</p>
              <div>
                <p class="label label--cobalt">Type {{ d.home.type }} · {{ levelName(d.home.level) }}</p>
                <h2 class="hcard__title">{{ d.home.title }}</h2>
              </div>
            </header>
            <div class="hcard__plan">
              <FloorPlan :plan="d.plan" plain :north="d.north" :id="`hc-${d.home.id}`" :title="`Plan of home ${d.home.id}`" />
            </div>
            <dl class="hcard__facts">
              <div><dt class="label">Inside</dt><dd>{{ d.home.internal }} m²</dd></div>
              <div><dt class="label">Faces</dt><dd>{{ capitalise(d.home.facings) }}</dd></div>
              <div><dt class="label">Sun a day</dt><dd>{{ oneDecimal(d.home.sun) }} h</dd></div>
            </dl>
            <div class="hcard__deal">
              <StatusBadge :status="d.home.status" />
              <p v-if="d.home.price !== null" class="hcard__price tnum">{{ kes(d.home.price) }}</p>
            </div>
            <div class="hcard__actions">
              <RouterLink class="btn btn--cobalt btn--small" :to="{ name: 'home-detail', params: { id: d.home.id } }">See this home <ArrowIcon /></RouterLink>
              <RouterLink class="btn btn--line btn--small" :to="{ name: 'viewing-new', query: { home: d.home.id } }">Book a viewing</RouterLink>
            </div>
          </article>
        </template>
      </RequestView>
    </div>
  </section>
</template>

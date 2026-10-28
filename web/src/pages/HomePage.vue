<script setup lang="ts">
import { computed } from "vue";
import elevation from "@/generated/elevation.json";
import places from "@/generated/places.json";
import ArrowIcon from "@/components/ArrowIcon.vue";
import PhotoPicture from "@/components/PhotoPicture.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import type { Status } from "@/api/client";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { kesShort, levelName, monthYear } from "@/lib/format";
import { statusName } from "@/lib/homes";
import { complexLd, organizationLd, websiteLd } from "@/lib/seo";
import { useFrameStore } from "@/stores/frame";
import { useSiteStore } from "@/stores/site";

const site = useSiteStore();
const frame = useFrameStore();
usePageData(() => Promise.all([site.loadOverview(), site.loadModel()]));

const o = computed(() => site.overview.data);
const completion = computed(() => (o.value ? monthYear(o.value.completion) : ""));

usePageMeta(() => ({
  description: `Eighty-six homes in Riverside, Nairobi, each with a glazed bay that catches the morning light. Find a home, see its plan and its light, and book a viewing. Completing ${completion.value}.`,
  path: "/",
  sheet: { number: "A-000", name: "General arrangement" },
  headerOver: true,
  jsonLd: o.value ? [organizationLd(frame.siteUrl), websiteLd(frame.siteUrl), complexLd(frame.siteUrl, o.value.completion)] : [],
}));

const numberWords = ["No", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"];
const tens = ["", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety"];
function inWords(n: number): string {
  if (n < 20) return numberWords[n]!;
  const t = tens[Math.floor(n / 10)]!;
  return n % 10 ? `${t}-${numberWords[n % 10]!.toLowerCase()}` : t;
}

// The stacking plan: every floor of homes, as the sales team sees it.
const stack = computed(() => {
  const m = site.model.data;
  if (!m) return [];
  const rows = new Map<number, { id: string; status: Status; type: string; wide: boolean }[]>();
  for (const u of m.units) {
    const cells = rows.get(u.l) ?? [];
    cells.push({ id: u.id, status: m.status[u.id] ?? "available", type: u.t, wide: u.l === 16 });
    rows.set(u.l, cells);
  }
  return [...rows.entries()]
    .sort(([a], [b]) => b - a)
    .map(([level, cells]) => ({ level, cells: cells.sort((x, y) => x.id.localeCompare(y.id)) }));
});

const progressTitle = computed(() => {
  const p = o.value?.progress;
  if (!p) return "";
  if (p.percent >= 100) return "Complete.";
  if (p.frame > 0 && p.frame < 18) return "Rising on Riverside.";
  if (p.frame >= 18) return "Topped out.";
  return "Breaking ground.";
});

const nearby = ["Westlands", "City centre", "Karura Forest", "JKIA"].map((name) => places.find((p) => p.name === name)!).filter(Boolean);
const statuses: Status[] = ["available", "held", "reserved", "sold", "withheld"];
const plans = [
  { name: "Pay as it rises", summary: "10% to reserve, 20% when you sign, 40% in quarterly instalments while the building goes up, and 30% when you get the keys." },
  { name: "Cash", summary: "10% to reserve and the rest within 90 days of signing, with 7% off the price." },
  { name: "Mortgage", summary: "20% from you before completion, and 80% from your bank when you get the keys." },
];
</script>

<template>
  <section class="hero hero--flat drawing-grid" aria-labelledby="hero-title">
    <div class="hero__stage">
      <!-- The south elevation traces itself floor by floor when the page first paints. -->
      <svg class="hero__drawing" viewBox="0 -12 600 668" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linejoin="round" aria-hidden="true" focusable="false">
        <path v-for="(s, i) in elevation" :key="i" :d="s.d" :class="s.class" pathLength="1" :style="`--i: ${s.order}`" data-draw />
      </svg>
    </div>
    <div class="wrap hero__content">
      <p class="label label--cobalt hero__place"><span>Riverside, Nairobi</span><span v-if="completion">Completing {{ completion }}</span></p>
      <h1 id="hero-title" class="display hero__title">Homes designed around the light.</h1>
      <p class="lede">Eighty-six apartments over fifteen floors. Every one has a glazed bay that reaches out for the morning sun and the long view over the river.</p>
      <div class="hero__actions">
        <RouterLink class="btn btn--cobalt" :to="{ name: 'residences' }">Find your home <ArrowIcon /></RouterLink>
        <RouterLink class="btn btn--line" :to="{ name: 'viewing-new' }">Book a viewing</RouterLink>
      </div>
    </div>
    <div class="hero__strip">
      <dl class="wrap hero__facts">
        <div><dt class="label">Homes</dt><dd>86</dd></div>
        <div><dt class="label">Bedrooms</dt><dd>1 to 4</dd></div>
        <div>
          <dt class="label">From</dt>
          <dd class="tnum">{{ o ? (o.from > 0 ? kesShort(o.from) : "Sold out") : "—" }}</dd>
        </div>
        <div><dt class="label">Completion</dt><dd>{{ completion || "—" }}</dd></div>
      </dl>
    </div>
  </section>

  <section class="section">
    <div class="wrap intro">
      <p class="sheet-tag label">A-001 · The idea</p>
      <div class="intro__body">
        <p>An oriel is a window that projects from a wall to catch more light and more view. <span class="accent">Every home at Oriel has one.</span></p>
        <p>Close to the equator the sun climbs high and crosses the sky fast, so the building is shaded by deep fins and balconies and opened up where the low morning and evening light comes in.</p>
        <RouterLink class="link-arrow" :to="{ name: 'building' }">How the building works <ArrowIcon /></RouterLink>
      </div>
    </div>
  </section>

  <section class="story story--bay" aria-labelledby="story-bay">
    <div class="wrap story-bay">
      <figure class="story-bay__photo">
        <PhotoPicture name="living-bay-b" alt="A woman reading on an oak window seat in a corner bay, with a cobalt cushion and treetops outside" sizes="(min-width: 1520px) 1408px, 92vw" />
      </figure>
      <div class="story-bay__card">
        <p class="sheet-tag label">A-002 · The bay</p>
        <h2 id="story-bay" class="h2">Sit in the light.</h2>
        <p>Each home has a window seat in a glazed bay at its corner, stepping out from the wall to catch the morning sun and the long view.</p>
        <RouterLink class="link-arrow" :to="{ name: 'building' }">How the bay works <ArrowIcon /></RouterLink>
      </div>
    </div>
  </section>

  <section class="story" aria-labelledby="story-homes">
    <div class="wrap story-homes">
      <div class="story-homes__list">
        <p class="sheet-tag label">A-003 · The homes</p>
        <h2 id="story-homes" class="h2">Five plans, eighty‑six homes.</h2>
        <ol v-if="o" class="story-types" role="list">
          <li v-for="t in o.types" :key="t.code">
            <RouterLink class="story-types__row" :to="{ name: 'home-type', params: { code: t.code } }">
              <span class="story-types__code mono">{{ t.code }}</span>
              <span class="story-types__name">{{ t.name }}</span>
              <span class="story-types__from tnum">
                <template v-if="t.available > 0 && t.from !== null">from {{ kesShort(t.from) }}</template>
                <template v-else-if="t.withheld > 0">Next release</template>
                <template v-else>Sold out</template>
              </span>
            </RouterLink>
          </li>
        </ol>
        <RouterLink class="btn btn--cobalt" :to="{ name: 'residences' }">See every home <ArrowIcon /></RouterLink>
      </div>
      <div class="story-homes__photos">
        <PhotoPicture name="kitchen-b" alt="A couple cooking at a pale stone island in an open kitchen with oak cabinets and tall windows" sizes="(min-width: 1000px) 34vw, 100vw" class="story-homes__a" />
        <PhotoPicture name="penthouse-terrace" alt="A couple sitting on a timber terrace with planters and a glass balustrade at sunset" sizes="(min-width: 1000px) 24vw, 50vw" class="story-homes__b" />
        <PhotoPicture name="study-d" alt="A study behind a graphite-framed glass partition, with oak shelves and a desk at the window" sizes="(min-width: 1000px) 24vw, 50vw" class="story-homes__c" />
      </div>
    </div>
  </section>

  <section class="story story--rule" aria-labelledby="story-stack">
    <div class="wrap story-stack">
      <div class="story-stack__text">
        <p class="sheet-tag label">A-004 · Stacking plan</p>
        <h2 id="story-stack" class="h2">{{ o ? inWords(o.available) : "" }} homes free today.</h2>
        <p class="lede">Every floor of the building, as the sales team sees it. Open any home to see its plan, the light it gets and what it costs to pay for.</p>
        <ul class="story-stack__key" role="list">
          <li v-for="st in statuses" :key="st"><StatusBadge :status="st" /></li>
        </ul>
        <RouterLink class="btn btn--line" :to="{ name: 'residences', query: { show: 'all' } }">See every home <ArrowIcon /></RouterLink>
      </div>
      <div v-if="o" class="stack" role="img" :aria-label="`Stacking plan: ${o.available} of ${o.total} homes are available.`">
        <div v-for="row in stack" :key="row.level" class="stack__row">
          <span class="stack__level mono" aria-hidden="true">{{ levelName(row.level) }}</span>
          <div class="stack__cells">
            <RouterLink
              v-for="c in row.cells"
              :key="c.id"
              :class="['stack__cell', `is-${c.status}`, { 'stack__cell--wide': c.wide }]"
              :to="{ name: 'home-detail', params: { id: c.id } }"
              :title="`Home ${c.id}, type ${c.type}: ${statusName(c.status)}`"
              tabindex="-1"
              aria-hidden="true"
              ><span>{{ c.id }}</span></RouterLink
            >
          </div>
        </div>
      </div>
    </div>
  </section>

  <section class="story story--rule" aria-labelledby="story-shared">
    <div class="wrap">
      <div class="story-wide">
        <figure class="story-wide__photo crop">
          <PhotoPicture name="rooftop-pool" alt="A long pool on the roof with a timber deck, loungers under a slatted pergola and treetops beyond" sizes="(min-width: 1520px) 1408px, 92vw" />
        </figure>
        <div class="story-wide__text">
          <p class="sheet-tag label">A-005 · Shared</p>
          <h2 id="story-shared" class="h2">A pool above the treetops.</h2>
          <p class="muted">A rooftop pool and terrace, a gym and a workroom on the garden, a concierge day and night, and a path along the river.</p>
          <RouterLink class="link-arrow" :to="{ name: 'building' }">Everything residents share <ArrowIcon /></RouterLink>
        </div>
      </div>
    </div>
  </section>

  <section class="story story--rule" aria-labelledby="story-place">
    <div class="wrap story-place">
      <div>
        <p class="sheet-tag label">A-006 · Location</p>
        <h2 id="story-place" class="h2">Five minutes to Westlands. Fifteen to the forest.</h2>
        <dl class="story-times">
          <div v-for="p in nearby" :key="p.name">
            <dt>{{ p.name }}</dt>
            <dd><span class="tnum">{{ p.minutes }}</span> min</dd>
          </div>
        </dl>
        <RouterLink class="link-arrow" :to="{ name: 'location' }">The neighbourhood <ArrowIcon /></RouterLink>
      </div>
      <PhotoPicture name="neighbourhood-street" alt="A quiet street in Riverside under jacaranda trees, with people walking and a café in a garden" sizes="(min-width: 1000px) 55vw, 100vw" class="story-place__photo" />
    </div>
  </section>

  <section class="story story--rule" aria-labelledby="story-buy">
    <div class="wrap story-buy">
      <div class="story-buy__head">
        <p class="sheet-tag label">A-007 · Buying</p>
        <h2 id="story-buy" class="h2">Pay your way.</h2>
        <p class="lede">Three ways to pay, from staged payments while the building rises to a mortgage at completion. Every home’s page works out its own schedule to the shilling.</p>
      </div>
      <ol class="story-buy__plans" role="list">
        <li v-for="p in plans" :key="p.name">
          <h3 class="h3">{{ p.name }}</h3>
          <p class="muted">{{ p.summary }}</p>
        </li>
      </ol>
      <RouterLink class="link-arrow" :to="{ name: 'buying' }">How buying off-plan works <ArrowIcon /></RouterLink>
    </div>
  </section>

  <section v-if="o" class="story story--rule" aria-labelledby="story-progress">
    <div class="wrap story-progress">
      <figure class="story-progress__photo crop">
        <PhotoPicture name="construction-3" alt="The concrete frame of the building rising, with workers on the top deck and a crane hook overhead" sizes="(min-width: 1000px) 50vw, 100vw" />
      </figure>
      <div class="story-progress__text">
        <p class="sheet-tag label">A-008 · Progress</p>
        <h2 id="story-progress" class="h2">{{ progressTitle }}</h2>
        <p class="story-progress__figure"><span class="tnum">{{ o.progress.percent }}%</span> complete</p>
        <div class="progress-bar" role="presentation"><span :style="`width: ${o.progress.percent}%`" /></div>
        <p class="muted">{{ o.progress.current }}. Completion in {{ completion }}.</p>
        <RouterLink class="link-arrow" :to="{ name: 'progress' }">Follow the build <ArrowIcon /></RouterLink>
      </div>
    </div>
  </section>
</template>

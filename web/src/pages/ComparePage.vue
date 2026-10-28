<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import FloorPlan from "@/components/FloorPlan.vue";
import RequestView from "@/components/RequestView.vue";
import type { HomeDetail } from "@/api/client";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { plainWidth } from "@/lib/floorplan";
import { kes, oneDecimal } from "@/lib/format";
import { capitalise, statusName } from "@/lib/homes";
import { useHomesStore } from "@/stores/homes";

const route = useRoute("compare");
const router = useRouter();
const homes = useHomesStore();

const ids = computed(() =>
  String(route.query.homes ?? "")
    .split(",")
    .map((s) => s.trim())
    .filter((s) => /^\d{4}$/.test(s))
    .filter((s, i, all) => all.indexOf(s) === i)
    .slice(0, 3),
);

usePageData(() => homes.loadCompared(ids.value), () => ids.value.join(","));

usePageMeta(() => ({
  title: ids.value.length ? `Compare homes ${ids.value.join(", ")}` : "Compare homes",
  description: "Homes at Oriel, Riverside side by side: plans drawn to the same scale, sizes, aspect, sun and prices.",
  path: "/compare",
  sheet: { number: "A-220", name: "Compare" },
}));

/** Every plan on the page is drawn to the scale of the widest. */
function planWidth(d: HomeDetail, all: HomeDetail[]): string {
  const widest = Math.max(...all.map((x) => plainWidth(x.plan)));
  return `width: ${((100 * plainWidth(d.plan)) / widest).toFixed(1)}%`;
}

function without(id: string) {
  const keep = ids.value.filter((x) => x !== id);
  if (keep.length === 0) return router.push({ name: "shortlist" });
  return router.replace({ query: { homes: keep.join(",") } });
}

const rows: { label: string; value: (d: HomeDetail) => string }[] = [
  { label: "Home", value: (d) => d.home.title },
  { label: "Type", value: (d) => `${d.home.type} · ${d.home.typeName}` },
  { label: "Faces", value: (d) => capitalise(d.home.facings) },
  { label: "River view", value: (d) => (d.home.riverView ? "Yes" : "No") },
  { label: "Inside", value: (d) => `${d.home.internal} m²` },
  { label: "Outside", value: (d) => `${d.home.outdoorName} ${d.home.outdoor} m²` },
  { label: "Sun a day", value: (d) => `${oneDecimal(d.home.sun)} hours on average` },
  { label: "Status", value: (d) => statusName(d.home.status) },
  { label: "Price", value: (d) => (d.home.price !== null ? kes(d.home.price) : "—") },
];
</script>

<template>
  <div class="wrap content-page">
    <header class="page-head content-head">
      <p class="sheet-tag label">A-220 · Compare</p>
      <h1 class="h1">Side by side.</h1>
      <p class="lede">
        <template v-if="(homes.compared.data?.length ?? 0) < 2">Add another home from your shortlist to compare them.</template>
        <template v-else>{{ homes.compared.data!.length }} homes, drawn to the same scale.</template>
      </p>
    </header>
    <RequestView :state="homes.compared" :retry="() => homes.loadCompared(ids, true)" loading-label="Loading the homes…">
      <template #default="{ data }">
        <div class="compare" :style="`--cols: ${Math.max(1, data.length)}`">
          <div class="compare__row compare__row--plans">
            <span class="compare__label label">Plan</span>
            <div v-for="d in data" :key="d.home.id" class="compare__cell">
              <RouterLink class="compare__id mono" :to="{ name: 'home-detail', params: { id: d.home.id } }">{{ d.home.id }}</RouterLink>
              <div class="compare__plan">
                <div class="compare__scale" :style="planWidth(d, data)">
                  <FloorPlan :plan="d.plan" plain :north="d.north" :id="`cmp-${d.home.id}`" :title="`Plan of home ${d.home.id}`" />
                </div>
              </div>
            </div>
          </div>
          <div v-for="row in rows" :key="row.label" class="compare__row">
            <span class="compare__label label">{{ row.label }}</span>
            <span v-for="d in data" :key="d.home.id" class="compare__cell">{{ row.value(d) }}</span>
          </div>
          <div class="compare__row compare__row--actions">
            <span class="compare__label" />
            <div v-for="d in data" :key="d.home.id" class="compare__cell">
              <RouterLink class="btn btn--cobalt btn--small" :to="{ name: 'home-detail', params: { id: d.home.id } }">See this home</RouterLink>
              <button class="link-arrow as-button compare__remove" type="button" @click="without(d.home.id)">Take it out</button>
            </div>
          </div>
        </div>
      </template>
    </RequestView>
    <p class="content-note muted"><RouterLink :to="{ name: 'shortlist' }">Back to your shortlist</RouterLink></p>
  </div>
</template>

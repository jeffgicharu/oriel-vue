<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import RequestView from "@/components/RequestView.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import DeskLayout from "@/components/desk/DeskLayout.vue";
import HomeDrawer from "@/components/desk/HomeDrawer.vue";
import type { DeskHome } from "@/api/desk";
import type { LiveEvent } from "@/composables/useLiveEvents";
import { usePageData } from "@/composables/usePageData";
import { kesShort } from "@/lib/format";
import { useDeskStore } from "@/stores/desk";
import { useDeskHomesStore } from "@/stores/deskHomes";

const route = useRoute("desk-stack");
const router = useRouter();
const desk = useDeskStore();
const homes = useDeskHomesStore();

// The home whose drawer is open is in the address, so it can be shared and
// the back button closes it.
const openId = computed(() => {
  const h = String(route.query.home ?? "");
  return /^\d{4}$/.test(h) ? h : "";
});
const openState = computed(() => (openId.value ? homes.home(openId.value) : null));

usePageData(
  async () => (route.name === "desk-stack" ? Promise.all([homes.loadStack(), openId.value ? homes.loadHome(openId.value) : undefined]) : undefined),
  () => openId.value,
);

const TYPES = ["A", "B", "C", "D", "P"];
const LEGEND = ["available", "held", "reserved", "sold", "withheld"] as const;

// Homes that have just changed fill in for a moment.
const changed = ref<string[]>([]);
function flash(id: string) {
  changed.value = [...changed.value.filter((x) => x !== id), id];
  window.setTimeout(() => (changed.value = changed.value.filter((x) => x !== id)), 2400);
}

function reload(event: LiveEvent) {
  if (!["unit", "prices", "reservation"].includes(event.type)) return;
  void homes.loadStack(true);
  if (event.type === "unit" && event.id) flash(event.id);
  if (openId.value) void homes.loadHome(openId.value, true);
}

function cellPrice(h: DeskHome): string {
  return h.status === "sold" || h.status === "reserved" ? h.statusName : kesShort(h.price);
}

// Opening a home from the plan moves focus into its drawer once it has loaded.
let focusDrawer = false;
function open(id: string) {
  focusDrawer = true;
  void router.push({ query: { home: id } });
}

function close() {
  if (!openId.value) return;
  const id = openId.value;
  void router.push({ query: {} }).then(() => document.querySelector<HTMLElement>(`[data-unit="${id}"]`)?.focus());
}

watch(
  () => openState.value?.status,
  async (status) => {
    if (!focusDrawer || status !== "ready") return;
    focusDrawer = false;
    await nextTick();
    document.querySelector<HTMLElement>("#drawer .desk-drawer__close")?.focus({ preventScroll: true });
  },
);

function onKey(e: KeyboardEvent) {
  if (e.key === "Escape" && openId.value && !(e.target instanceof HTMLInputElement)) close();
}
onMounted(() => document.addEventListener("keydown", onKey));
onBeforeUnmount(() => document.removeEventListener("keydown", onKey));
</script>

<template>
  <DeskLayout title="Stacking plan" nav="stack" :reload="reload">
    <DeskHeader title="Stacking plan" lede="Every home, floor by floor. Changes made anywhere, by the team or on the website, appear here as they happen.">
      <RouterLink class="btn btn--line btn--small" :to="{ name: 'desk-pricing' }">Prices</RouterLink>
    </DeskHeader>
    <RequestView :state="homes.stack" :retry="() => homes.loadStack(true)" loading-label="Loading the stacking plan…">
      <template #default="{ data }">
        <div class="desk-stack">
          <div class="desk-stack__bar">
            <fieldset class="chips">
              <legend class="visually-hidden">Show a type</legend>
              <label class="chip"><input type="radio" name="stack-type" value="" checked /><span>All types</span></label>
              <label v-for="code in TYPES" :key="code" class="chip"><input type="radio" name="stack-type" :value="code" /><span>{{ code }}</span></label>
            </fieldset>
            <ul class="desk-legend" role="list">
              <li v-for="st in LEGEND" :key="st"><StatusBadge :status="st" /> <span class="tnum">{{ data.counts[st] ?? 0 }}</span></li>
            </ul>
          </div>
          <div class="stack-grid" role="group" aria-label="Stacking plan">
            <div v-for="floor in data.floors" :key="floor.level" class="stack-grid__row">
              <span class="stack-grid__level mono">{{ floor.name }}</span>
              <button
                v-for="h in floor.homes"
                :key="h.id"
                class="stack-cell"
                :class="[`is-${h.status}`, { 'stack-cell--wide': h.wide, 'is-changed': changed.includes(h.id) }]"
                type="button"
                :data-unit="h.id"
                :data-type="h.type"
                :aria-current="openId === h.id ? 'true' : undefined"
                @click="open(h.id)"
              >
                <!-- The name is what the cell shows, whole, with the rest said before and after it. -->
                <span class="stack-cell__id mono"><span class="visually-hidden">Home </span>{{ h.id }}</span>
                <span class="stack-cell__type"><span class="visually-hidden">, </span>{{ h.type }}</span>
                <span class="stack-cell__price tnum"><span class="visually-hidden">, </span>{{ cellPrice(h) }}</span>
                <span class="visually-hidden">, {{ h.typeName }}{{ cellPrice(h) === h.statusName ? "" : `, ${h.statusName}` }}</span>
              </button>
            </div>
          </div>
        </div>
      </template>
    </RequestView>
    <aside id="drawer" class="desk-drawer" aria-label="Home details">
      <RequestView v-if="openState" :state="openState" :retry="() => homes.loadHome(openId, true)" loading-label="Loading the home…">
        <template #default="{ data }">
          <HomeDrawer :key="data.home.id" :detail="data" :now="desk.now" @close="close" />
        </template>
        <template #error="{ error }">
          <button class="desk-drawer__close" type="button" aria-label="Close" @click="close">×</button>
          <p class="request-error__title">{{ error.status === 404 ? `There is no home ${openId}.` : error.title }}</p>
          <button v-if="error.status !== 404" class="btn btn--line btn--small" type="button" @click="homes.loadHome(openId, true)">Try again</button>
        </template>
      </RequestView>
    </aside>
  </DeskLayout>
</template>

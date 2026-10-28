<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import ArrowIcon from "@/components/ArrowIcon.vue";
import RequestView from "@/components/RequestView.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import { usePageMeta } from "@/composables/usePageMeta";
import { kesShort } from "@/lib/format";
import { useHomesStore } from "@/stores/homes";
import { useShortlistStore } from "@/stores/shortlist";

const MAX_COMPARE = 3;
const homes = useHomesStore();
const shortlist = useShortlistStore();
const router = useRouter();

usePageMeta({
  title: "Your shortlist",
  description: "The homes you have shortlisted at Oriel, Riverside, kept in this browser. Choose up to three to compare side by side.",
  path: "/shortlist",
  sheet: { number: "A-210", name: "Shortlist" },
});

// The shortlist lives in this browser, so it is read once the page is
// running here; the server sends the page without it.
// The first three are chosen for comparing to begin with; after that the
// visitor's choices stand, less any home they take off the list.
const picked = ref<string[]>([]);
let chosenOnce = false;
watch(
  () => [shortlist.ready, shortlist.ids.join(",")] as const,
  ([ready]) => {
    if (!ready) return;
    picked.value = chosenOnce ? picked.value.filter((id) => shortlist.has(id)) : shortlist.ids.slice(0, MAX_COMPARE);
    chosenOnce = true;
    if (shortlist.ids.length) void homes.loadShortlisted(shortlist.ids);
  },
  { immediate: true },
);

const note = computed(() => (picked.value.length < 2 ? "Choose two or three." : `${picked.value.length} chosen.`));

function togglePick(id: string, on: boolean) {
  picked.value = on ? [...picked.value, id].slice(0, MAX_COMPARE) : picked.value.filter((x) => x !== id);
}

function compare() {
  if (picked.value.length) void router.push({ name: "compare", query: { homes: picked.value.join(",") } });
}
</script>

<template>
  <div class="wrap content-page">
    <header class="page-head content-head">
      <p class="sheet-tag label">A-210 · Shortlist</p>
      <h1 class="h1">Your shortlist.</h1>
      <p class="lede">The homes you have marked with a heart, kept in this browser. Choose up to three to compare side by side.</p>
    </header>
    <p v-if="!shortlist.ready" class="muted" aria-busy="true">Reading your shortlist…</p>
    <div v-else-if="shortlist.count === 0" class="shortlist-empty">
      <p>Nothing on your shortlist yet. Tap the heart on any home to add it.</p>
      <RouterLink class="btn btn--cobalt" :to="{ name: 'residences' }">See the homes <ArrowIcon /></RouterLink>
    </div>
    <RequestView v-else :state="homes.shortlisted" :retry="() => homes.loadShortlisted(shortlist.ids, true)" loading-label="Loading your homes…">
      <template #default="{ data }">
        <form class="shortlist" @submit.prevent="compare">
          <ul class="shortlist__list" role="list">
            <li v-for="h in data.homes.filter((x) => shortlist.has(x.id))" :key="h.id" class="shortlist__item">
              <label class="shortlist__pick">
                <input
                  type="checkbox"
                  :value="h.id"
                  :checked="picked.includes(h.id)"
                  :disabled="!picked.includes(h.id) && picked.length >= MAX_COMPARE"
                  @change="togglePick(h.id, ($event.target as HTMLInputElement).checked)"
                />
                <span class="visually-hidden">Compare home {{ h.id }}</span>
              </label>
              <RouterLink class="shortlist__home" :to="{ name: 'home-detail', params: { id: h.id } }">
                <span class="shortlist__id mono">{{ h.id }}</span>
                <span class="shortlist__title">{{ h.title }}</span>
                <span class="shortlist__meta">Faces {{ h.facings }} · {{ h.internal }} m²</span>
              </RouterLink>
              <span class="shortlist__deal">
                <StatusBadge :status="h.status" />
                <span v-if="h.price !== null" class="shortlist__price tnum">{{ kesShort(h.price) }}</span>
              </span>
              <button class="link-button shortlist__remove" type="button" @click="shortlist.remove(h.id)">Remove</button>
            </li>
          </ul>
          <div class="shortlist__actions">
            <button class="btn btn--cobalt" type="submit" :disabled="picked.length === 0">Compare the chosen homes <ArrowIcon /></button>
            <span class="form-small muted" aria-live="polite">{{ note }}</span>
          </div>
        </form>
      </template>
    </RequestView>
  </div>
</template>

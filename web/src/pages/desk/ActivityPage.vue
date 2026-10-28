<script setup lang="ts">
import { computed } from "vue";
import { useRoute, type LocationQueryRaw } from "vue-router";
import RequestView from "@/components/RequestView.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import DeskLayout from "@/components/desk/DeskLayout.vue";
import { usePageData } from "@/composables/usePageData";
import { activityWords, dotColour, when } from "@/lib/desk";
import { useDeskStore, type ActivityKind } from "@/stores/desk";

const route = useRoute("desk-activity");
const desk = useDeskStore();

// Sixty lines a page, the newest first.
const PAGE = 60;

const TABS: { key: ActivityKind; label: string }[] = [
  { key: "", label: "Everything" },
  { key: "staff", label: "The team" },
  { key: "visitor", label: "The website" },
  { key: "system", label: "Automatic" },
];

const actor = computed<ActivityKind>(() => {
  const a = String(route.query.actor ?? "");
  return a === "staff" || a === "visitor" || a === "system" ? a : "";
});
const page = computed(() => {
  const n = Number(route.query.page ?? 1);
  return Number.isInteger(n) && n > 1 ? Math.min(n, 1000) : 1;
});

usePageData(
  async () => (route.name === "desk-activity" ? desk.loadActivity(actor.value, (page.value - 1) * PAGE) : undefined),
  () => `${actor.value}:${page.value}`,
);

function pageQuery(a: ActivityKind, p: number): LocationQueryRaw {
  return { ...(a ? { actor: a } : {}), ...(p > 1 ? { page: String(p) } : {}) };
}

// New lines arrive at the top, so only the first page follows them.
function reload() {
  if (page.value === 1) void desk.loadActivity(actor.value, 0, true);
}
</script>

<template>
  <DeskLayout title="Activity" nav="activity" :reload="reload">
    <DeskHeader title="Activity" lede="Who changed what, and when: the team, buyers on the website, and the desk itself." />
    <nav class="desk-tabs" aria-label="Whose activity">
      <RouterLink v-for="t in TABS" :key="t.key" :to="{ name: 'desk-activity', query: pageQuery(t.key, 1) }" :aria-current="actor === t.key ? 'page' : undefined">{{
        t.label
      }}</RouterLink>
    </nav>
    <RequestView :state="desk.activity" :retry="() => desk.loadActivity(actor, (page - 1) * PAGE, true)" loading-label="Loading the log…">
      <template #default="{ data, busy }">
        <p v-if="data.entries.length === 0" class="desk-empty">Nothing yet.</p>
        <template v-else>
          <ol class="activity" role="list" :aria-busy="busy ? 'true' : undefined">
            <li v-for="e in data.entries" :key="e.id" class="activity__item">
              <span class="activity__when mono">{{ when(e.at, desk.now) }}</span>
              <span class="activity__who">
                <template v-if="e.actor === 'staff'"><span class="desk-dot" :style="{ backgroundColor: dotColour(e.colour) }" aria-hidden="true" /> {{ e.staff }}</template>
                <span v-else-if="e.actor === 'visitor'" class="activity__tag">Website</span>
                <span v-else class="activity__tag activity__tag--system">Automatic</span>
              </span>
              <span class="activity__what"
                >{{ activityWords(e.action).before }}<RouterLink v-if="e.link.startsWith('/desk')" :to="e.link">{{ e.subject }}</RouterLink
                ><strong v-else>{{ e.subject }}</strong>{{ activityWords(e.action).after }}
                <span v-if="e.detail" class="activity__detail">{{ e.detail }}</span>
              </span>
            </li>
          </ol>
          <nav class="activity__pages" aria-label="Pages">
            <RouterLink v-if="page > 1" class="btn btn--line btn--small" :to="{ name: 'desk-activity', query: pageQuery(actor, page - 1) }">Newer</RouterLink>
            <RouterLink v-if="data.more" class="btn btn--line btn--small" :to="{ name: 'desk-activity', query: pageQuery(actor, page + 1) }">Older</RouterLink>
          </nav>
        </template>
      </template>
    </RequestView>
  </DeskLayout>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import RequestView from "@/components/RequestView.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import DeskLayout from "@/components/desk/DeskLayout.vue";
import ReservationCard from "@/components/desk/ReservationCard.vue";
import type { DeskReservationResult, ReservationGroup } from "@/api/desk";
import type { LiveEvent } from "@/composables/useLiveEvents";
import { usePageData } from "@/composables/usePageData";
import { useDeskStore } from "@/stores/desk";
import { RESERVATION_GROUPS, useDeskHomesStore } from "@/stores/deskHomes";

const route = useRoute("desk-reservations");
const router = useRouter();
const desk = useDeskStore();
const homes = useDeskHomesStore();

const TABS: { key: ReservationGroup; label: string }[] = [
  { key: "waiting", label: "Waiting" },
  { key: "confirmed", label: "Awaiting deposit" },
  { key: "completed", label: "Sold" },
  { key: "closed", label: "Closed" },
];

const tab = computed<ReservationGroup>(() => {
  const t = String(route.query.tab ?? "");
  return (RESERVATION_GROUPS as string[]).includes(t) ? (t as ReservationGroup) : "waiting";
});

usePageData(
  async () => (route.name === "desk-reservations" ? homes.loadReservations(tab.value) : undefined),
  () => tab.value,
);

function reload(event: LiveEvent) {
  if (event.type === "reservation" || event.type === "unit") void homes.loadReservations(tab.value, true);
}

// A decision moves the reservation to its new tab, where the page opens at it.
async function decided(result: DeskReservationResult) {
  const group = result.reservation.group;
  await homes.loadReservations(group, true);
  await router.push({ query: { tab: group }, hash: `#${result.reservation.ref}` });
  desk.notice = result.notice;
  void desk.loadSession(true);
}

const empty: Record<ReservationGroup, string> = {
  waiting: "No requests are waiting. New ones from the website appear here at once.",
  confirmed: "No reservation is waiting for its deposit.",
  completed: "Nothing here yet.",
  closed: "Nothing here yet.",
};
</script>

<template>
  <DeskLayout title="Reservations" nav="reservations" :reload="reload">
    <DeskHeader
      title="Reservations"
      lede="Requests from the website hold a home for 72 hours. Confirm one to issue the letter of offer and reserve the home; record the deposit to sell it."
    />
    <RequestView :state="homes.reservations" :retry="() => homes.loadReservations(tab, true)" loading-label="Loading the reservations…">
      <template #default="{ data, busy }">
        <nav class="desk-tabs" aria-label="Reservations">
          <RouterLink v-for="t in TABS" :key="t.key" :to="{ name: 'desk-reservations', query: { tab: t.key } }" :aria-current="tab === t.key ? 'page' : undefined"
            >{{ t.label }}<span class="tnum">{{ data.counts[t.key] ?? 0 }}</span></RouterLink
          >
        </nav>
        <p v-if="data.reservations.length === 0" class="desk-empty">{{ empty[data.group] }}</p>
        <ol class="res-list" role="list" :aria-busy="busy ? 'true' : undefined">
          <ReservationCard
            v-for="r in data.reservations"
            :key="r.ref"
            :r="r"
            :manager="desk.manager"
            :now="desk.now"
            :target="route.hash === `#${r.ref}`"
            @decided="decided"
          />
        </ol>
      </template>
    </RequestView>
  </DeskLayout>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import ArrowIcon from "@/components/ArrowIcon.vue";
import DoneMark from "@/components/DoneMark.vue";
import FloorPlan from "@/components/FloorPlan.vue";
import RequestView from "@/components/RequestView.vue";
import RollingKes from "@/components/RollingKes.vue";
import NotFoundPage from "@/pages/NotFoundPage.vue";
import type { Reservation } from "@/api/client";
import { useHttpStatus } from "@/composables/useHttpStatus";
import { useLiveEvents } from "@/composables/useLiveEvents";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { kes, longDateTime, percent, shortDate } from "@/lib/format";
import { useHomesStore } from "@/stores/homes";
import { useReservationsStore } from "@/stores/reservations";

const route = useRoute("reservation");
const reservations = useReservationsStore();
const homes = useHomesStore();
const setStatus = useHttpStatus();

const ref_ = computed(() => route.params.ref);
const state = computed(() => reservations.reservation(ref_.value));
const missing = computed(() => state.value.error?.status === 404);
// The home's plan, for the summary beside the steps.
const detail = computed(() => {
  const id = state.value.data?.home.id;
  return id ? homes.detail(id).data : null;
});

usePageData(async () => {
  await reservations.loadReservation(ref_.value);
  const r = reservations.reservation(ref_.value);
  if (r.error?.status === 404) setStatus(404);
  if (r.data) await homes.loadHome(r.data.home.id);
}, ref_);

// The page changes the moment the sales team decides, without a reload.
useLiveEvents("/api/events", (e) => {
  if (e.type === "reservation" && e.id === ref_.value) void reservations.loadReservation(ref_.value, true);
});

usePageMeta(() =>
  missing.value
    ? { title: "Page not found", description: "This page could not be found at Oriel, Riverside.", path: route.path, sheet: { number: "A-711", name: `Reservation ${ref_.value}` }, quiet: true }
    : {
        title: `Reservation ${ref_.value}`,
        description: `Your reservation at Oriel, Riverside.`,
        path: `/reservations/${ref_.value}`,
        sheet: { number: "A-711", name: `Reservation ${ref_.value}` },
        quiet: true,
      },
);

const statusNames: Record<string, string> = {
  requested: "Waiting for confirmation",
  confirmed: "Confirmed",
  declined: "Declined",
  lapsed: "Lapsed",
  completed: "Deposit received",
  cancelled: "Cancelled",
};

function hoursLeft(seconds: number): string {
  if (seconds <= 0) return "now";
  const h = Math.round(seconds / 3600);
  if (h >= 2) return `in ${h} hours`;
  const m = Math.floor(seconds / 60);
  return m <= 1 ? "in a minute" : `in ${m} minutes`;
}

interface Step {
  title: string;
  when: string;
  state: "done" | "current" | "todo" | "stopped";
  note: string;
}

function steps(r: Reservation): Step[] {
  const s: Step[] = [
    { title: "Requested", when: longDateTime(r.requested), state: "done", note: "Home held while the sales team checks your request." },
    { title: "Letter of offer", when: "", state: "todo", note: "We confirm the home, the price and your payment plan." },
    { title: "Deposit", when: "", state: "todo", note: "Pay 10% by bank transfer within 7 days to secure the home." },
  ];
  switch (r.status) {
    case "requested":
      s[1]!.state = "current";
      s[1]!.note = `Held until ${longDateTime(r.holdEnds)}, ${hoursLeft(r.holdSecondsLeft)}.`;
      break;
    case "confirmed":
      Object.assign(s[1]!, { state: "done", when: longDateTime(r.decided), note: `Letter of offer ${r.offerNumber} issued.` });
      s[2]!.state = "current";
      break;
    case "completed":
      Object.assign(s[1]!, { state: "done", when: longDateTime(r.decided), note: `Letter of offer ${r.offerNumber} issued.` });
      Object.assign(s[2]!, { state: "done", when: longDateTime(r.deposit), note: `Deposit of ${kes(r.depositAmount)} received. The home is yours.` });
      break;
    case "declined":
      Object.assign(s[1]!, { state: "stopped", when: longDateTime(r.decided), note: `Declined: ${r.declineReason}` });
      break;
    case "lapsed":
      Object.assign(s[1]!, { state: "stopped", when: longDateTime(r.holdEnds), note: "The hold lapsed before it was confirmed, and the home was released." });
      break;
  }
  return s;
}

const firstName = (full: string) => full.split(" ")[0] ?? full;
const total = (r: Reservation) => r.schedule.instalments.reduce((sum, i) => sum + i.amount, 0);
</script>

<template>
  <NotFoundPage v-if="missing" />
  <RequestView v-else :state="state" :retry="() => reservations.loadReservation(ref_, true)" loading-label="Loading your reservation…">
    <template #default="{ data: r }">
      <div class="wrap form-page">
        <header class="page-head done-head">
          <DoneMark :ok="r.status !== 'declined' && r.status !== 'lapsed'" />
          <p class="sheet-tag label">Reservation {{ r.ref }}</p>
          <h1 class="h1" aria-live="polite">
            <template v-if="r.status === 'requested'">Home {{ r.home.id }} is held for you.</template>
            <template v-else-if="r.status === 'confirmed'">Your letter of offer is ready.</template>
            <template v-else-if="r.status === 'completed'">Home {{ r.home.id }} is yours.</template>
            <template v-else>{{ statusNames[r.status] ?? r.status }}.</template>
          </h1>
          <p class="lede">
            <template v-if="r.status === 'requested'">Thank you, {{ firstName(r.name) }}. The sales team will confirm within one working day; this page updates the moment they do.</template>
            <template v-else-if="r.status === 'confirmed'">The sales team has confirmed your reservation. Download your letter of offer below.</template>
            <template v-else-if="r.status === 'completed'">Your deposit has been received. Welcome to Oriel.</template>
            <template v-else>This reservation is closed.</template>
          </p>
        </header>
        <div class="form-grid">
          <div class="booking-card">
            <ol class="steps" role="list">
              <li v-for="s in steps(r)" :key="s.title" :class="['steps__item', `is-${s.state}`]">
                <span class="steps__dot" aria-hidden="true" />
                <div>
                  <p class="steps__title">{{ s.title }}</p>
                  <p v-if="s.when" class="steps__when mono">{{ s.when }}</p>
                  <p class="steps__note">{{ s.note }}</p>
                </div>
              </li>
            </ol>
            <div v-if="r.status === 'confirmed' || r.status === 'completed'" class="booking-actions">
              <a class="btn btn--cobalt" :href="`/api/reservations/${r.ref}/offer.pdf`">Download the letter of offer (PDF)</a>
            </div>
            <h2 class="h3 schedule-title">Your payments</h2>
            <div class="schedule-card">
              <div class="schedule-card__head">
                <div>
                  <span class="label">{{ r.schedule.plan.name }}</span>
                  <p class="schedule-card__total tnum"><RollingKes :amount="r.schedule.price" /></p>
                  <p v-if="r.schedule.discount > 0" class="schedule-card__saving">{{ kes(r.schedule.discount) }} off the list price of {{ kes(r.schedule.listPrice) }}</p>
                </div>
              </div>
              <table class="pay-table">
                <caption class="visually-hidden">Payments on the {{ r.schedule.plan.name }} plan</caption>
                <thead>
                  <tr><th scope="col">Payment</th><th scope="col">Due</th><th scope="col" class="num">Share</th><th scope="col" class="num">Amount</th></tr>
                </thead>
                <tbody>
                  <tr v-for="(i, n) in r.schedule.instalments" :key="n">
                    <th scope="row">{{ i.label }}<span v-if="i.note" class="pay-table__note">{{ i.note }}</span></th>
                    <td class="mono">{{ i.due ? shortDate(i.due) : "" }}</td>
                    <td class="num mono">{{ percent(i.percentBP) }}</td>
                    <td class="num tnum">{{ kes(i.amount) }}</td>
                  </tr>
                </tbody>
                <tfoot>
                  <tr><th scope="row" colspan="3">Total</th><td class="num tnum">{{ kes(total(r)) }}</td></tr>
                </tfoot>
              </table>
            </div>
          </div>
          <aside class="form-side">
            <div class="side-card reserve-summary">
              <p class="reserve-summary__id">{{ r.home.id }}</p>
              <h2 class="h3">{{ r.home.title }}</h2>
              <div v-if="detail" class="reserve-summary__plan">
                <FloorPlan :plan="detail.plan" plain :north="detail.north" :id="`rv-${r.home.id}`" :title="`Plan of home ${r.home.id}`" />
              </div>
              <p class="muted">Faces {{ r.home.facings }} · {{ r.home.internal }} m²</p>
              <dl class="booking-facts booking-facts--small">
                <div><dt class="label">Reserved by</dt><dd>{{ r.name }}</dd></div>
                <div><dt class="label">Price</dt><dd class="tnum">{{ kes(r.price) }}</dd></div>
                <div><dt class="label">Status</dt><dd>{{ statusNames[r.status] ?? r.status }}</dd></div>
                <div v-if="r.agent"><dt class="label">Your consultant</dt><dd>{{ r.agent }}</dd></div>
              </dl>
              <RouterLink class="link-arrow" :to="{ name: 'home-detail', params: { id: r.home.id } }">See the home <ArrowIcon /></RouterLink>
            </div>
          </aside>
        </div>
      </div>
    </template>
  </RequestView>
</template>

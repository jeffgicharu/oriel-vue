<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import ArrowIcon from "@/components/ArrowIcon.vue";
import DoneMark from "@/components/DoneMark.vue";
import RequestView from "@/components/RequestView.vue";
import NotFoundPage from "@/pages/NotFoundPage.vue";
import { isApiError } from "@/api/request";
import { useHttpStatus } from "@/composables/useHttpStatus";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { longDateTime } from "@/lib/format";
import { useContactStore } from "@/stores/contact";

const route = useRoute("viewing");
const contact = useContactStore();
const setStatus = useHttpStatus();
const ref_ = computed(() => route.params.ref);
const state = computed(() => contact.viewing(ref_.value));
const missing = computed(() => state.value.error?.status === 404);

usePageData(async () => {
  await contact.loadViewing(ref_.value);
  if (contact.viewing(ref_.value).error?.status === 404) setStatus(404);
}, ref_);

usePageMeta(() =>
  missing.value
    ? { title: "Page not found", description: "This page could not be found at Oriel, Riverside.", path: route.path, sheet: { number: "A-404", name: "Not found" } }
    : {
        title: `Viewing ${ref_.value}`,
        description: "Your viewing at Oriel's sales suite in Riverside, Nairobi.",
        path: `/viewings/${ref_.value}`,
        sheet: { number: "A-701", name: `Viewing ${ref_.value}` },
        quiet: true,
      },
);

const firstName = (full: string) => full.split(" ")[0] ?? full;
const cancelling = ref(false);
const cancelError = ref("");

async function cancel() {
  if (!window.confirm("Cancel this viewing?")) return;
  cancelling.value = true;
  cancelError.value = "";
  try {
    await contact.cancelViewing(ref_.value);
  } catch (e) {
    cancelError.value = isApiError(e) ? e.detail || e.title : "The viewing could not be cancelled just now. Please try again.";
  } finally {
    cancelling.value = false;
  }
}
</script>

<template>
  <NotFoundPage v-if="missing" />
  <RequestView v-else :state="state" :retry="() => contact.loadViewing(ref_, true)" loading-label="Loading your viewing…">
    <template #default="{ data: v }">
      <div class="wrap form-page">
        <header class="page-head done-head">
          <DoneMark :ok="v.status !== 'cancelled'" />
          <p class="sheet-tag label">Viewing {{ v.ref }}</p>
          <template v-if="v.status === 'cancelled'">
            <h1 class="h1">Viewing cancelled.</h1>
            <p class="lede">Your visit on {{ longDateTime(v.start) }} is cancelled. You are welcome to book another time.</p>
          </template>
          <template v-else-if="v.status === 'booked'">
            <h1 class="h1">You’re booked, {{ firstName(v.name) }}.</h1>
            <p class="lede">We look forward to showing you round on {{ longDateTime(v.start) }}. Keep this page: it is your booking.</p>
          </template>
          <template v-else>
            <h1 class="h1">Thank you for visiting.</h1>
            <p class="lede">Your viewing on {{ longDateTime(v.start) }} has taken place.</p>
          </template>
        </header>
        <div class="form-grid">
          <div class="booking-card">
            <dl class="booking-facts">
              <div><dt class="label">When</dt><dd>{{ longDateTime(v.start) }}</dd></div>
              <div><dt class="label">Where</dt><dd>The sales suite, Oriel, Riverside, Nairobi</dd></div>
              <div><dt class="label">Party</dt><dd>{{ v.party }} {{ v.party === 1 ? "person" : "people" }}</dd></div>
              <div v-if="v.home">
                <dt class="label">Home</dt>
                <dd><RouterLink :to="{ name: 'home-detail', params: { id: v.home.id } }"><span class="mono">{{ v.home.id }}</span>, {{ v.home.title }}</RouterLink></dd>
              </div>
              <div v-if="v.notes"><dt class="label">Your note</dt><dd class="booking-facts__note">{{ v.notes }}</dd></div>
              <div><dt class="label">Reference</dt><dd class="mono">{{ v.ref }}</dd></div>
              <div v-if="v.agent"><dt class="label">Your consultant</dt><dd>{{ v.agent }}</dd></div>
            </dl>
            <template v-if="v.status === 'booked'">
              <div class="booking-actions">
                <a class="btn btn--cobalt" :href="`/api/viewings/${v.ref}/calendar.ics`" download>Add to calendar</a>
                <RouterLink class="btn btn--line" :to="{ name: 'location', hash: '#getting-here' }">Getting here</RouterLink>
              </div>
              <div class="cancel-form">
                <button class="link-button" type="button" :disabled="cancelling" @click="cancel">{{ cancelling ? "Cancelling…" : "Cancel this viewing" }}</button>
                <p v-if="cancelError" class="field__error" role="alert">{{ cancelError }}</p>
              </div>
            </template>
            <div v-else-if="v.status === 'cancelled'" class="booking-actions">
              <RouterLink class="btn btn--cobalt" :to="{ name: 'viewing-new' }">Book another time <ArrowIcon /></RouterLink>
            </div>
          </div>
          <aside class="form-side">
            <div v-if="v.status === 'booked'" class="side-card">
              <h2 class="label">Before you come</h2>
              <ul class="ticks" role="list">
                <li>Look through the homes and their plans to pick the ones you’d like to see.</li>
                <li>Shortlist homes as you browse; we’ll have their plans ready.</li>
              </ul>
              <RouterLink class="link-arrow" :to="{ name: 'residences' }">See the homes <ArrowIcon /></RouterLink>
            </div>
            <div v-else class="side-card">
              <h2 class="label">Still deciding?</h2>
              <p>Ask the sales team about a home, prices or buying off-plan, and they will reply within one working day.</p>
              <RouterLink class="link-arrow" :to="{ name: 'enquire' }">Ask a question <ArrowIcon /></RouterLink>
            </div>
          </aside>
        </div>
      </div>
    </template>
  </RequestView>
</template>

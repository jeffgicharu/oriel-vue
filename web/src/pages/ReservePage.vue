<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import ArrowIcon from "@/components/ArrowIcon.vue";
import ErrorSummary from "@/components/ErrorSummary.vue";
import FloorPlan from "@/components/FloorPlan.vue";
import FormField from "@/components/FormField.vue";
import HoneyPot from "@/components/HoneyPot.vue";
import RequestView from "@/components/RequestView.vue";
import NotFoundPage from "@/pages/NotFoundPage.vue";
import { useForm } from "@/composables/useForm";
import { useHttpStatus } from "@/composables/useHttpStatus";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { kes, kesShort } from "@/lib/format";
import { statusName } from "@/lib/homes";
import { contactErrors, textErrors } from "@/lib/validation";
import { useHomesStore } from "@/stores/homes";
import { useReservationsStore, type ReserveValues } from "@/stores/reservations";

const route = useRoute("reserve");
const router = useRouter();
const homes = useHomesStore();
const reservations = useReservationsStore();
const setStatus = useHttpStatus();

const id = computed(() => route.params.id);
const state = computed(() => homes.detail(id.value));
const missing = computed(() => state.value.error?.status === 404);

usePageData(async () => {
  await homes.loadHome(id.value);
  if (homes.detail(id.value).error?.status === 404) setStatus(404);
}, id);

usePageMeta(() =>
  missing.value
    ? { title: "Page not found", description: "This page could not be found at Oriel, Riverside.", path: route.path, sheet: { number: "A-404", name: "Not found" } }
    : {
        title: `Reserve home ${id.value}`,
        description: `Request a reservation on home ${id.value} at Oriel, Riverside. The home is held for 72 hours while the sales team confirms it with a letter of offer.`,
        path: `/reserve/${id.value}`,
        sheet: { number: "A-710", name: `Reserve ${id.value}` },
        quiet: true,
      },
);

const startPlan = computed(() => {
  const q = String(route.query.plan ?? "");
  return ["staged", "cash", "mortgage"].includes(q) ? q : "staged";
});

const form = useForm<ReserveValues, unknown>({
  id: "reserve",
  values: { name: "", email: "", phone: "", plan: startPlan.value, buying: "home", note: "", agree: false, website: "" },
  order: ["home", "plan", "name", "phone", "email", "note", "agree"],
  check: (v) => ({
    ...(["staged", "cash", "mortgage"].includes(v.plan) ? {} : { plan: "Please choose how you will pay." }),
    ...contactErrors(v),
    ...textErrors("note", v.note, { max: 600, tooLong: "Please keep this under 600 characters." }),
    ...(v.agree ? {} : { agree: "Please confirm you understand how a reservation works." }),
  }),
  send: async (v) => {
    const made = await reservations.reserve(id.value, v);
    // The home is now held: its page and the lists read it afresh.
    void homes.loadHome(id.value, true);
    await router.push({ name: "reservation", params: { ref: made.ref }, query: { new: "1" } });
    return made;
  },
});

// Someone else reserved the home while the buyer filled in the form.
const takenMeanwhile = computed(() => form.errors.value.home !== undefined);
</script>

<template>
  <NotFoundPage v-if="missing" />
  <RequestView v-else :state="state" :retry="() => homes.loadHome(id, true)" loading-label="Loading the home…">
    <template #default="{ data }">
      <div class="wrap form-page">
        <header class="page-head">
          <p class="sheet-tag label">A-710 · Reserve</p>
          <h1 class="h1">Request a reservation.</h1>
          <p class="lede">Tell us how you would like to pay. We hold the home for 72 hours while the sales team confirms it with a letter of offer. No payment is taken here.</p>
        </header>
        <template v-if="data.home.status !== 'available' || takenMeanwhile">
          <div class="notice notice--alert" role="alert">
            <strong>Home {{ data.home.id }} is no longer available.</strong>
            <span v-if="data.similar.length"> It is {{ takenMeanwhile ? "on hold" : statusName(data.home.status).toLowerCase() }}. These are the closest still for sale:</span>
            <span v-else>
              It is {{ statusName(data.home.status).toLowerCase() }}, and no other home of its type is for sale just now.
              <RouterLink :to="{ name: 'residences' }">See every home for sale</RouterLink> or <RouterLink :to="{ name: 'enquire' }">ask to hear if one comes free</RouterLink>.
            </span>
          </div>
          <ul class="similar similar--wide" role="list">
            <li v-for="s in data.similar" :key="s.id">
              <RouterLink class="similar__card" :to="{ name: 'reserve', params: { id: s.id } }">
                <span class="mono similar__id">{{ s.id }}</span>
                <span>{{ s.title }}</span>
                <span class="muted">Faces {{ s.facings }}</span>
                <span v-if="s.price !== null" class="tnum similar__price">{{ kesShort(s.price) }}</span>
              </RouterLink>
            </li>
          </ul>
        </template>
        <div v-else class="form-grid">
          <form class="form-main" novalidate @submit.prevent="form.submit">
            <ErrorSummary id="reserve" :listed="form.listed.value" :problem="form.problem.value" @retry="form.submit" />
            <fieldset class="step">
              <legend class="step__title"><span class="step__n mono">1</span> How you’ll pay</legend>
              <div id="reserve-plan" class="payplans__choices">
                <label v-for="s in data.schedules" :key="s.plan.code" class="choice payplan">
                  <input v-model="form.values.plan" type="radio" name="plan" :value="s.plan.code" @change="form.touched('plan')" />
                  <span>
                    <strong>{{ s.plan.name }}</strong>
                    <span class="muted">{{ s.plan.summary }}</span>
                    <span class="payplan__deposit mono">Deposit {{ kes(s.instalments[0]?.amount ?? 0) }} · total {{ kes(s.price) }}</span>
                  </span>
                </label>
              </div>
              <p v-if="form.errors.value.plan" class="field__error">{{ form.errors.value.plan }}</p>
            </fieldset>
            <fieldset class="step">
              <legend class="step__title"><span class="step__n mono">2</span> Your details</legend>
              <div class="field-grid">
                <FormField
                  id="reserve-name"
                  v-model="form.values.name"
                  label="Full name, as it will appear on the letter of offer"
                  autocomplete="name"
                  :error="form.errors.value.name"
                  @touched="form.touched('name')"
                />
                <FormField id="reserve-phone" v-model="form.values.phone" label="Phone" type="tel" autocomplete="tel" :error="form.errors.value.phone" @touched="form.touched('phone')" />
                <FormField id="reserve-email" v-model="form.values.email" label="Email" type="email" autocomplete="email" :error="form.errors.value.email" @touched="form.touched('email')" />
                <div class="field" role="radiogroup" aria-labelledby="reserve-buying-label">
                  <span id="reserve-buying-label" class="field__label">Buying</span>
                  <div class="chips chips--field">
                    <label class="chip"><input v-model="form.values.buying" type="radio" name="buying" value="home" /><span>To live in</span></label>
                    <label class="chip"><input v-model="form.values.buying" type="radio" name="buying" value="investment" /><span>To let</span></label>
                  </div>
                </div>
              </div>
              <div class="field" :class="{ 'field--error': form.errors.value.note }">
                <label class="field__label" for="reserve-note">Anything to add <span class="label">Optional</span></label>
                <textarea id="reserve-note" v-model="form.values.note" class="textarea" maxlength="600" @input="form.touched('note')" />
                <p v-if="form.errors.value.note" class="field__error">{{ form.errors.value.note }}</p>
              </div>
            </fieldset>
            <div class="field" :class="{ 'field--error': form.errors.value.agree }">
              <label class="choice">
                <input id="reserve-agree" v-model="form.values.agree" type="checkbox" :aria-invalid="form.errors.value.agree ? 'true' : undefined" @change="form.touched('agree')" />
                <span>I understand this is a request. The home is held for 72 hours while the sales team confirms it with a letter of offer, and the 10% deposit is then paid by bank transfer.</span>
              </label>
              <p v-if="form.errors.value.agree" class="field__error">{{ form.errors.value.agree }}</p>
            </div>
            <HoneyPot v-model="form.values.website" />
            <button class="btn btn--cobalt btn--wide-phone" type="submit" :disabled="form.status.value === 'sending'">
              {{ form.status.value === "sending" ? "Sending…" : "Request the reservation" }} <ArrowIcon />
            </button>
          </form>
          <aside class="form-side">
            <div class="side-card reserve-summary">
              <p class="reserve-summary__id">{{ data.home.id }}</p>
              <h2 class="h3">{{ data.home.title }}</h2>
              <div class="reserve-summary__plan">
                <FloorPlan :plan="data.plan" plain :north="data.north" :id="`rs-${data.home.id}`" :title="`Plan of home ${data.home.id}`" />
              </div>
              <p class="muted">Faces {{ data.home.facings }} · {{ data.home.internal }} m² · {{ data.home.outdoorName }} {{ data.home.outdoor }} m²</p>
              <p v-if="data.home.price !== null" class="reserve-summary__price tnum">{{ kes(data.home.price) }}</p>
              <RouterLink class="link-arrow" :to="{ name: 'home-detail', params: { id: data.home.id } }">Back to the home <ArrowIcon /></RouterLink>
            </div>
          </aside>
        </div>
      </div>
    </template>
  </RequestView>
</template>

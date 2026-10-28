<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import ArrowIcon from "@/components/ArrowIcon.vue";
import ErrorSummary from "@/components/ErrorSummary.vue";
import FormField from "@/components/FormField.vue";
import HoneyPot from "@/components/HoneyPot.vue";
import PhotoPicture from "@/components/PhotoPicture.vue";
import RequestView from "@/components/RequestView.vue";
import type { Calendar } from "@/api/client";
import { useForm } from "@/composables/useForm";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { dayParts, longDate } from "@/lib/format";
import { contactErrors, textErrors } from "@/lib/validation";
import { useContactStore, type ViewingValues } from "@/stores/contact";
import { useHomesStore } from "@/stores/homes";

const route = useRoute("viewing-new");
const router = useRouter();
const contact = useContactStore();
const homes = useHomesStore();

usePageMeta({
  title: "Book a viewing",
  description: "Book a 45-minute visit to Oriel's sales suite in Riverside, Nairobi: the show apartments, the building model and the team. Open Tuesday to Sunday.",
  path: "/viewings/new",
  sheet: { number: "A-700", name: "Viewings" },
  quiet: true,
});

// A visit can be about one home, when it was booked from the home's page.
const homeId = computed(() => {
  const h = String(route.query.home ?? "");
  return /^\d{4}$/.test(h) ? h : "";
});
const home = computed(() => (homeId.value ? homes.detail(homeId.value).data?.home ?? null : null));

usePageData(() => Promise.all([contact.loadCalendar(), homeId.value ? homes.loadHome(homeId.value) : Promise.resolve()]));

type Day = Calendar["days"][number];
const usable = (d: Day) => d.open && d.free > 0;
const chosenDay = ref("");
const days = computed(() => contact.calendar.data?.days ?? []);
const day = computed(() => days.value.find((d) => d.date === chosenDay.value) ?? null);

// The first day with room is chosen to begin with.
watch(
  days,
  (all) => {
    if (!all.some((d) => d.date === chosenDay.value && usable(d))) chosenDay.value = all.find(usable)?.date ?? "";
  },
  { immediate: true },
);

// Closures worth pointing out under the days: a holiday, a day the sales team
// has closed, or a day already full. The weekly day off is in the hours.
const notes = computed(() =>
  days.value
    .filter((d) => d.blocked || (d.open && d.free === 0))
    .map((d) => {
      const p = dayParts(d.date);
      return { when: `${p.weekday} ${p.day} ${p.month}`, text: d.open ? "Fully booked" : d.reason };
    }),
);

function placesLeft(n: number) {
  return n === 0 ? "Full" : n === 1 ? "1 place" : `${n} places`;
}

const form = useForm<ViewingValues, unknown>({
  id: "viewing",
  values: { name: "", email: "", phone: "", party: 2, notes: "", slot: "", website: "", home: "" },
  order: ["slot", "name", "phone", "email", "party", "notes"],
  check: (v) => ({
    ...(v.slot ? {} : { slot: "Please choose a time." }),
    ...contactErrors(v),
    ...textErrors("notes", v.notes, { max: 600, tooLong: "Please keep this under 600 characters." }),
  }),
  send: async (v) => {
    const booked = await contact.bookViewing({ ...v, home: homeId.value });
    await router.push({ name: "viewing", params: { ref: booked.ref }, query: { booked: "1" } });
    return booked;
  },
});

function chooseDay(date: string) {
  chosenDay.value = date;
  if (form.values.slot && !form.values.slot.startsWith(date)) form.values.slot = "";
  form.touched("slot");
}

// A time taken by someone else while the visitor was filling in the form
// sends them back to choose again, with the calendar brought up to date.
watch(
  () => form.problem.value ?? form.errors.value.slot,
  () => {
    if (form.errors.value.slot?.includes("just been taken")) {
      form.values.slot = "";
      void contact.loadCalendar(true);
    }
  },
);
</script>

<template>
  <div class="wrap form-page">
    <header class="page-head">
      <p class="sheet-tag label">A-700 · Viewings</p>
      <h1 class="h1">Book a viewing.</h1>
      <p class="lede">Walk round the show apartments and the building model in our sales suite on site, with one of the sales team. A visit takes about 45 minutes.</p>
    </header>
    <div class="form-grid">
      <form class="form-main" novalidate @submit.prevent="form.submit">
        <ErrorSummary id="viewing" :listed="form.listed.value" :problem="form.problem.value" @retry="form.submit" />
        <div v-if="home" class="coming-to">
          <span class="label">You’re coming to see</span>
          <RouterLink :to="{ name: 'home-detail', params: { id: home.id } }"><span class="mono">{{ home.id }}</span> {{ home.title }}</RouterLink>
        </div>
        <RequestView :state="contact.calendar" :retry="() => contact.loadCalendar(true)" loading-label="Loading the calendar…">
          <template #default>
            <fieldset class="step">
              <legend class="step__title"><span class="step__n mono">1</span> Choose a day</legend>
              <div class="days">
                <label v-for="d in days" :key="d.date" class="day" :class="{ 'is-closed': !usable(d) }" :title="d.open ? (d.free === 0 ? 'Fully booked' : '') : d.reason">
                  <input type="radio" name="date" :value="d.date" :checked="d.date === chosenDay" :disabled="!usable(d)" @change="chooseDay(d.date)" />
                  <span class="day__name">{{ dayParts(d.date).weekday }}</span>
                  <span class="day__num">{{ dayParts(d.date).day }}</span>
                  <span class="day__month">{{ dayParts(d.date).month }}</span>
                </label>
              </div>
              <ul v-if="notes.length" class="days__notes" role="list">
                <li v-for="n in notes" :key="n.when"><span class="mono">{{ n.when }}</span> {{ n.text }}</li>
              </ul>
            </fieldset>
            <fieldset class="step">
              <legend class="step__title"><span class="step__n mono">2</span> Choose a time</legend>
              <div id="viewing-slot" class="slots" :class="{ 'field--error': form.errors.value.slot }">
                <p v-if="!day || !day.open || day.slots.length === 0" class="muted">{{ day?.reason ? `${day.reason}. Please choose another day.` : "Please choose a day first." }}</p>
                <template v-else>
                  <p class="slots__day">{{ longDate(day.date) }}</p>
                  <div class="slots__grid">
                    <label v-for="s in day.slots" :key="s.start" class="slot" :class="{ 'is-full': s.left === 0 }">
                      <input v-model="form.values.slot" type="radio" name="slot" :value="s.start" :disabled="s.left === 0" required @change="form.touched('slot')" />
                      <span class="slot__time mono">{{ s.time }}</span>
                      <span class="slot__left">{{ placesLeft(s.left) }}</span>
                    </label>
                  </div>
                </template>
                <p v-if="form.errors.value.slot" class="field__error">{{ form.errors.value.slot }}</p>
              </div>
            </fieldset>
          </template>
        </RequestView>
        <fieldset class="step">
          <legend class="step__title"><span class="step__n mono">3</span> Your details</legend>
          <div class="field-grid">
            <FormField id="viewing-name" v-model="form.values.name" label="Full name" autocomplete="name" :error="form.errors.value.name" @touched="form.touched('name')" />
            <FormField
              id="viewing-phone"
              v-model="form.values.phone"
              label="Phone"
              type="tel"
              autocomplete="tel"
              hint="We will call only about your viewing."
              :error="form.errors.value.phone"
              @touched="form.touched('phone')"
            />
            <FormField id="viewing-email" v-model="form.values.email" label="Email" type="email" autocomplete="email" :error="form.errors.value.email" @touched="form.touched('email')" />
            <div class="field">
              <label class="field__label" for="viewing-party">How many of you</label>
              <select id="viewing-party" v-model.number="form.values.party" class="select">
                <option v-for="n in 6" :key="n" :value="n">{{ n }} {{ n === 1 ? "person" : "people" }}</option>
              </select>
            </div>
          </div>
          <div class="field" :class="{ 'field--error': form.errors.value.notes }">
            <label class="field__label" for="viewing-notes">Anything we should know <span class="label">Optional</span></label>
            <textarea id="viewing-notes" v-model="form.values.notes" class="textarea" maxlength="600" @input="form.touched('notes')" />
            <p v-if="form.errors.value.notes" class="field__error">{{ form.errors.value.notes }}</p>
          </div>
        </fieldset>
        <HoneyPot v-model="form.values.website" />
        <button class="btn btn--cobalt btn--wide-phone" type="submit" :disabled="form.status.value === 'sending'">
          {{ form.status.value === "sending" ? "Booking…" : "Book this viewing" }} <ArrowIcon />
        </button>
        <p class="form-small muted">We keep your details only to arrange your visit. See our <RouterLink :to="{ name: 'privacy' }">privacy notice</RouterLink>.</p>
      </form>
      <aside class="form-side">
        <div class="side-card">
          <div class="side-card__photo">
            <PhotoPicture name="sales-suite" alt="A sales consultant showing a couple the white model of the building in a bright room" sizes="(min-width: 1000px) 30vw, 100vw" />
          </div>
          <h2 class="label">The sales suite</h2>
          <p>On site at Oriel, Riverside, Nairobi, with visitor parking at the entrance on Riverside Drive.</p>
          <dl class="hours">
            <div><dt>Tuesday to Sunday</dt><dd class="mono">10:00 to 17:00</dd></div>
            <div><dt>Monday</dt><dd class="mono">Closed</dd></div>
            <div><dt>Public holidays</dt><dd class="mono">Closed</dd></div>
          </dl>
          <RouterLink class="link-arrow" :to="{ name: 'location', hash: '#getting-here' }">Getting here <ArrowIcon /></RouterLink>
        </div>
        <div class="side-card">
          <h2 class="label">On the day</h2>
          <ul class="ticks" role="list">
            <li>See both show apartments: a one-bedroom and a corner two-bedroom.</li>
            <li>Stand in a full-size oriel bay and judge the light for yourself.</li>
            <li>Go through prices and payment plans with the team.</li>
          </ul>
        </div>
      </aside>
    </div>
  </div>
</template>

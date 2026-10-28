<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import ArrowIcon from "@/components/ArrowIcon.vue";
import DoneMark from "@/components/DoneMark.vue";
import ErrorSummary from "@/components/ErrorSummary.vue";
import FormField from "@/components/FormField.vue";
import HoneyPot from "@/components/HoneyPot.vue";
import WhatsAppLink from "@/components/WhatsAppLink.vue";
import { useForm } from "@/composables/useForm";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { contactErrors, textErrors } from "@/lib/validation";
import { useContactStore, type EnquiryValues } from "@/stores/contact";
import { useHomesStore } from "@/stores/homes";

const route = useRoute("enquire");
const contact = useContactStore();
const homes = useHomesStore();

usePageMeta({
  title: "Ask a question",
  description: "Ask Oriel's sales team about a home, the building, prices or buying off-plan in Riverside, Nairobi, by form or on WhatsApp.",
  path: "/enquire",
  sheet: { number: "A-720", name: "Questions" },
  quiet: true,
});

// A question can be about one home, when it was asked from the home's page.
const homeId = computed(() => {
  const h = String(route.query.home ?? "");
  return /^\d{4}$/.test(h) ? h : "";
});
const home = computed(() => (homeId.value ? homes.detail(homeId.value).data?.home ?? null : null));
usePageData(() => (homeId.value ? homes.loadHome(homeId.value) : Promise.resolve()));

const interests = computed(() => [...(home.value ? [`Home ${home.value.id}`] : []), "Not sure yet", "One bedroom", "Two bedrooms", "Three bedrooms", "Three bedrooms, study and DSQ", "Penthouse"]);

const form = useForm<EnquiryValues, Awaited<ReturnType<typeof contact.enquire>>>({
  id: "enquiry",
  values: { name: "", email: "", phone: "", interest: "", message: "", website: "", home: "" },
  order: ["name", "phone", "email", "interest", "message"],
  check: (v) => ({
    ...contactErrors(v),
    ...textErrors("message", v.message, { required: "Please write your question.", max: 1500, tooLong: "Please keep your question under 1,500 characters." }),
  }),
  send: (v) => contact.enquire({ ...v, interest: v.interest || interests.value[0]!, home: homeId.value }),
});
</script>

<template>
  <div class="wrap form-page">
    <header class="page-head">
      <p class="sheet-tag label">A-720 · Questions</p>
      <h1 class="h1">Ask us anything.</h1>
      <p class="lede">About a home, the building, prices or buying off-plan. A sales consultant replies within one working day.</p>
    </header>
    <div class="form-grid">
      <div v-if="form.status.value === 'sent' && form.result.value" class="booking-card" role="status">
        <DoneMark :ok="true" />
        <h2 class="h2">Thank you, {{ form.result.value.firstName }}.</h2>
        <p class="lede">Your question is with the sales team. While you wait, you might like to:</p>
        <div class="booking-actions">
          <RouterLink class="btn btn--cobalt" :to="{ name: 'residences' }">See the homes <ArrowIcon /></RouterLink>
          <RouterLink class="btn btn--line" :to="{ name: 'viewing-new' }">Book a viewing</RouterLink>
        </div>
      </div>
      <form v-else class="form-main" novalidate @submit.prevent="form.submit">
        <div v-if="home" class="coming-to">
          <span class="label">Your question is about</span>
          <RouterLink :to="{ name: 'home-detail', params: { id: home.id } }"><span class="mono">{{ home.id }}</span> {{ home.title }}</RouterLink>
        </div>
        <ErrorSummary id="enquiry" :listed="form.listed.value" :problem="form.problem.value" @retry="form.submit" />
        <div class="field-grid">
          <FormField id="enquiry-name" v-model="form.values.name" label="Full name" autocomplete="name" :error="form.errors.value.name" @touched="form.touched('name')" />
          <FormField id="enquiry-phone" v-model="form.values.phone" label="Phone" type="tel" autocomplete="tel" :error="form.errors.value.phone" @touched="form.touched('phone')" />
          <FormField id="enquiry-email" v-model="form.values.email" label="Email" type="email" autocomplete="email" :error="form.errors.value.email" @touched="form.touched('email')" />
          <div class="field">
            <label class="field__label" for="enquiry-interest">Interested in</label>
            <select id="enquiry-interest" v-model="form.values.interest" class="select">
              <option v-for="(opt, i) in interests" :key="opt" :value="i === 0 ? '' : opt">{{ opt }}</option>
            </select>
          </div>
        </div>
        <div class="field" :class="{ 'field--error': form.errors.value.message }">
          <label class="field__label" for="enquiry-message">Your question</label>
          <textarea
            id="enquiry-message"
            v-model="form.values.message"
            class="textarea"
            maxlength="1500"
            required
            :aria-invalid="form.errors.value.message ? 'true' : undefined"
            :aria-describedby="form.errors.value.message ? 'enquiry-message-note' : undefined"
            @input="form.touched('message')"
          />
          <p v-if="form.errors.value.message" id="enquiry-message-note" class="field__error">{{ form.errors.value.message }}</p>
        </div>
        <HoneyPot v-model="form.values.website" />
        <button class="btn btn--cobalt btn--wide-phone" type="submit" :disabled="form.status.value === 'sending'">
          {{ form.status.value === "sending" ? "Sending…" : "Send the question" }} <ArrowIcon />
        </button>
        <p class="form-small muted">We keep your details only to answer you. See our <RouterLink :to="{ name: 'privacy' }">privacy notice</RouterLink>.</p>
      </form>
      <aside class="form-side">
        <div class="side-card">
          <h2 class="label">Rather message us?</h2>
          <p>Send the sales team a message on WhatsApp{{ home ? ` about home ${home.id}` : "" }}. We reply during opening hours.</p>
          <WhatsAppLink :home="home" />
        </div>
        <div class="side-card">
          <h2 class="label">Rather see it?</h2>
          <p>The sales suite is open Tuesday to Sunday, 10:00 to 17:00.</p>
          <RouterLink class="link-arrow" :to="{ name: 'viewing-new' }">Book a viewing <ArrowIcon /></RouterLink>
        </div>
      </aside>
    </div>
  </div>
</template>

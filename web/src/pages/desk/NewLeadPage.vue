<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";
import ErrorSummary from "@/components/ErrorSummary.vue";
import FormField from "@/components/FormField.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import DeskLayout from "@/components/desk/DeskLayout.vue";
import { useForm } from "@/composables/useForm";
import { usePageData } from "@/composables/usePageData";
import { contactErrors, textErrors } from "@/lib/validation";
import { useDeskStore } from "@/stores/desk";
import { useDeskLeadsStore, type NewLeadValues } from "@/stores/deskLeads";

const router = useRouter();
const desk = useDeskStore();
const leads = useDeskLeadsStore();

usePageData(() => (desk.manager ? desk.loadTeam() : Promise.resolve()));

const SOURCES = ["Walk-in", "Phone call", "Referral", "Agent network", "Site hoarding", "Property portal", "Social media"];
const agents = computed(() => (desk.team.data ?? []).filter((m) => m.role === "agent"));
// The list starts on the first agent, as Oriel's did, until another is chosen.
const agent = computed({
  get: () => form.values.agent || (agents.value[0]?.id ?? 0),
  set: (id: number) => (form.values.agent = id),
});

const form = useForm<NewLeadValues, unknown>({
  id: "lead",
  values: { name: "", phone: "", email: "", source: "Walk-in", interest: "", buying: "home", agent: 0, note: "" },
  order: ["name", "phone", "email", "source", "interest", "agent", "note"],
  check: (v) => ({
    ...contactErrors(v),
    ...textErrors("interest", v.interest, { max: 120, tooLong: "Please keep this under 120 characters." }),
    ...textErrors("note", v.note, { max: 600, tooLong: "Please keep this under 600 characters." }),
  }),
  send: async (v) => {
    const added = await leads.addLead({ ...v, agent: desk.manager ? agent.value : 0 });
    await router.push({ name: "desk-lead", params: { id: String(added.lead.id) } });
    desk.notice = added.notice;
    return added;
  },
});
</script>

<template>
  <DeskLayout title="Add a lead" nav="leads">
    <nav class="breadcrumb label" aria-label="Breadcrumb">
      <RouterLink :to="{ name: 'desk-leads' }">Leads</RouterLink><span aria-hidden="true">/</span><span aria-current="page">New lead</span>
    </nav>
    <DeskHeader title="Add a lead" lede="Someone who called, walked in or was referred." />
    <form class="desk-panel desk-form new-lead" novalidate @submit.prevent="form.submit">
      <ErrorSummary id="lead" :listed="form.listed.value" :problem="form.problem.value" @retry="form.submit" />
      <div class="field-grid">
        <FormField id="lead-name" v-model="form.values.name" label="Full name" :error="form.errors.value.name" @touched="form.touched('name')" />
        <FormField id="lead-phone" v-model="form.values.phone" label="Phone" type="tel" :error="form.errors.value.phone" @touched="form.touched('phone')" />
        <FormField id="lead-email" v-model="form.values.email" label="Email" type="email" :error="form.errors.value.email" @touched="form.touched('email')" />
        <div class="field">
          <label class="field__label" for="lead-source">Came from</label>
          <select id="lead-source" v-model="form.values.source" class="select">
            <option v-for="s in SOURCES" :key="s" :value="s">{{ s }}</option>
          </select>
        </div>
        <FormField
          id="lead-interest"
          v-model="form.values.interest"
          label="Interested in"
          hint="Such as: 2 bed, high floor"
          :required="false"
          :error="form.errors.value.interest"
          @touched="form.touched('interest')"
        />
        <div class="field">
          <label class="field__label" for="lead-buying">Buying</label>
          <select id="lead-buying" v-model="form.values.buying" class="select">
            <option value="home">To live in</option>
            <option value="investment">To let</option>
          </select>
        </div>
        <div v-if="desk.manager" class="field" :class="{ 'field--error': form.errors.value.agent }">
          <label class="field__label" for="lead-agent">Agent</label>
          <select id="lead-agent" v-model.number="agent" class="select" @change="form.touched('agent')">
            <option v-for="m in agents" :key="m.id" :value="m.id">{{ m.name }}</option>
          </select>
          <p v-if="form.errors.value.agent" class="field__error">{{ form.errors.value.agent }}</p>
        </div>
      </div>
      <div class="field" :class="{ 'field--error': form.errors.value.note }">
        <label class="field__label" for="lead-note">Note <span class="label">Optional</span></label>
        <textarea id="lead-note" v-model="form.values.note" class="textarea" maxlength="600" @input="form.touched('note')" />
        <p v-if="form.errors.value.note" class="field__error">{{ form.errors.value.note }}</p>
      </div>
      <button class="btn btn--cobalt btn--small" type="submit" :disabled="form.status.value === 'sending'">Add the lead</button>
    </form>
  </DeskLayout>
</template>

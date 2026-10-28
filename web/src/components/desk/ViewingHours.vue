<script setup lang="ts">
import ErrorSummary from "@/components/ErrorSummary.vue";
import type { DeskHours } from "@/api/desk";
import { useForm } from "@/composables/useForm";
import { useDeskStore } from "@/stores/desk";
import { useDeskViewingsStore } from "@/stores/deskViewings";

/** The sales suite's opening days, viewing times and parties a slot takes. For the manager. */
const props = defineProps<{ hours: DeskHours }>();
const desk = useDeskStore();
const viewings = useDeskViewingsStore();

// Monday first, as the week strip runs; 0 is Sunday.
const WEEK = [
  { n: 1, name: "Mon" },
  { n: 2, name: "Tue" },
  { n: 3, name: "Wed" },
  { n: 4, name: "Thu" },
  { n: 5, name: "Fri" },
  { n: 6, name: "Sat" },
  { n: 0, name: "Sun" },
];

const form = useForm({
  id: "hours",
  values: { days: [...props.hours.days], times: props.hours.times.join(", "), capacity: props.hours.capacity },
  order: ["days", "times", "capacity"],
  check: (v): Record<string, string> => {
    const errs: Record<string, string> = {};
    if (v.days.length === 0) errs.days = "Choose at least one day.";
    const times = v.times.split(",").map((t) => t.trim()).filter(Boolean);
    const wrong = times.find((t) => !/^([01]?\d|2[0-3]):[0-5]\d$/.test(t));
    if (times.length === 0) errs.times = "Choose at least one time.";
    else if (wrong) errs.times = `“${wrong}” is not a time; write times like 10:00, 14:45.`;
    return errs;
  },
  send: async (v) => {
    const times = v.times.split(",").map((t) => t.trim()).filter(Boolean);
    const saved = await viewings.saveHours(v.days, times, v.capacity);
    v.times = saved.hours.times.join(", ");
    desk.notice = saved.notice;
    return saved;
  },
});
</script>

<template>
  <form class="desk-panel desk-form" novalidate @submit.prevent="form.submit">
    <h2 class="desk-h2">Opening hours</h2>
    <ErrorSummary id="hours" :listed="form.listed.value" :problem="form.problem.value" @retry="form.submit" />
    <fieldset id="hours-days" class="chips">
      <legend class="visually-hidden">Open on</legend>
      <label v-for="d in WEEK" :key="d.n" class="chip"
        ><input v-model="form.values.days" type="checkbox" name="days" :value="d.n" @change="form.touched('days')" /><span>{{ d.name }}</span></label
      >
    </fieldset>
    <p v-if="form.errors.value.days" class="field__error">{{ form.errors.value.days }}</p>
    <div class="field" :class="{ 'field--error': form.errors.value.times }">
      <label class="field__label" for="hours-times">Viewing times</label>
      <textarea
        id="hours-times"
        v-model="form.values.times"
        class="textarea textarea--short mono"
        name="times"
        rows="3"
        :aria-invalid="form.errors.value.times ? 'true' : undefined"
        aria-describedby="hours-times-hint"
        @input="form.touched('times')"
      />
      <p v-if="form.errors.value.times" class="field__error">{{ form.errors.value.times }}</p>
      <p id="hours-times-hint" class="field__hint">24-hour times, separated by commas. Each visit takes {{ hours.minutes }} minutes.</p>
    </div>
    <div class="field">
      <label class="field__label" for="hours-capacity">Parties at once</label>
      <select id="hours-capacity" v-model.number="form.values.capacity" class="select" name="capacity">
        <option v-for="n in 6" :key="n" :value="n">{{ n }}</option>
      </select>
    </div>
    <button class="btn btn--small" type="submit" :disabled="form.status.value === 'sending'">Save the hours</button>
  </form>
</template>

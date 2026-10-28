<script setup lang="ts">
import { ref } from "vue";
import ErrorSummary from "@/components/ErrorSummary.vue";
import type { DeskBlock } from "@/api/desk";
import { isApiError } from "@/api/request";
import { useForm } from "@/composables/useForm";
import { dayLabel, isDay } from "@/lib/desk";
import { useDeskStore } from "@/stores/desk";
import { useDeskViewingsStore } from "@/stores/deskViewings";

/** The days the sales suite is closed for viewings, and closing another. For the manager. */
defineProps<{ blocks: DeskBlock[]; today: string }>();
const emit = defineEmits<{ changed: [] }>();
const desk = useDeskStore();
const viewings = useDeskViewingsStore();

const form = useForm({
  id: "block",
  values: { day: "", reason: "" },
  order: ["day", "reason"],
  check: (v): Record<string, string> => ({
    ...(isDay(v.day) ? {} : { day: "That date doesn’t look right." }),
    ...(v.reason.trim() ? {} : { reason: "Please give a reason." }),
    ...([...v.reason.trim()].length > 120 ? { reason: "Please keep this under 120 characters." } : {}),
  }),
  send: async (v) => {
    const done = await viewings.closeDay(v.day, v.reason.trim());
    desk.notice = done.notice;
    v.day = "";
    v.reason = "";
    emit("changed");
    return done;
  },
});

// Opening a day again is one click; a refusal is said where the list is.
const opening = ref("");
const problem = ref("");
async function open(day: string) {
  opening.value = day;
  problem.value = "";
  try {
    const done = await viewings.openDay(day);
    desk.notice = done.notice;
    emit("changed");
  } catch (e) {
    problem.value = isApiError(e) ? e.title : "The day could not be opened.";
  } finally {
    opening.value = "";
  }
}
</script>

<template>
  <section class="desk-panel" aria-labelledby="closed-days">
    <h2 id="closed-days" class="desk-h2">Closed days</h2>
    <div v-if="problem" class="notice notice--alert" role="alert">{{ problem }}</div>
    <ul v-if="blocks.length" class="desk-list" role="list">
      <li v-for="b in blocks" :key="b.day" class="desk-list__row closed-day">
        <span class="mono desk-list__id">{{ dayLabel(b.day) }}</span>
        <span class="desk-list__main">{{ b.reason }}</span>
        <button class="link-button" type="button" :disabled="opening === b.day" :aria-label="`Open ${dayLabel(b.day)}`" @click="open(b.day)">Open</button>
      </li>
    </ul>
    <form class="desk-form" novalidate @submit.prevent="form.submit">
      <ErrorSummary id="block" :listed="form.listed.value" :problem="form.problem.value" @retry="form.submit" />
      <div class="desk-form__row">
        <div class="field" :class="{ 'field--error': form.errors.value.day }">
          <label class="field__label" for="block-day">Day</label>
          <input
            id="block-day"
            v-model="form.values.day"
            class="input"
            name="day"
            type="date"
            :min="today"
            :aria-invalid="form.errors.value.day ? 'true' : undefined"
            :aria-describedby="form.errors.value.day ? 'block-day-note' : undefined"
            @input="form.touched('day')"
          />
          <p v-if="form.errors.value.day" id="block-day-note" class="field__error">{{ form.errors.value.day }}</p>
        </div>
        <div class="field closed-day__reason" :class="{ 'field--error': form.errors.value.reason }">
          <label class="field__label" for="block-reason">Reason</label>
          <input
            id="block-reason"
            v-model="form.values.reason"
            class="input"
            name="reason"
            type="text"
            maxlength="120"
            placeholder="Such as: staff training"
            :aria-invalid="form.errors.value.reason ? 'true' : undefined"
            :aria-describedby="form.errors.value.reason ? 'block-reason-note' : undefined"
            @input="form.touched('reason')"
          />
          <p v-if="form.errors.value.reason" id="block-reason-note" class="field__error">{{ form.errors.value.reason }}</p>
        </div>
      </div>
      <button class="btn btn--small" type="submit" :disabled="form.status.value === 'sending'">Close the day</button>
    </form>
  </section>
</template>

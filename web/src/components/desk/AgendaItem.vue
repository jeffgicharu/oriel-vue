<script setup lang="ts">
import ErrorSummary from "@/components/ErrorSummary.vue";
import type { DeskViewing } from "@/api/desk";
import { useForm } from "@/composables/useForm";
import { plural, viewingStatus } from "@/lib/desk";
import { useDeskStore } from "@/stores/desk";
import { useDeskViewingsStore } from "@/stores/deskViewings";

/** One viewing on the day's agenda; once its time has come, whether the visitor came. */
const props = defineProps<{ v: DeskViewing }>();
const desk = useDeskStore();
const viewings = useDeskViewingsStore();

const form = useForm({
  id: `outcome-${props.v.ref}`,
  values: { notes: "", outcome: "attended" as "attended" | "no_show" },
  order: ["notes"],
  check: (v): Record<string, string> => ([...v.notes.trim()].length > 300 ? { notes: "Please keep this under 300 characters." } : {}),
  send: async (v) => {
    const done = await viewings.recordOutcome(props.v.ref, v.outcome, v.notes.trim());
    desk.notice = done.notice;
    return done;
  },
});
</script>

<template>
  <li class="agenda__item" :class="`is-${v.status}`">
    <span class="agenda__time mono">{{ v.start.slice(11, 16) }}</span>
    <div class="agenda__main">
      <RouterLink class="agenda__name" :to="{ name: 'desk-lead', params: { id: String(v.leadId) } }">{{ v.name }}</RouterLink>
      <span class="agenda__meta"
        >{{ v.party }} {{ plural(v.party, "person", "people") }}<template v-if="v.home"> · home {{ v.home }}</template> · {{ v.phone
        }}<template v-if="v.agent"> · {{ v.agent }}</template></span
      >
      <span v-if="v.notes" class="agenda__note">“{{ v.notes }}”</span>
      <span v-if="v.outcome" class="agenda__note">{{ v.outcome }}</span>
    </div>
    <div class="agenda__state">
      <form v-if="v.status === 'booked' && v.due" class="agenda__outcome" novalidate @submit.prevent="form.submit">
        <ErrorSummary :id="`outcome-${v.ref}`" :listed="form.listed.value" :problem="form.problem.value" @retry="form.submit" />
        <label class="visually-hidden" :for="`outcome-${v.ref}-notes`">Notes</label>
        <input
          :id="`outcome-${v.ref}-notes`"
          v-model="form.values.notes"
          class="input"
          name="notes"
          type="text"
          maxlength="300"
          placeholder="Notes, if any"
          :aria-invalid="form.errors.value.notes ? 'true' : undefined"
          :aria-describedby="form.errors.value.notes ? `outcome-${v.ref}-notes-note` : undefined"
          @input="form.touched('notes')"
        />
        <button class="btn btn--small btn--cobalt" type="submit" :disabled="form.status.value === 'sending'" @click="form.values.outcome = 'attended'">Came</button>
        <button class="btn btn--small btn--line" type="submit" :disabled="form.status.value === 'sending'" @click="form.values.outcome = 'no_show'">Did not come</button>
        <p v-if="form.errors.value.notes" :id="`outcome-${v.ref}-notes-note`" class="field__error">{{ form.errors.value.notes }}</p>
      </form>
      <span v-else class="agenda__status" :class="`is-${v.status}`">{{ viewingStatus(v.status) }}</span>
    </div>
  </li>
</template>

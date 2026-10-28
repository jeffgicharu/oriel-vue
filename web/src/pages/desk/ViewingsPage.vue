<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import RequestView from "@/components/RequestView.vue";
import AgendaItem from "@/components/desk/AgendaItem.vue";
import ClosedDays from "@/components/desk/ClosedDays.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import DeskLayout from "@/components/desk/DeskLayout.vue";
import ViewingHours from "@/components/desk/ViewingHours.vue";
import type { LiveEvent } from "@/composables/useLiveEvents";
import { usePageData } from "@/composables/usePageData";
import { addDays, isDay, mondayOf, plural, weekdayShort } from "@/lib/desk";
import { longDate, parseDay } from "@/lib/format";
import { useDeskStore } from "@/stores/desk";
import { useDeskViewingsStore } from "@/stores/deskViewings";

const route = useRoute("desk-viewings");
const desk = useDeskStore();
const viewings = useDeskViewingsStore();

// The day on show, in the address; today when there is none. The strip
// shows its week, Monday to Sunday.
const day = computed(() => {
  const d = String(route.query.day ?? "");
  return isDay(d) ? d : desk.today;
});
const monday = computed(() => mondayOf(day.value));
const sunday = computed(() => addDays(monday.value, 6));

usePageData(
  async () => (route.name === "desk-viewings" ? viewings.loadAgenda(monday.value, sunday.value) : undefined),
  () => monday.value,
);

function reload(event: LiveEvent) {
  if (["lead", "lead-changed"].includes(event.type)) void viewings.loadAgenda(monday.value, sunday.value, true);
}

const blocked = computed(() => viewings.agenda.data?.days.find((d) => d.date === day.value)?.blocked ?? "");
const list = computed(() => (viewings.agenda.data?.viewings ?? []).filter((v) => v.start.startsWith(day.value)));
</script>

<template>
  <DeskLayout title="Viewings" nav="viewings" :reload="reload">
    <DeskHeader title="Viewings" lede="The sales suite’s agenda. Mark each visit once it has happened." />
    <RequestView :state="viewings.agenda" :retry="() => viewings.loadAgenda(monday, sunday, true)" loading-label="Loading the agenda…">
      <template #default="{ data, busy }">
        <div class="week-strip" :aria-busy="busy ? 'true' : undefined">
          <RouterLink class="week-strip__step" :to="{ name: 'desk-viewings', query: { day: addDays(monday, -7) } }" aria-label="The week before">←</RouterLink>
          <ol class="week-strip__days" role="list">
            <li v-for="d in data.days" :key="d.date">
              <RouterLink
                class="week-day"
                :class="{ 'is-closed': !d.open, 'is-today': d.date === desk.today }"
                :to="{ name: 'desk-viewings', query: { day: d.date } }"
                :aria-current="d.date === day ? 'page' : undefined"
              >
                <span class="week-day__name">{{ weekdayShort(d.date) }}</span>
                <span class="week-day__num">{{ parseDay(d.date).day }}</span>
                <span class="week-day__count">{{ !d.open ? "Closed" : d.viewings === 0 ? "—" : `${d.viewings} ${plural(d.viewings, "visit", "visits")}` }}</span>
              </RouterLink>
            </li>
          </ol>
          <RouterLink class="week-strip__step" :to="{ name: 'desk-viewings', query: { day: addDays(monday, 7) } }" aria-label="The week after">→</RouterLink>
        </div>
        <div class="agenda-layout">
          <section class="desk-panel" aria-labelledby="agenda-day">
            <div class="desk-panel__head">
              <h2 id="agenda-day" class="desk-h2">{{ longDate(day) }}</h2>
              <span v-if="blocked" class="stage-tag stage-tag--lost">Closed: {{ blocked }}</span>
            </div>
            <p v-if="list.length === 0" class="desk-empty">No viewings this day.</p>
            <ol v-else class="agenda" role="list">
              <AgendaItem v-for="v in list" :key="v.ref" :v="v" />
            </ol>
          </section>
          <div v-if="desk.manager" class="agenda-side">
            <ViewingHours :hours="data.hours" />
            <ClosedDays :blocks="data.blocks" :today="desk.today" @changed="viewings.loadAgenda(monday, sunday, true)" />
          </div>
        </div>
      </template>
    </RequestView>
  </DeskLayout>
</template>

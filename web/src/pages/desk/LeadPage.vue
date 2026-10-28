<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import RequestView from "@/components/RequestView.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import DeskLayout from "@/components/desk/DeskLayout.vue";
import LeadDetail from "@/components/desk/LeadDetail.vue";
import type { LiveEvent } from "@/composables/useLiveEvents";
import { useHttpStatus } from "@/composables/useHttpStatus";
import { usePageData } from "@/composables/usePageData";
import { stageTitle } from "@/lib/desk";
import { useDeskStore } from "@/stores/desk";
import { useDeskLeadsStore } from "@/stores/deskLeads";

const route = useRoute("desk-lead");
const desk = useDeskStore();
const leads = useDeskLeadsStore();
const setStatus = useHttpStatus();

const id = computed(() => route.params.id);
const state = computed(() => leads.page(id.value));

usePageData(async () => {
  if (route.name !== "desk-lead" || !id.value) return;
  await Promise.all([leads.loadLead(id.value), desk.manager ? desk.loadTeam() : Promise.resolve()]);
  const status = leads.page(id.value).error?.status;
  if (status === 403 || status === 404) setStatus(status);
}, id);

// A lead that is not there, or is another agent's, is said so on the page.
const refused = computed(() => (state.value.status === "error" ? state.value.error?.status : undefined));
const title = computed(() => (refused.value === 404 ? "No such lead" : (state.value.data?.lead.name ?? "Lead")));

function reload(event: LiveEvent) {
  if (event.type === "lead-changed" ? event.id === id.value : ["lead", "reservation"].includes(event.type)) void leads.loadLead(id.value, true);
}
</script>

<template>
  <DeskLayout :title="title" nav="leads" :reload="reload">
    <template v-if="refused === 404 || refused === 403">
      <DeskHeader
        :title="refused === 404 ? 'There is no such lead.' : 'This lead is another agent’s.'"
        :lede="refused === 404 ? 'It may have been mistyped, or it is not on this desk.' : 'Each agent sees their own leads; the sales manager sees everyone’s.'"
      />
      <p><RouterLink class="btn btn--line btn--small" :to="{ name: 'desk-leads' }">Back to the leads</RouterLink></p>
    </template>
    <RequestView v-else :state="state" :retry="() => leads.loadLead(id, true)" loading-label="Loading the lead…">
      <template #default="{ data }">
        <nav class="breadcrumb label" aria-label="Breadcrumb">
          <RouterLink :to="{ name: 'desk-leads' }">Leads</RouterLink><span aria-hidden="true">/</span><span aria-current="page">{{ data.lead.name }}</span>
        </nav>
        <DeskHeader :title="data.lead.name" :lede="data.lead.interest">
          <span class="stage-tag stage-tag--big" :class="`stage-tag--${data.lead.stage}`">{{ stageTitle(data.lead.stage) }}</span>
        </DeskHeader>
        <LeadDetail :page="data" :team="desk.team.data ?? []" />
      </template>
    </RequestView>
  </DeskLayout>
</template>

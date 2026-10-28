<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute, useRouter, type LocationQueryRaw } from "vue-router";
import RequestView from "@/components/RequestView.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import DeskLayout from "@/components/desk/DeskLayout.vue";
import LeadBoard from "@/components/desk/LeadBoard.vue";
import type { LiveEvent } from "@/composables/useLiveEvents";
import { usePageData } from "@/composables/usePageData";
import { dayLabel, dotColour, isStage, STAGES, stageTitle, when } from "@/lib/desk";
import { useDeskStore } from "@/stores/desk";
import { useDeskLeadsStore, type LeadQuery } from "@/stores/deskLeads";

const route = useRoute("desk-leads");
const router = useRouter();
const desk = useDeskStore();
const leads = useDeskLeadsStore();

/** The filters in the address. The board is the pipeline; the table takes the filters. */
const filter = computed<LeadQuery>(() => {
  const q = route.query;
  const one = (v: unknown) => (Array.isArray(v) ? String(v[0] ?? "") : String(v ?? ""));
  const stage = one(q.stage);
  const sort = one(q.sort);
  const agent = Number(one(q.agent));
  return {
    view: one(q.view) === "table" ? "table" : "board",
    stage: isStage(stage) ? stage : "",
    agent: Number.isInteger(agent) && agent > 0 ? agent : 0,
    q: one(q.q).trim().slice(0, 80),
    source: one(q.source).trim().slice(0, 40),
    sort: sort === "follow" || sort === "name" ? sort : "recent",
  };
});

// While the router moves on to another page, the address is that page's, so
// nothing is read for it here; the other pages do the same.
usePageData(
  async () => (route.name === "desk-leads" ? Promise.all([leads.loadList(filter.value), desk.manager ? desk.loadTeam() : undefined]) : undefined),
  () => JSON.stringify(filter.value),
);

const lede = computed(() => (desk.manager ? "Everyone the team is talking to, from the first enquiry to the keys." : "Your leads, from the first enquiry to the keys."));

// The spreadsheet of the same list, with the same filters.
const csv = computed(() => {
  const f = filter.value;
  const p = new URLSearchParams();
  if (f.view === "table") {
    if (f.q) p.set("q", f.q);
    if (f.stage) p.set("stage", f.stage);
  }
  if (f.agent && desk.manager) p.set("agent", String(f.agent));
  if (f.view === "table") {
    if (f.source) p.set("source", f.source);
    if (f.sort !== "recent") p.set("sort", f.sort);
  }
  const s = p.toString();
  return "/api/desk/leads.csv" + (s ? `?${s}` : "");
});

// The table's filters, as typed, until they are shown.
const draft = reactive({ q: "", stage: "", agent: "", source: "", sort: "recent" });
watch(
  filter,
  (f) => Object.assign(draft, { q: f.q, stage: f.stage, agent: f.agent ? String(f.agent) : "", source: f.source, sort: f.sort }),
  { immediate: true },
);

function show() {
  const query: LocationQueryRaw = { view: "table" };
  if (draft.q.trim()) query.q = draft.q.trim();
  if (draft.stage) query.stage = draft.stage;
  if (draft.agent) query.agent = draft.agent;
  if (draft.source) query.source = draft.source;
  if (draft.sort !== "recent") query.sort = draft.sort;
  void router.push({ query });
}

// A change made elsewhere waits until a card being dragged has landed.
const boardBusy = ref(false);
let pending = false;
function reload(event?: LiveEvent) {
  if (event && !["lead", "lead-changed", "reservation"].includes(event.type)) return;
  if (boardBusy.value) {
    pending = true;
    return;
  }
  void leads.loadList(filter.value, true);
}
function setBusy(busy: boolean) {
  boardBusy.value = busy;
  if (!busy && pending) {
    pending = false;
    reload();
  }
}
</script>

<template>
  <DeskLayout title="Leads" nav="leads" :reload="reload">
    <DeskHeader title="Leads" :lede="lede">
      <a class="btn btn--line btn--small" :href="csv">Export CSV</a>
      <RouterLink class="btn btn--cobalt btn--small" :to="{ name: 'desk-lead-new' }">Add a lead</RouterLink>
    </DeskHeader>
    <nav class="desk-tabs" aria-label="View">
      <RouterLink :to="{ name: 'desk-leads' }" :aria-current="filter.view === 'board' ? 'page' : undefined">Pipeline</RouterLink>
      <RouterLink :to="{ name: 'desk-leads', query: { view: 'table' } }" :aria-current="filter.view === 'table' ? 'page' : undefined"
        >Table<span class="tnum">{{ leads.list.data?.total ?? "" }}</span></RouterLink
      >
    </nav>
    <template v-if="filter.view === 'board'">
      <p class="desk-small">Drag a lead to move it along, or use the ⋯ button on its card. Moving a lead to Lost asks why.</p>
      <RequestView :state="leads.list" :retry="() => leads.loadList(filter, true)" loading-label="Loading the pipeline…">
        <template #default="{ data }">
          <LeadBoard v-if="data.filter.view === 'board'" :columns="data.columns" :now="desk.now" @busy="setBusy" @failed="leads.loadList(filter, true)" />
          <p v-else class="muted request-loading">Loading the pipeline…</p>
        </template>
      </RequestView>
    </template>
    <template v-else>
      <form class="desk-filters" method="get" action="/desk/leads" @submit.prevent="show">
        <input type="hidden" name="view" value="table" />
        <div class="field">
          <label class="field__label" for="f-q">Search</label>
          <input id="f-q" v-model="draft.q" class="input" name="q" type="search" placeholder="Name, email, phone or interest" />
        </div>
        <div class="field">
          <label class="field__label" for="f-stage">Stage</label>
          <select id="f-stage" v-model="draft.stage" class="select" name="stage">
            <option value="">Every stage</option>
            <option v-for="s in STAGES" :key="s" :value="s">{{ stageTitle(s) }}</option>
          </select>
        </div>
        <div v-if="desk.manager" class="field">
          <label class="field__label" for="f-agent">Agent</label>
          <select id="f-agent" v-model="draft.agent" class="select" name="agent">
            <option value="">Everyone</option>
            <option v-for="m in desk.team.data ?? []" :key="m.id" :value="String(m.id)">{{ m.name }}</option>
          </select>
        </div>
        <div class="field">
          <label class="field__label" for="f-source">Source</label>
          <select id="f-source" v-model="draft.source" class="select" name="source">
            <option value="">Every source</option>
            <option v-if="draft.source && !(leads.list.data?.sources ?? []).includes(draft.source)" :value="draft.source">{{ draft.source }}</option>
            <option v-for="s in leads.list.data?.sources ?? []" :key="s" :value="s">{{ s }}</option>
          </select>
        </div>
        <div class="field">
          <label class="field__label" for="f-sort">Order</label>
          <select id="f-sort" v-model="draft.sort" class="select" name="sort">
            <option value="recent">Latest activity</option>
            <option value="follow">Follow-up date</option>
            <option value="name">Name</option>
          </select>
        </div>
        <button class="btn btn--small" type="submit">Show</button>
      </form>
      <RequestView :state="leads.list" :retry="() => leads.loadList(filter, true)" loading-label="Loading the leads…">
        <template #default="{ data, busy }">
          <p v-if="data.filter.view !== 'table'" class="muted request-loading">Loading the leads…</p>
          <p v-else-if="data.leads.length === 0" class="desk-empty">No leads match.</p>
          <template v-else>
            <div class="desk-table-wrap" :aria-busy="busy ? 'true' : undefined">
              <table class="desk-table">
                <thead>
                  <tr>
                    <th scope="col">Name</th>
                    <th scope="col">Stage</th>
                    <th scope="col">Agent</th>
                    <th scope="col">Interest</th>
                    <th scope="col">Source</th>
                    <th scope="col">Follow up</th>
                    <th scope="col">Last activity</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="l in data.leads" :key="l.id">
                    <th scope="row">
                      <RouterLink :to="{ name: 'desk-lead', params: { id: String(l.id) } }">{{ l.name }}</RouterLink>
                    </th>
                    <td>
                      <span class="stage-tag" :class="`stage-tag--${l.stage}`">{{ stageTitle(l.stage) }}</span>
                    </td>
                    <td><span class="desk-dot" :style="{ backgroundColor: dotColour(l.agentColour) }" aria-hidden="true" /> {{ l.agent }}</td>
                    <td class="wrap-cell">{{ l.interest }}</td>
                    <td>{{ l.source }}</td>
                    <td :class="{ 'is-late': l.overdue }">{{ dayLabel(l.followUpOn) }}</td>
                    <td>{{ when(l.updated, desk.now) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-if="data.more || data.leads.length >= 300" class="desk-small">Showing the first 300. Search or filter to narrow the list.</p>
          </template>
        </template>
      </RequestView>
    </template>
  </DeskLayout>
</template>

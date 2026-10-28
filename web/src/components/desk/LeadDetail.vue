<script setup lang="ts">
import { computed, watch } from "vue";
import ErrorSummary from "@/components/ErrorSummary.vue";
import type { DeskArrival, DeskLeadPage, DeskStaff } from "@/api/desk";
import { useForm } from "@/composables/useForm";
import { buyingName, dayMonth, isDay, phoneLink, STAGES, stageTitle, viewingStatus, when, type Stage } from "@/lib/desk";
import { kes, kesShort } from "@/lib/format";
import { useDeskStore } from "@/stores/desk";
import { useDeskLeadsStore } from "@/stores/deskLeads";

/**
 * A lead's page below its heading: a note, where they came from on the
 * website, their history; and beside these their contact details, the
 * stage, the day to follow up, the agent, their viewings and reservations.
 */
const props = defineProps<{ page: DeskLeadPage; team: DeskStaff[] }>();
const desk = useDeskStore();
const leads = useDeskLeadsStore();
const lead = computed(() => props.page.lead);
const id = computed(() => String(props.page.lead.id));

function said(changed: DeskLeadPage) {
  desk.notice = changed.notice;
  return changed;
}

const note = useForm({
  id: "note",
  values: { body: "" },
  order: ["body"],
  check: (v): Record<string, string> => {
    const body = v.body.trim();
    if (!body) return { body: "The note is empty." };
    if ([...body].length > 2000) return { body: "Please keep this under 2,000 characters." };
    return {};
  },
  send: async (v) => {
    const changed = said(await leads.addNote(id.value, v.body.trim()));
    v.body = "";
    return changed;
  },
});

const stage = useForm({
  id: "stage",
  values: { stage: lead.value.stage as Stage, reason: "" },
  order: ["stage", "reason"],
  check: (v): Record<string, string> => {
    const reason = v.reason.trim();
    if (v.stage === "lost" && !reason) return { reason: "Please say why the lead was lost." };
    if ([...reason].length > 200) return { reason: "Please keep this under 200 characters." };
    return {};
  },
  send: async (v) => {
    const changed = said(await leads.setStage(id.value, v.stage, v.reason.trim()));
    v.reason = "";
    return changed;
  },
});

const follow = useForm({
  id: "follow",
  values: { day: lead.value.followUpOn, clear: false },
  order: ["day"],
  check: (v): Record<string, string> => (v.clear || v.day === "" || isDay(v.day) ? {} : { day: "That date doesn’t look right." }),
  send: async (v) => said(await leads.setFollowUp(id.value, v.clear ? "" : v.day)),
});

const assign = useForm({
  id: "assign",
  values: { agent: String(lead.value.agentId) },
  order: ["agent"],
  check: (v): Record<string, string> => (v.agent ? {} : { agent: "Please choose an agent." }),
  send: async (v) => said(await leads.assign(id.value, Number(v.agent))),
});

// The forms follow the lead when it changes, here or anywhere else.
watch(
  () => lead.value.stage,
  (s) => (stage.values.stage = s as Stage),
);
watch(
  () => lead.value.followUpOn,
  (d) => (follow.values.day = d),
);
watch(
  () => lead.value.agentId,
  (a) => (assign.values.agent = String(a)),
);

const actions: Record<string, string> = { enquiry: "Sent a question", viewing: "Booked a viewing", reservation: "Asked to reserve" };

function or(s: string): string {
  return s || "—";
}

const arrivals = (t: DeskLeadPage["touches"][number]): { label: string; a: DeskArrival }[] => [
  { label: "This visit", a: t.last },
  { label: "First visit", a: t.first },
];
</script>

<template>
  <div class="lead-page">
    <div class="lead-page__main">
      <form class="desk-panel desk-form" novalidate @submit.prevent="note.submit">
        <label class="desk-h2" for="note-body">Add a note</label>
        <ErrorSummary id="note" :listed="note.listed.value" :problem="note.problem.value" @retry="note.submit" />
        <textarea
          id="note-body"
          v-model="note.values.body"
          class="textarea"
          name="body"
          maxlength="2000"
          required
          placeholder="A call, a message, what they asked for…"
          :aria-invalid="note.errors.value.body ? 'true' : undefined"
          :aria-describedby="note.errors.value.body ? 'note-body-note' : undefined"
          @input="note.touched('body')"
        />
        <p v-if="note.errors.value.body" id="note-body-note" class="field__error">{{ note.errors.value.body }}</p>
        <button class="btn btn--small" type="submit" :disabled="note.status.value === 'sending'">Save the note</button>
      </form>

      <section class="desk-panel" aria-labelledby="lead-touches">
        <h2 id="lead-touches" class="desk-h2">Where they came from</h2>
        <p v-if="page.touches.length === 0" class="desk-empty">Nothing they sent came through the website.</p>
        <div v-else class="desk-table-wrap" role="region" aria-labelledby="lead-touches" tabindex="0">
          <table class="desk-table touches">
            <thead>
              <tr>
                <th scope="col">Visit</th>
                <th scope="col">Channel</th>
                <th scope="col">Campaign</th>
                <th scope="col">Referrer</th>
                <th scope="col">Landing page</th>
                <th scope="col">When</th>
              </tr>
            </thead>
            <tbody v-for="(t, i) in page.touches" :key="i">
              <tr class="touches__sent">
                <td colspan="6">
                  <strong>{{ actions[t.action] ?? t.action }}<template v-if="t.home"> · home {{ t.home }}</template></strong> · {{ when(t.at, desk.now) }}
                </td>
              </tr>
              <tr v-for="x in arrivals(t)" :key="x.label">
                <th scope="row">{{ x.label }}</th>
                <td>{{ or(x.a.channel) }}</td>
                <td class="wrap-cell">{{ or(x.a.campaign) }}</td>
                <td>{{ or(x.a.referrer) }}</td>
                <td class="mono">{{ or(x.a.landing) }}</td>
                <td>{{ x.a.at ? when(x.a.at, desk.now) : "—" }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="desk-panel" aria-labelledby="lead-history">
        <h2 id="lead-history" class="desk-h2">History</h2>
        <ol class="desk-timeline lead-timeline" role="list">
          <li v-for="(e, i) in page.timeline" :key="i" :class="`is-${e.kind}`">
            <span class="desk-timeline__when mono">{{ when(e.at, desk.now) }}</span>
            <span class="desk-timeline__body">
              {{ e.body }}
              <span v-if="e.staff" class="desk-timeline__sub">{{ e.staff }}</span>
            </span>
          </li>
        </ol>
      </section>
    </div>

    <aside class="lead-page__side">
      <section class="desk-panel" aria-labelledby="lead-contact">
        <h2 id="lead-contact" class="desk-h2">Contact</h2>
        <dl class="desk-facts">
          <div>
            <dt class="label">Email</dt>
            <dd><a :href="`mailto:${lead.email}`">{{ lead.email }}</a></dd>
          </div>
          <div>
            <dt class="label">Phone</dt>
            <dd><a :href="`tel:${phoneLink(lead.phone)}`">{{ lead.phone }}</a></dd>
          </div>
          <div>
            <dt class="label">Source</dt>
            <dd>{{ lead.source }}</dd>
          </div>
          <div v-if="lead.budget > 0">
            <dt class="label">Budget</dt>
            <dd class="tnum">{{ kes(lead.budget) }}</dd>
          </div>
          <div v-if="lead.buying">
            <dt class="label">Buying</dt>
            <dd>{{ buyingName(lead.buying) }}</dd>
          </div>
          <div>
            <dt class="label">First contact</dt>
            <dd>{{ when(lead.created, desk.now) }}</dd>
          </div>
          <div v-if="lead.lostReason">
            <dt class="label">Lost because</dt>
            <dd>{{ lead.lostReason }}</dd>
          </div>
        </dl>
      </section>

      <form class="desk-panel desk-form" novalidate @submit.prevent="stage.submit">
        <h2 class="desk-h2">Stage</h2>
        <ErrorSummary id="stage" :listed="stage.listed.value" :problem="stage.problem.value" @retry="stage.submit" />
        <div class="field">
          <label class="visually-hidden" for="stage-stage">Stage</label>
          <select id="stage-stage" v-model="stage.values.stage" class="select" name="stage" @change="stage.touched('stage')">
            <option v-for="s in STAGES" :key="s" :value="s">{{ stageTitle(s) }}</option>
          </select>
        </div>
        <div class="field" :class="{ 'field--error': stage.errors.value.reason }">
          <label class="field__label" for="stage-reason">Reason <span class="label">If lost</span></label>
          <input
            id="stage-reason"
            v-model="stage.values.reason"
            class="input"
            name="reason"
            type="text"
            maxlength="200"
            :aria-invalid="stage.errors.value.reason ? 'true' : undefined"
            :aria-describedby="stage.errors.value.reason ? 'stage-reason-note' : undefined"
            @input="stage.touched('reason')"
          />
          <p v-if="stage.errors.value.reason" id="stage-reason-note" class="field__error">{{ stage.errors.value.reason }}</p>
        </div>
        <button class="btn btn--small" type="submit" :disabled="stage.status.value === 'sending'">Move</button>
      </form>

      <form class="desk-panel desk-form" novalidate @submit.prevent="follow.submit">
        <h2 class="desk-h2">Follow up on</h2>
        <ErrorSummary id="follow" :listed="follow.listed.value" :problem="follow.problem.value" @retry="follow.submit" />
        <div class="desk-form__row">
          <div class="field" :class="{ 'field--error': follow.errors.value.day }">
            <label class="visually-hidden" for="follow-day">Date</label>
            <input
              id="follow-day"
              v-model="follow.values.day"
              class="input"
              name="day"
              type="date"
              :aria-invalid="follow.errors.value.day ? 'true' : undefined"
              :aria-describedby="follow.errors.value.day ? 'follow-day-note' : undefined"
              @input="follow.touched('day')"
            />
          </div>
          <button class="btn btn--small" type="submit" :disabled="follow.status.value === 'sending'" @click="follow.values.clear = false">Set</button>
          <button v-if="lead.followUpOn" class="btn btn--small btn--line" type="submit" :disabled="follow.status.value === 'sending'" @click="follow.values.clear = true">
            Clear
          </button>
        </div>
        <p v-if="follow.errors.value.day" id="follow-day-note" class="field__error">{{ follow.errors.value.day }}</p>
      </form>

      <form v-if="desk.manager" class="desk-panel desk-form" novalidate @submit.prevent="assign.submit">
        <h2 class="desk-h2">Agent</h2>
        <ErrorSummary id="assign" :listed="assign.listed.value" :problem="assign.problem.value" @retry="assign.submit" />
        <div class="desk-form__row">
          <div class="field">
            <label class="visually-hidden" for="assign-agent">Agent</label>
            <select id="assign-agent" v-model="assign.values.agent" class="select" name="agent" @change="assign.touched('agent')">
              <option v-for="m in team" :key="m.id" :value="String(m.id)">{{ m.name }}</option>
            </select>
          </div>
          <button class="btn btn--small" type="submit" :disabled="assign.status.value === 'sending'">Assign</button>
        </div>
      </form>
      <section v-else class="desk-panel">
        <h2 class="desk-h2">Agent</h2>
        <p>{{ lead.agent }}</p>
      </section>

      <section v-if="page.viewings.length" class="desk-panel" aria-labelledby="lead-viewings">
        <h2 id="lead-viewings" class="desk-h2">Viewings</h2>
        <ul class="desk-list" role="list">
          <li v-for="v in page.viewings" :key="v.ref" :class="{ 'is-cancelled': v.status === 'cancelled' }">
            <RouterLink class="desk-list__row" :to="{ name: 'desk-viewings', query: { day: v.start.slice(0, 10) } }">
              <span class="mono desk-list__id">{{ dayMonth(v.start) }}</span>
              <span class="desk-list__main">{{ v.start.slice(11, 16) }}<span class="desk-list__sub">{{ viewingStatus(v.status) }}</span></span>
            </RouterLink>
          </li>
        </ul>
      </section>

      <section v-if="page.reservations.length" class="desk-panel" aria-labelledby="lead-res">
        <h2 id="lead-res" class="desk-h2">Reservations</h2>
        <ul class="desk-list" role="list">
          <li v-for="r in page.reservations" :key="r.ref">
            <RouterLink class="desk-list__row" :to="{ name: 'desk-reservations', query: { tab: r.group }, hash: `#${r.ref}` }">
              <span class="mono desk-list__id">{{ r.home }}</span>
              <span class="desk-list__main">{{ r.statusName }}<span class="desk-list__sub">{{ r.ref }} · {{ kesShort(r.price) }}</span></span>
            </RouterLink>
          </li>
        </ul>
      </section>
    </aside>
  </div>
</template>

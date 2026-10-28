<script setup lang="ts">
import { computed } from "vue";
import RequestView from "@/components/RequestView.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import ChannelChart from "@/components/desk/ChannelChart.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import DeskLayout from "@/components/desk/DeskLayout.vue";
import HBars from "@/components/desk/HBars.vue";
import MonthBars from "@/components/desk/MonthBars.vue";
import StatusBar from "@/components/desk/StatusBar.vue";
import type { DeskDashboard } from "@/api/desk";
import { usePageData } from "@/composables/usePageData";
import { dayLabel, dotColour, firstName, lapses, partOfDay, plural, stageTitle, trend, weekdayTime } from "@/lib/desk";
import { STATUS_ORDER } from "@/lib/deskCharts";
import { kes, kesShort, longDate } from "@/lib/format";
import { useDeskStore } from "@/stores/desk";

const desk = useDeskStore();
usePageData(() => desk.loadDashboard());

const greeting = computed(() => `Good ${partOfDay(desk.now)}, ${firstName(desk.staff?.name ?? "")}.`);

/** The homes sold back months from the latest. */
function soldIn(d: DeskDashboard, back: number): number {
  return d.months[d.months.length - 1 - back]?.sold ?? 0;
}

function taps(d: DeskDashboard): number {
  return d.whatsapp.reduce((n, w) => n + w.taps, 0);
}
</script>

<template>
  <DeskLayout title="Dashboard" nav="dashboard" :reload="() => desk.loadDashboard(true)">
    <DeskHeader :title="greeting" :lede="desk.now ? longDate(desk.now) : ''">
      <RouterLink class="btn btn--line btn--small" :to="{ name: 'desk-stack' }">Stacking plan</RouterLink>
    </DeskHeader>
    <RequestView :state="desk.dashboard" :retry="() => desk.loadDashboard(true)" loading-label="Loading the dashboard…">
      <template #default="{ data }">
        <div class="desk-kpis">
          <div class="desk-kpi">
            <span class="label">Available</span>
            <span class="desk-kpi__n tnum">{{ data.counts.available ?? 0 }}</span>
            <span class="desk-kpi__note">of {{ data.total }} homes · {{ kesShort(data.stock) }} to sell</span>
          </div>
          <div class="desk-kpi">
            <span class="label">Sold and reserved</span>
            <span class="desk-kpi__n tnum">{{ kesShort(data.soldValue) }}</span>
            <span class="desk-kpi__note">{{ (data.counts.sold ?? 0) + (data.counts.reserved ?? 0) }} homes</span>
          </div>
          <div class="desk-kpi">
            <span class="label">Sold this month</span>
            <span class="desk-kpi__n tnum">{{ soldIn(data, 0) }}</span>
            <span class="desk-kpi__note">{{ trend(soldIn(data, 0), soldIn(data, 1), "month") }}</span>
          </div>
          <div class="desk-kpi">
            <span class="label">New leads, 7 days</span>
            <span class="desk-kpi__n tnum">{{ data.newLeads }}</span>
            <span class="desk-kpi__note">{{ trend(data.newLeads, data.newLeadsWas, "week") }}</span>
          </div>
        </div>

        <section class="desk-panel" aria-labelledby="dash-status">
          <div class="desk-panel__head">
            <h2 id="dash-status" class="desk-h2">Every home by status</h2>
            <RouterLink class="desk-link" :to="{ name: 'desk-stack' }">Open the stacking plan</RouterLink>
          </div>
          <StatusBar :counts="data.counts" :total="data.total" />
          <ul class="desk-legend" role="list">
            <li v-for="st in STATUS_ORDER" :key="st"><StatusBadge :status="st" /> <span class="tnum">{{ data.counts[st] ?? 0 }}</span></li>
          </ul>
        </section>

        <div class="desk-grid desk-grid--three">
          <section class="desk-panel" aria-labelledby="dash-waiting">
            <div class="desk-panel__head">
              <h2 id="dash-waiting" class="desk-h2">Waiting for a decision</h2>
              <RouterLink class="desk-link" :to="{ name: 'desk-reservations' }">All reservations</RouterLink>
            </div>
            <p v-if="data.waiting.length === 0" class="desk-empty">No reservation requests are waiting.</p>
            <ul v-else class="desk-list" role="list">
              <li v-for="r in data.waiting" :key="r.ref">
                <RouterLink class="desk-list__row" :to="{ name: 'desk-reservations', hash: `#${r.ref}` }">
                  <span class="mono desk-list__id">{{ r.home }}</span>
                  <span class="desk-list__main">{{ r.name }}<span class="desk-list__sub">{{ r.planName }} · <span class="desk-amount">{{ kes(r.price) }}</span></span></span>
                  <span class="desk-list__aside">{{ lapses(r.holdSecondsLeft) }}</span>
                </RouterLink>
              </li>
            </ul>
          </section>
          <section class="desk-panel" aria-labelledby="dash-follow">
            <div class="desk-panel__head">
              <h2 id="dash-follow" class="desk-h2">Follow-ups due</h2>
              <RouterLink class="desk-link" :to="{ name: 'desk-leads', query: { view: 'table', sort: 'follow' } }">All leads</RouterLink>
            </div>
            <p v-if="data.followUps.length === 0" class="desk-empty">Nothing due today.</p>
            <ul v-else class="desk-list" role="list">
              <li v-for="l in data.followUps" :key="l.id">
                <RouterLink class="desk-list__row" :to="{ name: 'desk-lead', params: { id: String(l.id) } }">
                  <span class="desk-dot" :style="{ backgroundColor: dotColour(l.agentColour) }" :title="l.agent" aria-hidden="true" />
                  <span class="desk-list__main">{{ l.name }}<span class="desk-list__sub">{{ stageTitle(l.stage) }} · {{ l.interest }}</span></span>
                  <span class="desk-list__aside" :class="{ 'is-late': l.followUpOn < desk.today }">{{ dayLabel(l.followUpOn) }}</span>
                </RouterLink>
              </li>
            </ul>
          </section>
          <section class="desk-panel" aria-labelledby="dash-week">
            <div class="desk-panel__head">
              <h2 id="dash-week" class="desk-h2">Viewings this week</h2>
              <RouterLink class="desk-link" :to="{ name: 'desk-viewings' }">Agenda</RouterLink>
            </div>
            <p v-if="data.week.length === 0" class="desk-empty">No viewings booked in the next seven days.</p>
            <ul v-else class="desk-list" role="list">
              <li v-for="v in data.week" :key="v.ref" :class="{ 'is-cancelled': v.status === 'cancelled' }">
                <RouterLink class="desk-list__row" :to="{ name: 'desk-lead', params: { id: String(v.leadId) } }">
                  <span class="mono desk-list__id">{{ weekdayTime(v.start) }}</span>
                  <span class="desk-list__main">
                    {{ v.name }}
                    <span class="desk-list__sub"
                      >{{ v.party }} {{ plural(v.party, "person", "people") }}<template v-if="v.home"> · home {{ v.home }}</template
                      ><template v-if="v.status === 'cancelled'"> · cancelled</template></span
                    >
                  </span>
                  <span class="desk-list__aside">{{ v.agent }}</span>
                </RouterLink>
              </li>
            </ul>
          </section>
        </div>

        <div class="desk-grid">
          <section class="desk-panel" aria-labelledby="dash-sales">
            <div class="desk-panel__head">
              <h2 id="dash-sales" class="desk-h2">Homes sold each month</h2>
            </div>
            <MonthBars :months="data.months" />
          </section>
          <section class="desk-panel" aria-labelledby="dash-funnel">
            <div class="desk-panel__head">
              <h2 id="dash-funnel" class="desk-h2">How far leads get</h2>
              <RouterLink class="desk-link" :to="{ name: 'desk-leads' }">Pipeline</RouterLink>
            </div>
            <ol class="funnel" role="list">
              <li v-for="f in data.funnel" :key="f.stage" class="funnel__row" :class="{ 'is-lost': f.lost }">
                <span class="funnel__label">{{ f.label }}</span>
                <span class="funnel__track"><span class="funnel__bar" :style="{ width: `${Math.max(f.percent, 1)}%` }" /></span>
                <span class="funnel__n tnum">{{ f.reached }}</span>
                <span class="funnel__pct tnum">{{ f.percent }}%</span>
              </li>
            </ol>
            <p class="desk-small">The share of every lead that reached each stage. A lead counts in every stage it has passed.</p>
          </section>
        </div>

        <section class="desk-panel" aria-labelledby="dash-channels">
          <div class="desk-panel__head">
            <h2 id="dash-channels" class="desk-h2">Where leads come from</h2>
            <span class="desk-small">The website’s leads, by how they first arrived</span>
          </div>
          <ChannelChart :rows="data.channels" />
          <p class="desk-small">
            Viewed counts the leads who booked a viewing and kept it; reserved, those who asked to reserve a home. Leads the team added by hand are left out.
          </p>
        </section>

        <div class="desk-grid">
          <section class="desk-panel" aria-labelledby="dash-sources">
            <div class="desk-panel__head">
              <h2 id="dash-sources" class="desk-h2">Every lead by source</h2>
              <RouterLink class="desk-link" :to="{ name: 'desk-leads', query: { view: 'table' } }">Table</RouterLink>
            </div>
            <p v-if="data.sources.length === 0" class="desk-empty">No leads yet.</p>
            <HBars v-else :items="data.sources.map((s) => ({ label: s.source, n: s.leads }))" unit="leads" title="Every lead by source" />
          </section>
          <section class="desk-panel" aria-labelledby="dash-whatsapp">
            <div class="desk-panel__head">
              <h2 id="dash-whatsapp" class="desk-h2">Taps on WhatsApp</h2>
              <span class="desk-small">{{ taps(data) }} in the last 30 days, by how each visitor arrived</span>
            </div>
            <p v-if="taps(data) === 0" class="desk-empty">Nobody has opened WhatsApp from the website in the last 30 days.</p>
            <template v-else>
              <HBars :items="data.whatsapp.map((w) => ({ label: w.channel, n: w.taps }))" unit="taps" title="Taps on WhatsApp in the last 30 days, by channel" />
              <details class="chart-figures">
                <summary class="desk-link">See the figures</summary>
                <table class="desk-table">
                  <thead>
                    <tr><th scope="col">Channel</th><th scope="col" class="num">Taps</th></tr>
                  </thead>
                  <tbody>
                    <tr v-for="w in data.whatsapp" :key="w.channel">
                      <th scope="row">{{ w.channel }}</th>
                      <td class="num">{{ w.taps }}</td>
                    </tr>
                  </tbody>
                </table>
              </details>
            </template>
          </section>
        </div>
      </template>
    </RequestView>
  </DeskLayout>
</template>

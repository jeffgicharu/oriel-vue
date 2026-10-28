<script setup lang="ts">
import { computed, onBeforeUnmount } from "vue";
import ErrorSummary from "@/components/ErrorSummary.vue";
import RequestView from "@/components/RequestView.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import DeskLayout from "@/components/desk/DeskLayout.vue";
import PriceListForm from "@/components/desk/PriceListForm.vue";
import { isApiError } from "@/api/request";
import type { LiveEvent } from "@/composables/useLiveEvents";
import { useForm } from "@/composables/useForm";
import { usePageData } from "@/composables/usePageData";
import { changePercent, plural, signedKes, when } from "@/lib/desk";
import { kes, kesShort, percent } from "@/lib/format";
import { useDeskStore } from "@/stores/desk";
import { useDeskHomesStore } from "@/stores/deskHomes";
import { useSiteStore } from "@/stores/site";

const desk = useDeskStore();
const homes = useDeskHomesStore();
const site = useSiteStore();

// Only the manager's page is read; anyone else is told it is the manager's.
usePageData(() => (desk.manager ? Promise.all([homes.loadPricing(), site.loadOverview()]) : Promise.resolve()));

function reload(event: LiveEvent) {
  if (desk.manager && event.type === "prices") void homes.loadPricing(true);
}

const CODES = ["A", "B", "C", "D", "P"] as const;
const types = computed(() => CODES.map((code) => ({ code, name: site.overview.data?.types.find((t) => t.code === code)?.name ?? "" })));

const publish = useForm({
  id: "publish",
  values: {},
  order: [],
  check: () => ({}),
  send: async () => {
    const draft = homes.preview;
    if (!draft) return null;
    try {
      const published = await homes.publishPrices(draft.list, draft.note);
      desk.notice = published.notice;
      window.scrollTo({ top: 0 });
      return published;
    } catch (e) {
      // A refusal about one price is said here, where the button is.
      if (isApiError(e) && Object.keys(e.fields).length > 0) throw { ...e, title: Object.values(e.fields)[0] ?? e.title, fields: {} };
      throw e;
    }
  },
});

// A draft is only for the visit it was drawn in.
onBeforeUnmount(() => (homes.preview = null));

function publishIt() {
  if (window.confirm("Publish this price list? The website and the price list follow at once.")) void publish.submit();
}
</script>

<template>
  <DeskLayout title="Pricing" nav="pricing" :reload="reload">
    <DeskHeader
      title="Pricing"
      lede="Every unsold home’s price comes from the list in force: a base price for each type, raised for each floor above level 2 and for a river view."
    />
    <RequestView :state="homes.pricing" :retry="() => homes.loadPricing(true)" loading-label="Loading the price lists…">
      <template #default="{ data }">
        <div class="pricing">
          <PriceListForm :key="data.current.id" :current="data.current" :types="types" :now="desk.now" />
          <section class="desk-panel" aria-labelledby="price-history">
            <h2 id="price-history" class="desk-h2">Price lists</h2>
            <ol class="desk-list" role="list">
              <li v-for="h in data.history" :key="h.id" class="desk-list__row price-history__row">
                <span class="mono desk-list__id">{{ h.id }}</span>
                <span class="desk-list__main">
                  {{ h.note || `Price list ${h.id}` }}
                  <span class="desk-list__sub">Floor {{ percent(h.floorBP) }} · river {{ percent(h.riverBP) }} · type B {{ kesShort(h.base.B) }}</span>
                </span>
                <span class="desk-list__aside">{{ when(h.published, desk.now) }}<br />{{ h.by }}</span>
              </li>
            </ol>
          </section>
        </div>
        <section v-if="homes.preview" id="preview" class="desk-panel price-preview" aria-labelledby="preview-title">
          <div class="desk-panel__head">
            <h2 id="preview-title" class="desk-h2" tabindex="-1">What this list would do</h2>
          </div>
          <div class="desk-kpis price-preview__kpis">
            <div class="desk-kpi">
              <span class="label">Homes repriced</span>
              <span class="desk-kpi__n tnum">{{ homes.preview.changed }}</span>
              <span class="desk-kpi__note">{{ homes.preview.keptOwn }} {{ plural(homes.preview.keptOwn, "keeps its", "keep their") }} own price</span>
            </div>
            <div class="desk-kpi">
              <span class="label">Stock value now</span>
              <span class="desk-kpi__n tnum">{{ kesShort(homes.preview.oldTotal) }}</span>
              <span class="desk-kpi__note">every unsold home</span>
            </div>
            <div class="desk-kpi">
              <span class="label">After</span>
              <span class="desk-kpi__n tnum">{{ kesShort(homes.preview.newTotal) }}</span>
              <span class="desk-kpi__note">at the new prices</span>
            </div>
            <div class="desk-kpi">
              <span class="label">Change</span>
              <span
                class="desk-kpi__n tnum"
                :class="{ 'is-up': homes.preview.newTotal > homes.preview.oldTotal, 'is-down': homes.preview.newTotal < homes.preview.oldTotal }"
                >{{ signedKes(homes.preview.newTotal - homes.preview.oldTotal) }}</span
              >
              <span class="desk-kpi__note">{{ changePercent(homes.preview.oldTotal, homes.preview.newTotal) }}</span>
            </div>
          </div>
          <form class="price-preview__publish" novalidate @submit.prevent="publishIt">
            <ErrorSummary id="publish" :listed="publish.listed.value" :problem="publish.problem.value" @retry="publishIt" />
            <button class="btn btn--cobalt btn--small" type="submit" :disabled="publish.status.value === 'sending'">Publish the new prices</button>
            <span class="desk-small">The website, every home’s schedule and the price-list PDF change at once.</span>
          </form>
          <div class="desk-table-wrap" role="region" aria-label="Every home’s price now and under this list" tabindex="0">
            <table class="desk-table">
              <thead>
                <tr>
                  <th scope="col">Home</th>
                  <th scope="col">Type</th>
                  <th scope="col">Status</th>
                  <th scope="col" class="num">Now</th>
                  <th scope="col" class="num">New</th>
                  <th scope="col" class="num">Change</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in homes.preview.rows" :key="row.home">
                  <th scope="row" class="mono">{{ row.home }}</th>
                  <td>{{ row.type }}</td>
                  <td><StatusBadge :status="row.status" /></td>
                  <td class="num">{{ kes(row.old) }}</td>
                  <td class="num">
                    <span v-if="row.keepsOwn" class="desk-small">keeps its own price</span>
                    <template v-else>{{ kes(row.new) }}</template>
                  </td>
                  <td class="num" :class="{ 'is-up': row.change > 0, 'is-down': row.change < 0 }">{{ signedKes(row.change) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </template>
    </RequestView>
  </DeskLayout>
</template>

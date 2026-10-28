<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import ArrowIcon from "@/components/ArrowIcon.vue";
import FloorPlan from "@/components/FloorPlan.vue";
import KeyPlan from "@/components/KeyPlan.vue";
import LightChart from "@/components/LightChart.vue";
import MiniElevation from "@/components/MiniElevation.vue";
import PaymentPlans from "@/components/PaymentPlans.vue";
import PhotoGallery from "@/components/PhotoGallery.vue";
import PlanFigure from "@/components/PlanFigure.vue";
import RequestView from "@/components/RequestView.vue";
import ShortlistButton from "@/components/ShortlistButton.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import WhatsAppLink from "@/components/WhatsAppLink.vue";
import NotFoundPage from "@/pages/NotFoundPage.vue";
import { useHttpStatus } from "@/composables/useHttpStatus";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { kes, kesShort, levelName, ordinal } from "@/lib/format";
import { homeTransition, showsPrice, statusName } from "@/lib/homes";
import { lightSummary, morningOrEvening } from "@/lib/light";
import { shotsFor } from "@/lib/photos";
import { apartmentLd, breadcrumbLd } from "@/lib/seo";
import { useFrameStore } from "@/stores/frame";
import { useHomesStore } from "@/stores/homes";
import { useShortlistStore } from "@/stores/shortlist";

const route = useRoute("home-detail");
const homes = useHomesStore();
const frame = useFrameStore();
const shortlist = useShortlistStore();
const setStatus = useHttpStatus();

const id = computed(() => route.params.id);
const state = computed(() => homes.detail(id.value));
const d = computed(() => state.value.data);
const missing = computed(() => state.value.error?.status === 404);

usePageData(async () => {
  await homes.loadHome(id.value);
  if (homes.detail(id.value).error?.status === 404) setStatus(404);
}, id);

const furniture = ref(true);
const onSale = computed(() => d.value?.home.status === "available");
const compareHref = computed(() => ({ name: "compare" as const, query: { homes: shortlist.compareWith(id.value).join(",") } }));

const bedrooms = computed(() => {
  if (!d.value) return "";
  const t = d.value.type;
  return `${t.bedrooms}${t.study ? " and a study" : ""}${t.dsq ? ", with DSQ" : ""}`;
});

usePageMeta(() => {
  const h = d.value?.home;
  if (!h) {
    return missing.value
      ? { title: "Page not found", description: "This page could not be found at Oriel, Riverside.", path: route.path, sheet: { number: "A-404", name: "Not found" } }
      : { title: `Home ${id.value}`, description: `Home ${id.value} at Oriel, Riverside: plan, light study and payment schedule.`, path: route.path, nav: "residences" };
  }
  const desc =
    `${h.title} at Oriel, Riverside, facing ${h.facings}. ${h.internal} m² inside with a ${h.outdoor} m² ${h.outdoorName.toLowerCase()}. ` +
    (h.price !== null ? `${kes(h.price)}. ` : `${statusName(h.status)}. `) +
    "Plan, light study and payment schedule.";
  return {
    title: `Home ${h.id}, ${h.typeName}`,
    description: desc,
    path: `/residences/${h.id}`,
    sheet: { number: `A-2·${h.id}`, name: `Home ${h.id}` },
    nav: "residences",
    bodyClass: onSale.value ? "has-unit-bar" : "",
    jsonLd: [
      apartmentLd(frame.siteUrl, h, d.value!.type.summary),
      breadcrumbLd(frame.siteUrl, [
        { name: "Residences", path: "/residences" },
        { name: `Type ${h.type}`, path: `/residences/types/${h.type}` },
        { name: `Home ${h.id}`, path: `/residences/${h.id}` },
      ]),
    ],
  };
});
</script>

<template>
  <NotFoundPage v-if="missing" />
  <RequestView v-else :state="state" :retry="() => homes.loadHome(id, true)" loading-label="Loading the home…">
    <template #default="{ data }">
      <article class="unit">
        <div class="wrap">
          <nav class="breadcrumb label" aria-label="Breadcrumb">
            <RouterLink :to="{ name: 'residences' }">Residences</RouterLink><span aria-hidden="true">/</span>
            <RouterLink :to="{ name: 'home-type', params: { code: data.home.type } }">Type {{ data.home.type }}</RouterLink><span aria-hidden="true">/</span>
            <span aria-current="page">{{ data.home.id }}</span>
          </nav>
          <header class="unit-head">
            <div class="unit-head__id">
              <p class="unit-head__number mono" aria-hidden="true" :style="homeTransition(data.home.id)">{{ data.home.id }}</p>
              <div class="unit-head__titles">
                <p class="label label--cobalt">Home {{ data.home.id }} · Type {{ data.home.type }}</p>
                <h1 class="h2">{{ data.home.title }}, facing {{ data.home.facings }}</h1>
              </div>
            </div>
            <div class="unit-head__deal">
              <StatusBadge :status="data.home.status" />
              <p v-if="data.home.price !== null" class="unit-head__price"><span class="label">Price</span><span class="tnum">{{ kes(data.home.price) }}</span></p>
              <p v-else class="unit-head__price is-gone"><span class="label">Price</span><span>{{ statusName(data.home.status) }}</span></p>
              <div class="unit-head__actions">
                <RouterLink v-if="onSale" class="btn btn--cobalt" :to="{ name: 'reserve', params: { id: data.home.id }, query: route.query.plan ? { plan: route.query.plan } : {} }"
                  >Request a reservation <ArrowIcon
                /></RouterLink>
                <RouterLink class="btn" :class="onSale ? 'btn--line' : 'btn--cobalt'" :to="{ name: 'viewing-new', query: { home: data.home.id } }">Book a viewing</RouterLink>
                <RouterLink class="btn btn--line" :to="compareHref">Compare</RouterLink>
                <ShortlistButton :id="data.home.id" />
              </div>
              <p v-if="data.home.status === 'held'" class="unit-head__note">
                A reservation on this home is waiting to be confirmed. You can still book a viewing, and we will tell you if it comes free.
              </p>
              <div class="unit-head__ask">
                <RouterLink class="link-arrow" :to="{ name: 'enquire', query: { home: data.home.id } }">Ask about this home <ArrowIcon /></RouterLink>
                <WhatsAppLink :home="data.home" label="Ask on WhatsApp" plain />
              </div>
            </div>
          </header>
          <dl class="unit-facts">
            <div><dt class="label">Inside</dt><dd class="tnum">{{ data.home.internal }} m²</dd></div>
            <div><dt class="label">{{ data.home.outdoorName }}</dt><dd class="tnum">{{ data.home.outdoor }} m²</dd></div>
            <div><dt class="label">Bedrooms</dt><dd>{{ bedrooms }}</dd></div>
            <div><dt class="label">Bathrooms</dt><dd>{{ data.home.bathrooms }}</dd></div>
            <div><dt class="label">Level</dt><dd>{{ ordinal(data.home.level) }} of 16</dd></div>
            <div><dt class="label">Faces</dt><dd>{{ data.home.aspect }}</dd></div>
            <div><dt class="label">Parking</dt><dd>{{ data.type.parking }} {{ data.type.parking === 1 ? "bay" : "bays" }}</dd></div>
          </dl>
        </div>

        <section class="unit-plan section--tight" aria-labelledby="plan-title">
          <div class="wrap unit-plan__grid">
            <div class="unit-plan__drawing">
              <div class="unit-plan__bar">
                <h2 id="plan-title" class="label">A-2·{{ data.home.id }} · Floor plan</h2>
                <label class="switch">
                  <input v-model="furniture" type="checkbox" />
                  <span>Furniture</span>
                </label>
              </div>
              <PlanFigure :furniture="furniture">
                <p class="plan-hint label">Slide sideways to see the whole plan</p>
                <div class="plan-scroll" tabindex="0" role="region" aria-label="Floor plan; on a narrow screen it scrolls sideways">
                  <FloorPlan :plan="data.plan" :north="data.north" :title="`Floor plan of home ${data.home.id}, ${data.home.typeName}`" />
                </div>
                <figcaption class="label muted">
                  <template v-if="data.handed">Drawn as built: the mirror image of the type {{ data.home.type }} plan. </template>Not to scale. Areas are internal and approximate.
                </figcaption>
              </PlanFigure>
            </div>
            <aside class="unit-plan__side" aria-label="Where it is">
              <div class="side-card">
                <h2 class="label">Key plan · {{ levelName(data.home.level) }}</h2>
                <KeyPlan :floor="data.floor" :number="data.home.number" :home-id="data.home.id" />
                <p class="muted">
                  Home {{ data.home.id }} is position {{ data.home.number }} on the {{ ordinal(data.home.level) }} floor{{ data.home.riverView ? ", on the river side" : ", on the garden side" }}.
                </p>
              </div>
              <div class="side-card side-card--elev">
                <h2 class="label">In the building</h2>
                <MiniElevation :level="data.home.level" />
              </div>
              <RouterLink class="link-arrow" :to="{ name: 'explore', query: { home: data.home.id } }">See it in the 3D building <ArrowIcon /></RouterLink>
            </aside>
          </div>
        </section>

        <section v-if="shotsFor(data.home.type).length" class="unit-inside section--tight section--rule" aria-labelledby="inside-title">
          <div class="wrap">
            <p class="sheet-tag label">Inside</p>
            <h2 id="inside-title" class="h3 gallery-title">How a type {{ data.home.type }} home looks finished</h2>
            <PhotoGallery :shots="shotsFor(data.home.type)" note="A typical home of this type, finished to the specification; furniture is not included." />
          </div>
        </section>

        <section class="unit-light section--tight section--rule" aria-labelledby="light-title">
          <div class="wrap unit-light__grid">
            <div>
              <p class="sheet-tag label">Light study</p>
              <h2 id="light-title" class="h3">{{ morningOrEvening(data.light) }}</h2>
              <p class="muted">
                {{ lightSummary(data.light) }} Worked out from the sun’s path over Riverside, the slab edge over each window and the fins on the east and west faces.
              </p>
            </div>
            <LightChart :light="data.light" />
          </div>
        </section>

        <section class="unit-pay section--tight section--rule" aria-labelledby="pay-title">
          <div class="wrap">
            <div class="section-head section-head--split">
              <div>
                <p class="sheet-tag label">Paying for it</p>
                <h2 id="pay-title" class="h2">Choose how you pay.</h2>
              </div>
              <p class="muted">Dates run from a reservation made today. Your letter of offer confirms the exact figures.</p>
            </div>
            <PaymentPlans v-if="showsPrice(data.home) && data.schedules.length" :home-id="data.home.id" :schedules="data.schedules" />
            <p v-else class="notice">
              This home is {{ statusName(data.home.status).toLowerCase() }}. The same plans apply to every home still available:
              <RouterLink :to="{ name: 'residences', query: { beds: String(data.home.bedrooms) } }">see similar homes</RouterLink>.
            </p>
          </div>
        </section>

        <section class="unit-more section--tight section--rule" aria-labelledby="more-title">
          <div class="wrap unit-more__grid">
            <div>
              <p class="sheet-tag label">Type {{ data.home.type }}</p>
              <h2 id="more-title" class="h3">{{ data.type.name }}</h2>
              <p class="muted">{{ data.type.summary }}</p>
              <ul class="ticks" role="list">
                <li v-for="f in data.type.features" :key="f">{{ f }}</li>
              </ul>
              <RouterLink class="link-arrow" :to="{ name: 'building', hash: '#specification' }">Finishes and specification <ArrowIcon /></RouterLink>
            </div>
            <div v-if="data.similar.length">
              <p class="sheet-tag label">Also available</p>
              <ul class="similar" role="list">
                <li v-for="s in data.similar" :key="s.id">
                  <RouterLink class="similar__card" :to="{ name: 'home-detail', params: { id: s.id } }">
                    <span class="mono similar__id">{{ s.id }}</span>
                    <span>{{ s.title }}</span>
                    <span class="muted">Faces {{ s.facings }}</span>
                    <span v-if="s.price !== null" class="tnum similar__price">{{ kesShort(s.price) }}</span>
                  </RouterLink>
                </li>
              </ul>
            </div>
          </div>
        </section>

        <div v-if="onSale && data.home.price !== null" class="unit-bar">
          <div>
            <span class="mono">{{ data.home.id }}</span>
            <strong class="tnum">{{ kesShort(data.home.price) }}</strong>
          </div>
          <RouterLink class="btn btn--cobalt btn--small" :to="{ name: 'reserve', params: { id: data.home.id } }">Reserve</RouterLink>
        </div>
      </article>
    </template>
  </RequestView>
</template>

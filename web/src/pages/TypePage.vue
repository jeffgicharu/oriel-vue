<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import FloorPlan from "@/components/FloorPlan.vue";
import PhotoGallery from "@/components/PhotoGallery.vue";
import PlanFigure from "@/components/PlanFigure.vue";
import RequestView from "@/components/RequestView.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { kesShort, levelName, oneDecimal } from "@/lib/format";
import { capitalise, showsPrice } from "@/lib/homes";
import { shotsFor } from "@/lib/photos";
import { breadcrumbLd } from "@/lib/seo";
import { useFrameStore } from "@/stores/frame";
import { useHomesStore } from "@/stores/homes";

const route = useRoute("home-type");
const homes = useHomesStore();
const frame = useFrameStore();
const code = computed(() => route.params.code.toUpperCase());
const state = computed(() => homes.typeOf(code.value));
const furniture = ref(true);

usePageData(() => homes.loadType(code.value), code);

usePageMeta(() => {
  const t = state.value.data?.type;
  return {
    title: t ? `Type ${t.code}: ${t.name}` : `Type ${code.value}`,
    description: t ? `${t.summary} ${t.internal} m² inside, ${t.total} homes at Oriel, Riverside.` : `Type ${code.value} homes at Oriel, Riverside.`,
    path: `/residences/types/${code.value}`,
    sheet: { number: `A-21${"ABCDP".indexOf(code.value)}`, name: `Type ${code.value}` },
    nav: "residences",
    jsonLd: [
      breadcrumbLd(frame.siteUrl, [
        { name: "Residences", path: "/residences" },
        { name: `Type ${code.value}`, path: `/residences/types/${code.value}` },
      ]),
    ],
  };
});
</script>

<template>
  <div class="wrap">
    <RequestView :state="state" :retry="() => homes.loadType(code, true)" loading-label="Loading the type…">
      <template #default="{ data }">
        <nav class="breadcrumb label" aria-label="Breadcrumb">
          <RouterLink :to="{ name: 'residences' }">Residences</RouterLink><span aria-hidden="true">/</span>
          <span aria-current="page">Type {{ data.type.code }}</span>
        </nav>
        <header class="page-head page-head--split">
          <div>
            <p class="sheet-tag label">Type {{ data.type.code }} · {{ data.type.total }} homes</p>
            <h1 class="h1">{{ data.type.name }}.</h1>
          </div>
          <div class="page-head__aside">
            <p class="lede">{{ data.type.summary }}</p>
            <dl class="mini-facts">
              <div><dt class="label">Inside</dt><dd class="tnum">{{ data.type.internal }} m²</dd></div>
              <div><dt class="label">{{ data.type.outdoorName }}</dt><dd class="tnum">{{ data.type.outdoor }} m²</dd></div>
              <div><dt class="label">Available</dt><dd class="tnum">{{ data.type.available }} of {{ data.type.total }}</dd></div>
              <div>
                <dt class="label">From</dt>
                <dd class="tnum">
                  <template v-if="data.type.available > 0 && data.type.from !== null">{{ kesShort(data.type.from) }}</template>
                  <template v-else-if="data.type.withheld > 0">Next release</template>
                  <template v-else>Sold out</template>
                </dd>
              </div>
            </dl>
          </div>
        </header>
        <section class="type-plan section--tight" aria-label="Plan">
          <div class="unit-plan__bar">
            <h2 class="label">Floor plan</h2>
            <label class="switch"><input v-model="furniture" type="checkbox" /><span>Furniture</span></label>
          </div>
          <PlanFigure :furniture="furniture">
            <p class="plan-hint label">Slide sideways to see the whole plan</p>
            <div class="plan-scroll">
              <FloorPlan :plan="data.plan" :title="`Floor plan of type ${data.type.code}, ${data.type.name}`" />
            </div>
            <figcaption class="label muted">The plan as drawn for the homes on the left of each pair; their neighbours are its mirror image. Not to scale.</figcaption>
          </PlanFigure>
          <ul class="ticks ticks--wide" role="list">
            <li v-for="f in data.type.features" :key="f">{{ f }}</li>
          </ul>
        </section>
        <section v-if="shotsFor(data.type.code).length" class="section--tight section--rule" aria-labelledby="inside-title">
          <h2 id="inside-title" class="h3 gallery-title">Inside a type {{ data.type.code }} home</h2>
          <PhotoGallery :shots="shotsFor(data.type.code)" note="Pictures show a typical home of this type, finished to the specification; furniture is not included." />
        </section>
        <section class="section--tight section--rule" aria-labelledby="homes-title">
          <h2 id="homes-title" class="h3">Every type {{ data.type.code }} home</h2>
          <table class="schedule schedule--compact">
            <thead>
              <tr>
                <th scope="col">Home</th>
                <th scope="col" class="num">Level</th>
                <th scope="col">Faces</th>
                <th scope="col" class="num">Sun a day</th>
                <th scope="col" class="num">Price</th>
                <th scope="col">Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="h in data.homes" :key="h.id" class="schedule__row" :class="{ 'is-off': !showsPrice(h) }">
                <th scope="row" class="schedule__home">
                  <RouterLink :to="{ name: 'home-detail', params: { id: h.id } }"><span class="mono">{{ h.id }}</span></RouterLink>
                </th>
                <td class="num mono" data-label="Level">{{ levelName(h.level) }}</td>
                <td data-label="Faces">{{ capitalise(h.facings) }}</td>
                <td class="num mono" data-label="Sun a day">{{ oneDecimal(h.sun) }} h</td>
                <td class="num" data-label="Price">
                  <span v-if="h.price !== null" class="tnum schedule__price">{{ kesShort(h.price) }}</span>
                  <span v-else class="muted">—</span>
                </td>
                <td data-label="Status"><StatusBadge :status="h.status" /></td>
              </tr>
            </tbody>
          </table>
        </section>
      </template>
    </RequestView>
  </div>
</template>

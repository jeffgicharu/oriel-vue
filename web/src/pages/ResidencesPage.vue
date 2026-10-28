<script setup lang="ts">
import { computed } from "vue";
import FloorPlan from "@/components/FloorPlan.vue";
import RequestView from "@/components/RequestView.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import { useHomeFilters } from "@/composables/useHomeFilters";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { priceSteps, type Floors, type Side, type Sort } from "@/lib/filters";
import { kesShort, levelName, oneDecimal } from "@/lib/format";
import { capitalise, homeTransition, showsPrice } from "@/lib/homes";
import { useHomesStore } from "@/stores/homes";

const homes = useHomesStore();
const { filter, narrowed, update, reset } = useHomeFilters();

usePageMeta({
  title: "Residences",
  description: "Every home at Oriel, Riverside: one, two and three bedrooms and two penthouses, with live availability, prices, plans and the hours of sun each one gets.",
  path: "/residences",
  sheet: { number: "A-200", name: "Residences" },
  nav: "residences",
});

usePageData(() => Promise.all([homes.loadList(filter.value), homes.loadPlans()]), () => filter.value);

const list = computed(() => homes.list.data);

function choose(event: Event) {
  const el = event.target as HTMLInputElement | HTMLSelectElement;
  switch (el.name) {
    case "beds":
      return update({ beds: Number(el.value) || 0 });
    case "max":
      return update({ max: Number(el.value) || 0 });
    case "floors":
      return update({ floors: el.value as Floors });
    case "side":
      return update({ side: el.value as Side });
    case "sort":
      return update({ sort: el.value as Sort });
    case "show":
      return update({ all: (el as HTMLInputElement).checked });
  }
}
</script>

<template>
  <div class="wrap">
    <header class="page-head">
      <p class="sheet-tag label">A-200 · Residences</p>
      <h1 class="h1">Find your home.</h1>
      <p v-if="list" class="lede">
        {{ list.available }} of {{ list.total }} homes are available<template v-if="list.from > 0">, from <span class="tnum">{{ kesShort(list.from) }}</span></template
        >. Every one has its own plan, aspect and light.
      </p>
    </header>
    <section v-if="list" class="types" aria-label="Home types">
      <ol class="types__list" role="list">
        <li v-for="t in list.types" :key="t.code">
          <RouterLink class="type-card" :to="{ name: 'home-type', params: { code: t.code } }">
            <span class="type-card__plan" aria-hidden="true">
              <FloorPlan v-if="homes.plans.data" :plan="homes.plans.data[t.code]" :id="`tp-${t.code}`" plain />
            </span>
            <span class="type-card__code label">Type {{ t.code }}</span>
            <span class="type-card__name">{{ t.name }}</span>
            <span class="type-card__facts mono"
              ><span>{{ t.internal }}&nbsp;m² inside</span><span>{{ t.outdoorName }} {{ t.outdoor }}&nbsp;m²</span></span
            >
            <span class="type-card__price">
              <template v-if="t.available > 0 && t.from !== null">From <strong class="tnum">{{ kesShort(t.from) }}</strong></template>
              <strong v-else-if="t.withheld > 0">Next release soon</strong>
              <strong v-else>Sold out</strong>
            </span>
            <span class="type-card__left label">{{ t.available }} of {{ t.total }} left</span>
          </RouterLink>
        </li>
      </ol>
    </section>
  </div>
  <section class="finder" aria-label="Find a home">
    <form class="filters" method="get" action="/residences" @change="choose" @submit.prevent>
      <div class="wrap filters__inner">
        <fieldset class="filter">
          <legend class="label">Bedrooms</legend>
          <div class="chips">
            <label class="chip"><input type="radio" name="beds" value="" :checked="filter.beds === 0" /><span>Any</span></label>
            <label v-for="n in 4" :key="n" class="chip"
              ><input type="radio" name="beds" :value="n" :checked="filter.beds === n" /><span>{{ n }}</span></label
            >
          </div>
        </fieldset>
        <label class="filter">
          <span class="label">Up to</span>
          <select class="select select--small" name="max">
            <option value="" :selected="filter.max === 0">Any price</option>
            <option v-for="step in priceSteps" :key="step" :value="step" :selected="filter.max === step">{{ kesShort(step) }}</option>
          </select>
        </label>
        <label class="filter">
          <span class="label">Floors</span>
          <select class="select select--small" name="floors">
            <option value="" :selected="filter.floors === ''">All floors</option>
            <option value="lower" :selected="filter.floors === 'lower'">Lower, 2 to 6</option>
            <option value="middle" :selected="filter.floors === 'middle'">Middle, 7 to 11</option>
            <option value="upper" :selected="filter.floors === 'upper'">Upper, 12 to 16</option>
          </select>
        </label>
        <label class="filter">
          <span class="label">Side</span>
          <select class="select select--small" name="side">
            <option value="" :selected="filter.side === ''">Both sides</option>
            <option value="river" :selected="filter.side === 'river'">River side</option>
            <option value="garden" :selected="filter.side === 'garden'">Garden side</option>
          </select>
        </label>
        <label class="filter">
          <span class="label">Sort by</span>
          <select class="select select--small" name="sort">
            <option value="price" :selected="filter.sort === 'price'">Lowest price</option>
            <option value="price-desc" :selected="filter.sort === 'price-desc'">Highest price</option>
            <option value="floor" :selected="filter.sort === 'floor'">Highest floor</option>
            <option value="size" :selected="filter.sort === 'size'">Largest first</option>
          </select>
        </label>
        <label class="filter filter--toggle">
          <input type="checkbox" name="show" value="all" :checked="filter.all" />
          <span>Include sold and reserved</span>
        </label>
        <noscript><button class="btn btn--small" type="submit">Show homes</button></noscript>
      </div>
    </form>
    <div id="results" class="wrap results">
      <RequestView :state="homes.list" :retry="() => homes.loadList(filter, true)" loading-label="Finding homes…">
        <template #default="{ data, busy }">
          <div class="results__bar">
            <p class="results__count" role="status">
              <strong class="tnum">{{ data.homes.length }}</strong> {{ data.homes.length === 1 ? "home" : "homes" }}{{ filter.all ? "" : " available" }}
            </p>
            <div class="results__links">
              <button v-if="narrowed" class="link-arrow as-button" type="button" @click="reset">Clear filters</button>
            </div>
          </div>
          <div v-if="data.homes.length === 0" class="empty">
            <p class="h3">No home matches all of those.</p>
            <p class="muted">Try another price or floor, or include homes that are already reserved to see the whole building.</p>
            <button class="btn btn--line btn--small" type="button" @click="reset">Show every available home</button>
          </div>
          <table v-else class="schedule" :aria-busy="busy">
            <caption class="visually-hidden">Homes matching your choices</caption>
            <thead>
              <tr>
                <th scope="col">Home</th>
                <th scope="col">Type</th>
                <th scope="col" class="num">Level</th>
                <th scope="col" class="num">Size</th>
                <th scope="col">Faces</th>
                <th scope="col" class="num">Sun a day</th>
                <th scope="col" class="num">Price</th>
                <th scope="col">Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="h in data.homes" :key="h.id" class="schedule__row" :class="{ 'is-off': !showsPrice(h) }">
                <th scope="row" class="schedule__home">
                  <RouterLink :to="{ name: 'home-detail', params: { id: h.id } }"
                    ><span class="mono" :style="homeTransition(h.id)">{{ h.id }}</span><span class="visually-hidden">, {{ h.title }}</span></RouterLink
                  >
                </th>
                <td data-label="Type"><span class="schedule__type">{{ h.type }}</span> {{ h.typeShort }}</td>
                <td class="num mono" data-label="Level">{{ levelName(h.level) }}</td>
                <td class="num mono" data-label="Size">{{ h.internal }} m²</td>
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
        </template>
      </RequestView>
    </div>
  </section>
</template>

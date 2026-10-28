<script setup lang="ts">
import { computed } from "vue";
import baySection from "@/generated/bay-section.svg?raw";
import sunPath from "@/generated/sun-path.svg?raw";
import ArrowIcon from "@/components/ArrowIcon.vue";
import ContentHead from "@/components/ContentHead.vue";
import DrawingSheet from "@/components/DrawingSheet.vue";
import PhotoPicture from "@/components/PhotoPicture.vue";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { amenities, specification } from "@/lib/content";
import { shortMonthYear } from "@/lib/format";
import { photo } from "@/lib/photos";
import { useSiteStore } from "@/stores/site";

const site = useSiteStore();
usePageData(() => site.loadOverview());
const overview = computed(() => site.overview.data);

usePageMeta({
  title: "The building",
  description: "Oriel's seventeen storeys of homes in Riverside, Nairobi: the glazed bays, how the building shades itself, the shared spaces and the specification of every home.",
  path: "/building",
  sheet: { number: "A-300", name: "The building" },
  nav: "building",
});
</script>

<template>
  <div class="wrap content-page">
    <ContentHead
      tag="A-300 · The building"
      title="Built around the light."
      lede="Seventeen storeys of homes above a garden by the river, each with a glazed bay at its corner. Here is how the building works, what it shares and what every home comes with."
    />
    <dl class="content-facts">
      <div><dt class="label">Storeys</dt><dd>17</dd></div>
      <div><dt class="label">Homes</dt><dd>86</dd></div>
      <div><dt class="label">Homes per floor</dt><dd>6</dd></div>
      <div><dt class="label">Floor to floor</dt><dd class="tnum">3.2 m</dd></div>
      <div><dt class="label">Completion</dt><dd>{{ overview ? shortMonthYear(overview.completion) : "—" }}</dd></div>
    </dl>
  </div>
  <section class="content-section" aria-labelledby="bay-title">
    <div class="wrap content-split">
      <div class="content-split__text">
        <div class="content-section__head">
          <p class="sheet-tag label">A-301 · The bay</p>
          <h2 id="bay-title" class="h2">A window you can sit in.</h2>
        </div>
        <div class="prose">
          <p>Every home has a bay at its corner: a glazed box that steps 1.2 metres out from the wall, so you can see along the façade as well as straight out. Corner bays look two ways.</p>
          <p>
            The glass starts at a solid oak seat 0.45 metres off the floor, with storage under it, and stops under a cobalt band. The band and the slab above shade the glass from the high midday
            sun, while the low light of early morning reaches far into the room.
          </p>
          <p>
            From the street the bays read as a column of blue-capped lanterns up each corner of the tower. That is the building’s signature, and its name: an oriel is a window that projects
            from a wall to catch more light and more view.
          </p>
        </div>
      </div>
      <figure class="content-sheet">
        <DrawingSheet :svg="baySection" label="Section through a bay" :pan-width="620" :pan="1" />
        <figcaption class="content-sheet__caption"><span class="mono">A-301</span> Section through a bay. Dimensions in metres.</figcaption>
      </figure>
    </div>
    <div v-if="photo('living-bay-b')" class="wrap">
      <figure class="content-photo content-photo--wide crop">
        <PhotoPicture name="living-bay-b" alt="A woman reading on an oak window seat in a corner bay, with a cobalt cushion and treetops outside" sizes="(min-width: 1520px) 1408px, 92vw" class="photo--bay" />
        <figcaption class="content-photo__caption">The bay of a two-bedroom home, on a morning in February.</figcaption>
      </figure>
    </div>
    <div class="wrap">
      <div class="photo-pair">
        <figure class="crop">
          <PhotoPicture name="facade-bays-dusk" alt="Three corner bays stacked up the building at dusk, lit from inside, each under a deep cobalt band" sizes="(min-width: 760px) 45vw, 100vw" />
          <figcaption class="content-photo__caption">The corner bays at dusk, lit like lanterns.</figcaption>
        </figure>
        <figure class="crop">
          <PhotoPicture
            name="facade-detail-day"
            alt="A corner bay in bright sun, its cobalt band casting a shadow on the glass, beside the deep white fins of the side façade"
            sizes="(min-width: 760px) 45vw, 100vw"
          />
          <figcaption class="content-photo__caption">At midday the band shades the glass, and the fins shade the side façade.</figcaption>
        </figure>
      </div>
    </div>
  </section>
  <section class="content-section content-section--rule" aria-labelledby="sun-title">
    <div class="wrap content-split content-split--flip">
      <div class="content-split__text">
        <div class="content-section__head">
          <p class="sheet-tag label">A-302 · Sun and shade</p>
          <h2 id="sun-title" class="h2">Shaped by an overhead sun.</h2>
        </div>
        <div class="prose">
          <p>Riverside is a degree and a quarter south of the equator, so the sun climbs almost straight up and comes down almost straight. It tracks a little north of overhead in June and a little south in December.</p>
          <p>
            That makes the long north and south faces easy to shade: the slab edge and the balconies keep the high sun off the glass. The hard light comes low from the east in the morning and
            the west in the afternoon, so those faces carry deep concrete fins.
          </p>
          <p>Every home’s page shows how many hours of direct sun it gets in each month, worked out from this same sun path and the shade around each window.</p>
        </div>
        <ul class="sun-legend" role="list">
          <li><span class="sun-legend__line sun-legend__line--june" />21 June</li>
          <li><span class="sun-legend__line sun-legend__line--equinox" />20 March and 22 September</li>
          <li><span class="sun-legend__line sun-legend__line--december" />21 December</li>
        </ul>
      </div>
      <figure class="content-sheet content-sheet--square">
        <DrawingSheet :svg="sunPath" label="The sun's path over Riverside" />
        <figcaption class="content-sheet__caption"><span class="mono">A-302</span> The sun’s path over Riverside, as seen from the roof. North is at the top; the centre is straight overhead.</figcaption>
      </figure>
    </div>
  </section>
  <section class="content-section content-section--rule" aria-labelledby="homes-title">
    <div class="wrap">
      <div class="content-section__head">
        <p class="sheet-tag label">A-303 · The homes</p>
        <h2 id="homes-title" class="h2">Five kinds of home.</h2>
      </div>
      <ol v-if="overview" class="type-list" role="list">
        <li v-for="t in overview.types" :key="t.code">
          <RouterLink class="type-list__row" :to="{ name: 'home-type', params: { code: t.code } }">
            <span class="type-list__code mono">{{ t.code }}</span>
            <span class="type-list__name">{{ t.name }}</span>
            <span class="type-list__area tnum">{{ t.internal }} m²</span>
            <span class="type-list__count muted">{{ t.total }} {{ t.total === 1 ? "home" : "homes" }}</span>
            <ArrowIcon />
          </RouterLink>
        </li>
      </ol>
    </div>
  </section>
  <section class="content-section content-section--rule" aria-labelledby="shared-title">
    <div class="wrap">
      <div class="content-section__head">
        <p class="sheet-tag label">A-304 · Shared spaces</p>
        <h2 id="shared-title" class="h2">What every resident shares.</h2>
      </div>
      <ul class="amenities" role="list">
        <li v-for="a in amenities" :key="a.code" class="amenity">
          <div v-if="photo(a.image)" class="amenity__photo">
            <PhotoPicture :name="a.image" :alt="a.alt" sizes="(min-width: 1000px) 33vw, (min-width: 640px) 50vw, 100vw" />
          </div>
          <div v-else class="amenity__key" aria-hidden="true"><span>{{ a.code }}</span></div>
          <div class="amenity__body">
            <p class="label">{{ a.where }}</p>
            <h3 class="h3">{{ a.name }}</h3>
            <p class="muted">{{ a.text }}</p>
          </div>
        </li>
      </ul>
    </div>
  </section>
  <section id="specification" class="content-section content-section--rule" aria-labelledby="spec-title">
    <div class="wrap">
      <div class="spec-head">
        <div class="content-section__head">
          <p class="sheet-tag label">A-305 · Specification</p>
          <h2 id="spec-title" class="h2">What every home comes with.</h2>
          <p class="lede">Every home is finished the same way, from a one-bedroom on level 2 to the penthouses: a short list of honest materials, chosen to age well in Nairobi’s sun.</p>
        </div>
        <figure class="spec-head__photo crop">
          <PhotoPicture
            name="materials"
            alt="Samples of the finishes on a plaster table: board-marked concrete, oak, terrazzo, linen, a graphite window profile, brushed brass and a cobalt glazed tile"
            sizes="(min-width: 1000px) 40vw, 100vw"
          />
          <figcaption class="content-photo__caption">The finishes: concrete, oak, terrazzo, linen, graphite, brass and one cobalt tile.</figcaption>
        </figure>
      </div>
      <div class="spec">
        <div v-for="g in specification" :key="g.code" class="spec__group">
          <h3 class="spec__title"><span class="mono">{{ g.code }}</span> {{ g.title }}</h3>
          <dl class="spec__items">
            <div v-for="([name, text], i) in g.items" :key="name" class="spec__item">
              <dt><span class="mono spec__code">{{ g.code }}.{{ i + 1 }}</span> {{ name }}</dt>
              <dd>{{ text }}</dd>
            </div>
          </dl>
        </div>
      </div>
      <p class="content-note muted">The specification may change in detail as the design is developed; anything that changes will be of the same standard or better.</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import locationMap from "@/generated/location-map.svg?raw";
import places from "@/generated/places.json";
import ArrowIcon from "@/components/ArrowIcon.vue";
import ContentHead from "@/components/ContentHead.vue";
import DrawingSheet from "@/components/DrawingSheet.vue";
import PhotoPicture from "@/components/PhotoPicture.vue";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { monthYear } from "@/lib/format";
import { useFrameStore } from "@/stores/frame";
import { useSiteStore } from "@/stores/site";

const site = useSiteStore();
const frame = useFrameStore();
usePageData(() => site.loadProgress());
const showFlats = computed(() => site.progress.data?.showFlats ?? "");

usePageMeta({
  title: "Location",
  description: "Riverside, Nairobi: five minutes from Westlands, twelve from the city centre and fifteen from Karura Forest. How far everything is, and how to reach the sales suite.",
  path: "/location",
  sheet: { number: "A-400", name: "Location" },
  nav: "location",
});
</script>

<template>
  <div class="wrap content-page">
    <ContentHead
      tag="A-400 · Location"
      title="Between the forest and the city."
      lede="Riverside is a quiet, green neighbourhood on the river between Westlands and the city centre: five minutes from one, twelve from the other, and fifteen from Karura Forest."
    />
  </div>
  <section class="content-section content-section--first" aria-labelledby="map-heading">
    <div class="wrap location">
      <figure class="content-sheet location__map">
        <DrawingSheet :svg="locationMap" label="Map of Riverside" :pan-width="760" :pan="0.4" />
        <figcaption class="content-sheet__caption"><span class="mono">A-401</span> Riverside and the places around it. A diagram: roads and distances are simplified.</figcaption>
      </figure>
      <div class="location__list">
        <h2 id="map-heading" class="h3">How far it is</h2>
        <dl class="places">
          <div v-for="p in places" :key="p.name">
            <dt>{{ p.name }}</dt>
            <dd><span class="tnum">{{ p.minutes }} min</span> <span class="muted">{{ p.how }}</span></dd>
          </div>
        </dl>
        <p class="content-note muted">Times by car are outside the rush hour; allow longer between 7 and 9 in the morning and 5 and 7 in the evening.</p>
      </div>
    </div>
  </section>
  <section class="content-section content-section--rule" aria-labelledby="around-title">
    <div class="wrap">
      <div class="content-section__head">
        <p class="sheet-tag label">A-402 · Around the corner</p>
        <h2 id="around-title" class="h2">Everything close, nothing loud.</h2>
      </div>
      <ul class="nearby" role="list">
        <li><span class="label">Green space</span><p>The Arboretum is a ten-minute walk; Karura Forest, with some fifty kilometres of trails and a waterfall, is fifteen minutes’ drive.</p></li>
        <li><span class="label">Schools</span><p>Four international schools and several of Nairobi’s best-known Kenyan schools are within fifteen minutes.</p></li>
        <li><span class="label">Health</span><p>Two of the city’s main private hospitals are within ten minutes, and a pharmacy and clinic in Westlands.</p></li>
        <li><span class="label">Shopping</span><p>Westlands’ shopping centres, markets and restaurants are five minutes away; there is a grocer on Riverside Drive.</p></li>
        <li><span class="label">Work</span><p>Westlands, the city centre and Upper Hill are all within fifteen minutes; the UN offices in Gigiri are eighteen.</p></li>
        <li><span class="label">Travel</span><p>The Expressway joins Waiyaki Way at Westlands and reaches the airport in about thirty-five minutes.</p></li>
      </ul>
      <div class="content-photos">
        <PhotoPicture name="neighbourhood-street" alt="A quiet tree-lined street in Riverside in the morning" sizes="(min-width: 1000px) 60vw, 100vw" class="content-photos__wide" />
        <PhotoPicture name="forest-path" alt="A path through tall trees in Karura Forest" sizes="(min-width: 1000px) 34vw, 100vw" class="content-photos__tall" />
      </div>
    </div>
  </section>
  <section id="getting-here" class="content-section content-section--rule" aria-labelledby="here-title">
    <div class="wrap content-split">
      <div class="content-split__text">
        <div class="content-section__head">
          <p class="sheet-tag label">A-403 · Getting here</p>
          <h2 id="here-title" class="h2">Visiting the sales suite.</h2>
        </div>
        <div class="prose">
          <p>The sales suite is on site, off Riverside Drive: look for the Oriel hoarding and the blue gate. There is visitor parking inside the gate.</p>
          <p>By taxi or ride-hailing app, ask for the Oriel sales suite on Riverside Drive. By matatu, the Riverside Drive stop is a three-minute walk.</p>
          <p v-if="showFlats && frame.today >= showFlats">Viewings are now held in the building itself, in finished show apartments on level 3.</p>
          <p v-else-if="showFlats">From {{ monthYear(showFlats) }} viewings move into the building itself, to finished show apartments on level 3.</p>
        </div>
      </div>
      <div class="side-card here-card">
        <h3 class="label">Opening hours</h3>
        <dl class="hours">
          <div><dt>Tuesday to Sunday</dt><dd class="mono">10:00 to 17:00</dd></div>
          <div><dt>Monday</dt><dd class="mono">Closed</dd></div>
          <div><dt>Public holidays</dt><dd class="mono">Closed</dd></div>
        </dl>
        <RouterLink class="btn btn--cobalt" :to="{ name: 'viewing-new' }">Book a viewing <ArrowIcon /></RouterLink>
      </div>
    </div>
  </section>
</template>

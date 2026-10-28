<script setup lang="ts">
import { computed } from "vue";
import ArrowIcon from "@/components/ArrowIcon.vue";
import { drawn } from "@/lib/format";
import { useFrameStore } from "@/stores/frame";

const frame = useFrameStore();
const revision = computed(() => frame.release.slice(0, 7).toUpperCase());
const year = computed(() => frame.today.slice(0, 4));
</script>

<template>
  <footer class="site-footer">
    <div class="wrap">
      <div v-if="!frame.page.quiet" class="footer-cta">
        <h2 class="h2">See the light for yourself.</h2>
        <div class="hero__actions">
          <RouterLink class="btn btn--cobalt" :to="{ name: 'viewing-new' }">Book a viewing <ArrowIcon /></RouterLink>
          <RouterLink class="btn btn--line" :to="{ name: 'residences' }">Find a home</RouterLink>
        </div>
      </div>
      <div class="footer-cols">
        <div class="footer-col footer-col--visit">
          <span class="label">Sales suite</span>
          <p>On site at Oriel, Riverside, Nairobi. Open Tuesday to Sunday, 10:00 to 17:00, by appointment.</p>
          <RouterLink class="link-arrow" :to="{ name: 'location' }">Getting here <ArrowIcon /></RouterLink>
        </div>
        <div class="footer-col">
          <span class="label">The building</span>
          <RouterLink :to="{ name: 'residences' }">Residences</RouterLink>
          <RouterLink :to="{ name: 'explore' }">Explore in 3D</RouterLink>
          <RouterLink :to="{ name: 'building' }">Amenities and specification</RouterLink>
          <RouterLink :to="{ name: 'progress' }">Construction progress</RouterLink>
        </div>
        <div class="footer-col">
          <span class="label">Buying</span>
          <RouterLink :to="{ name: 'buying' }">Buying off-plan</RouterLink>
          <RouterLink :to="{ name: 'buying', hash: '#payment-plans' }">Payment plans</RouterLink>
          <a href="/api/price-list.pdf">Price list (PDF)</a>
          <RouterLink :to="{ name: 'viewing-new' }">Book a viewing</RouterLink>
        </div>
        <div class="footer-col">
          <span class="label">Oriel</span>
          <RouterLink :to="{ name: 'enquire' }">Ask a question</RouterLink>
          <RouterLink :to="{ name: 'location' }">Location</RouterLink>
          <RouterLink :to="{ name: 'privacy' }">Privacy</RouterLink>
          <RouterLink :to="{ name: 'terms' }">Terms</RouterLink>
        </div>
      </div>
      <svg class="footer-mark" viewBox="0 0 1000 214" aria-hidden="true" focusable="false"><text x="0" y="200" textLength="1000" lengthAdjust="spacingAndGlyphs">ORIEL</text></svg>
      <dl class="title-block">
        <div><dt class="label">Project</dt><dd>Oriel, Riverside, Nairobi</dd></div>
        <div><dt class="label">Sheet</dt><dd>{{ frame.page.sheet.number }} · {{ frame.page.sheet.name }}</dd></div>
        <div><dt class="label">Drawn</dt><dd>{{ frame.today ? drawn(frame.today) : "" }}</dd></div>
        <div><dt class="label">Rev.</dt><dd>{{ revision }}</dd></div>
        <div><dt class="label">©</dt><dd>{{ year }} Oriel Residences Limited</dd></div>
      </dl>
    </div>
  </footer>
</template>

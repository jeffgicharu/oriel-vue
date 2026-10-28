<script setup lang="ts">
import ContentHead from "@/components/ContentHead.vue";
import { usePageMeta } from "@/composables/usePageMeta";

usePageMeta({
  title: "Terms of use",
  description: "The terms of using the Oriel website. Buying a home is governed by its own letter of offer and agreement for sale.",
  path: "/terms",
  sheet: { number: "A-901", name: "Terms" },
});
</script>

<template>
  <div class="wrap content-page">
    <ContentHead tag="A-901 · Terms" title="Terms of use." lede="The fine print for using this website. Buying a home is governed by its own letter of offer and agreement for sale, not by these terms." />
    <div class="prose legal">
      <h2 class="h3">Information on this site</h2>
      <p>The pictures, plans, dimensions, specification and prices on this site show the building as designed. They are a guide, may change as the design is developed, and do not form part of any offer or contract.</p>
      <p>Floor areas are measured to the inside face of external walls and exclude the balcony or terrace, which is given separately. Plans are not to scale unless dimensioned.</p>
      <h2 class="h3">Prices and availability</h2>
      <p>Prices and availability change as homes are sold and are confirmed only in a letter of offer. A reservation request holds a home for 72 hours; it becomes binding only when the sales team issues a letter of offer and the deposit is paid.</p>
      <h2 class="h3">Using the site</h2>
      <p>You may use this site to find out about Oriel and to arrange viewings and reservations for yourself. Please don’t try to break it, overload it, or collect its content automatically.</p>
      <p>The design, drawings, text and pictures belong to Oriel Residences Limited and may not be copied without permission.</p>
      <h2 class="h3">Law</h2>
      <p>These terms are governed by the laws of Kenya.</p>
      <p class="muted">Last updated 17 October 2026.</p>
    </div>
  </div>
</template>

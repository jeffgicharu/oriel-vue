<script setup lang="ts">
import ArrowIcon from "@/components/ArrowIcon.vue";
import { useHttpStatus } from "@/composables/useHttpStatus";
import { usePageMeta } from "@/composables/usePageMeta";
import { useRoute } from "vue-router";

const route = useRoute();
useHttpStatus()(404);
usePageMeta({
  title: "Page not found",
  description: "This page could not be found at Oriel, Riverside.",
  path: route.path,
  sheet: { number: "A-404", name: "Not found" },
});
</script>

<template>
  <section class="section drawing-grid error-page">
    <div class="wrap error-page__inner">
      <p class="label label--cobalt">Error 404 · Sheet not found</p>
      <h1 class="h1">This page isn’t on the plans.</h1>
      <p class="lede">The address may be mistyped, or the page may have moved. These will take you somewhere useful.</p>
      <div class="hero__actions">
        <RouterLink class="btn btn--cobalt" :to="{ name: 'residences' }">See the homes <ArrowIcon /></RouterLink>
        <RouterLink class="btn btn--line" :to="{ name: 'home' }">Back to the start</RouterLink>
      </div>
    </div>
  </section>
</template>

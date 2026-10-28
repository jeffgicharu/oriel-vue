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
  <div class="wrap error-page">
    <p class="sheet-tag label">A-404 · Not found</p>
    <h1 class="h1">Page not found.</h1>
    <p class="lede">There is nothing at this address. The home or page may have moved, or the link may have a typing mistake in it.</p>
    <div class="hero__actions">
      <RouterLink class="btn btn--cobalt" :to="{ name: 'residences' }">See every home <ArrowIcon /></RouterLink>
      <RouterLink class="btn btn--line" :to="{ name: 'home' }">Back to the start</RouterLink>
    </div>
  </div>
</template>

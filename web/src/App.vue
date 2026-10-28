<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import SiteFooter from "@/components/SiteFooter.vue";
import SiteHeader from "@/components/SiteHeader.vue";
import { useShortlistStore } from "@/stores/shortlist";

const router = useRouter();
const route = useRoute();
// The sales desk has a frame of its own.
const desk = computed(() => route.meta.desk === true);
const shortlist = useShortlistStore();
const announcement = ref("");
const main = ref<HTMLElement | null>(null);

onMounted(() => {
  shortlist.restore();
  // After moving to another page in the browser, a screen reader hears the
  // new page's title and focus starts at its content, as on a full page load.
  router.afterEach((to, from, failure) => {
    if (failure || to.path === from.path) return;
    void nextTick(() => {
      announcement.value = document.title;
      // The desk's frame has a main of its own, with the same id.
      (main.value ?? document.getElementById("main"))?.focus({ preventScroll: true });
    });
  });
});
</script>

<template>
  <RouterView v-if="desk" />
  <template v-else>
    <a class="skip-link" href="#main">Skip to content</a>
    <SiteHeader />
    <main id="main" ref="main" tabindex="-1">
      <RouterView />
    </main>
    <SiteFooter />
  </template>
  <p class="visually-hidden" aria-live="polite" aria-atomic="true">{{ announcement }}</p>
</template>

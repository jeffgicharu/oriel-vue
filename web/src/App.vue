<script setup lang="ts">
import { nextTick, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import SiteFooter from "@/components/SiteFooter.vue";
import SiteHeader from "@/components/SiteHeader.vue";
import { useShortlistStore } from "@/stores/shortlist";

const router = useRouter();
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
      main.value?.focus({ preventScroll: true });
    });
  });
});
</script>

<template>
  <a class="skip-link" href="#main">Skip to content</a>
  <SiteHeader />
  <main id="main" ref="main" tabindex="-1">
    <RouterView />
  </main>
  <SiteFooter />
  <p class="visually-hidden" aria-live="polite" aria-atomic="true">{{ announcement }}</p>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";

/**
 * A floor plan's frame. The plan draws itself the first time it comes into
 * view, walls first, then glass and doors, then the rest; and its furniture
 * can be switched off to see the bare rooms.
 */
withDefaults(defineProps<{ furniture?: boolean }>(), { furniture: true });

const figure = ref<HTMLElement | null>(null);
const drawing = ref(false);
let observer: IntersectionObserver | undefined;

onMounted(() => {
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches || !("IntersectionObserver" in window) || !figure.value) return;
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting)) {
        drawing.value = true;
        observer?.disconnect();
      }
    },
    { threshold: 0.25 },
  );
  observer.observe(figure.value);
});

onBeforeUnmount(() => observer?.disconnect());
</script>

<template>
  <figure ref="figure" class="plan-figure" :class="{ 'is-drawing': drawing, 'hide-furniture': !furniture }">
    <slot />
  </figure>
</template>

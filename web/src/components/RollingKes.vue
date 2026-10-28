<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue";
import { kes } from "@/lib/format";

/**
 * An amount that rolls from the old figure to the new one when it changes,
 * such as a total when the buyer picks another plan, so the change can be
 * read. It is still at once for anyone who prefers less motion.
 */
const props = defineProps<{ amount: number }>();
const shown = ref(props.amount);
let frame = 0;

watch(
  () => props.amount,
  (to, from) => {
    cancelAnimationFrame(frame);
    if (from === undefined || window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      shown.value = to;
      return;
    }
    const start = performance.now();
    const length = 620;
    const step = (now: number) => {
      const t = Math.min(1, (now - start) / length);
      const eased = 1 - Math.pow(1 - t, 3);
      shown.value = Math.round(from + (to - from) * eased);
      if (t < 1) frame = requestAnimationFrame(step);
    };
    frame = requestAnimationFrame(step);
  },
);

onBeforeUnmount(() => cancelAnimationFrame(frame));
</script>

<template>
  <span>{{ kes(shown) }}</span>
</template>

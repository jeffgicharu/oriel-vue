<script setup lang="ts">
import { computed } from "vue";
import { n1, statusBar } from "@/lib/deskCharts";

// Every home as one bar, divided by status, hatched as the drawings are.
const props = defineProps<{ counts: Record<string, number>; total: number }>();
const bar = computed(() => statusBar(props.counts, props.total));
</script>

<template>
  <svg class="chart chart--status" viewBox="0 0 1000 44" preserveAspectRatio="none" role="img" :aria-label="bar.summary" xmlns="http://www.w3.org/2000/svg">
    <defs>
      <pattern id="cs-held" width="7" height="7" patternUnits="userSpaceOnUse" patternTransform="rotate(45)"><line x1="0" y1="0" x2="0" y2="7" /></pattern>
      <pattern id="cs-reserved" width="7" height="7" patternUnits="userSpaceOnUse"><path d="M0 0L7 7M7 0L0 7" /></pattern>
    </defs>
    <rect v-for="s in bar.segments" :key="s.status" :class="`cs-${s.status}`" :x="n1(s.x)" y="0" :width="n1(s.w)" height="44">
      <title>{{ s.name }}: {{ s.n }}</title>
    </rect>
  </svg>
</template>

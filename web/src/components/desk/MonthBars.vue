<script setup lang="ts">
import { computed } from "vue";
import { monthBars, monthChart, monthTitle, n1 } from "@/lib/deskCharts";

// The homes sold each month, the latest drawn solid.
const props = defineProps<{ months: { label: string; sold: number; value: number }[] }>();
const chart = computed(() => monthBars(props.months));
</script>

<template>
  <svg class="chart chart--months" :viewBox="`0 0 ${monthChart.w} ${monthChart.h}`" role="img" :aria-label="chart.summary" xmlns="http://www.w3.org/2000/svg">
    <line class="chart__base" x1="0" :y1="monthChart.base" :x2="monthChart.w" :y2="monthChart.base" />
    <g v-for="b in chart.bars" :key="b.label">
      <rect class="chart__bar" :class="{ 'is-current': b.current }" :x="n1(b.x)" :y="n1(b.y)" :width="n1(b.w)" :height="n1(b.h)">
        <title>{{ monthTitle(b) }}</title>
      </rect>
      <text class="chart__value" :x="n1(b.x + b.w / 2)" :y="n1(b.y - 8)" text-anchor="middle">{{ b.n }}</text>
      <text class="chart__label" :x="n1(b.x + b.w / 2)" :y="n1(monthChart.base + 20)" text-anchor="middle">{{ b.label }}</text>
    </g>
  </svg>
</template>

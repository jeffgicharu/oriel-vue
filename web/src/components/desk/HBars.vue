<script setup lang="ts">
import { computed } from "vue";
import { hBars, hbarChart, n1 } from "@/lib/deskCharts";

// A horizontal bar for each label, with its figure at the end.
const props = defineProps<{ items: { label: string; n: number }[]; unit: string; title?: string }>();
const chart = computed(() => hBars(props.items));
const label = computed(() => (props.title ? `${props.title}: ${chart.value.summary}` : chart.value.summary));
</script>

<template>
  <svg class="chart chart--hbars" :viewBox="`0 0 ${hbarChart.w} ${chart.height}`" role="img" :aria-label="label" xmlns="http://www.w3.org/2000/svg">
    <g v-for="b in chart.bars" :key="b.label">
      <text class="chart__label chart__label--left" x="0" :y="n1(b.y + hbarChart.row / 2)" dominant-baseline="middle">{{ b.label }}</text>
      <rect class="chart__bar" :x="hbarChart.labelW" :y="n1(b.y + hbarChart.row / 2 - 7)" :width="n1(b.w)" height="14">
        <title>{{ b.label }}: {{ b.n }} {{ unit }}</title>
      </rect>
      <text class="chart__value" :x="n1(hbarChart.labelW + b.w + 8)" :y="n1(b.y + hbarChart.row / 2)" dominant-baseline="middle">{{ b.n }}</text>
    </g>
  </svg>
</template>

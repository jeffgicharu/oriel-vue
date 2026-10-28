<script setup lang="ts">
import { computed } from "vue";
import type { HomeDetail } from "@/api/client";
import { monthName, oneDecimal } from "@/lib/format";
import { barPath, bars, chart, chartY, lightSummary } from "@/lib/light";

const props = defineProps<{ light: HomeDetail["light"] }>();
const laid = computed(() => bars(props.light));
const grid = computed(() => {
  const out = [];
  for (let v = 0; v <= laid.value.max; v += 2) out.push(v);
  return out;
});
const letters = "JFMAMJJASOND";
const f1 = (v: number) => v.toFixed(1);
</script>

<template>
  <figure class="light-chart">
    <div class="light-chart__legend" aria-hidden="true">
      <span><i class="swatch swatch--morning" />Morning</span>
      <span><i class="swatch swatch--afternoon" />Afternoon</span>
    </div>
    <svg :viewBox="`0 0 ${f1(chart.w)} ${f1(chart.h)}`" role="img" :aria-label="lightSummary(light)">
      <template v-for="g in grid" :key="g">
        <path class="light-chart__grid" :d="`M${f1(chart.left)} ${f1(chartY(g, laid.max))}H${f1(chart.w - 8)}`" />
        <text class="light-chart__axis" :x="f1(chart.left - 8)" :y="f1(chartY(g, laid.max))" text-anchor="end" dominant-baseline="middle">{{ g }}h</text>
      </template>
      <g v-for="b in laid.bars" :key="b.month" class="light-chart__bar" tabindex="0">
        <title>{{ monthName(b.month) }}: {{ oneDecimal(b.total) }} hours, {{ oneDecimal(b.morning) }} in the morning and {{ oneDecimal(b.afternoon) }} in the afternoon</title>
        <rect class="light-chart__hit" :x="f1(b.x - 4)" :y="f1(chart.top)" :width="f1(b.w + 8)" :height="f1(chart.base - chart.top)" />
        <path v-if="b.morningH > 0" class="light-chart__morning" :d="barPath(b.x, b.morningY, b.w, b.morningH, b.afternoonH <= 0)" />
        <path v-if="b.afternoonH > 0" class="light-chart__afternoon" :d="barPath(b.x, b.afternoonY, b.w, b.afternoonH, true)" />
        <text class="light-chart__value" :class="{ 'is-shown': b.labelled }" :x="f1(b.x + b.w / 2)" :y="f1(Math.min(b.morningY, b.afternoonY) - 7)" text-anchor="middle">{{ oneDecimal(b.total) }}</text>
        <text class="light-chart__axis" :x="f1(b.x + b.w / 2)" :y="f1(chart.base + 18)" text-anchor="middle">{{ letters[b.month - 1] }}</text>
      </g>
      <path class="light-chart__base" :d="`M${f1(chart.left)} ${f1(chart.base)}H${f1(chart.w - 8)}`" />
    </svg>
    <details class="light-chart__table">
      <summary class="label">See the figures</summary>
      <table>
        <thead>
          <tr><th scope="col">Month</th><th scope="col" class="num">Morning</th><th scope="col" class="num">Afternoon</th><th scope="col" class="num">Total</th></tr>
        </thead>
        <tbody>
          <tr v-for="b in laid.bars" :key="b.month">
            <th scope="row">{{ monthName(b.month) }}</th>
            <td class="num mono">{{ oneDecimal(b.morning) }} h</td>
            <td class="num mono">{{ oneDecimal(b.afternoon) }} h</td>
            <td class="num mono">{{ oneDecimal(b.total) }} h</td>
          </tr>
        </tbody>
      </table>
    </details>
  </figure>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { channelBars, channelChart, n1, SERIES, type ChannelRow } from "@/lib/deskCharts";

/**
 * The website's leads by the channel they first arrived through: how many
 * came, how many viewed and how many asked to reserve, as three bars on one
 * scale, with the same figures in a table beside the chart.
 */
const props = defineProps<{ rows: ChannelRow[] }>();
const chart = computed(() => channelBars(props.rows));
const totals = computed(() =>
  props.rows.reduce((t, r) => ({ leads: t.leads + r.leads, viewed: t.viewed + r.viewed, reserved: t.reserved + r.reserved }), { leads: 0, viewed: 0, reserved: 0 }),
);
const { labelW } = channelChart;
</script>

<template>
  <div class="channels">
    <figure class="channels__chart">
      <ul class="desk-legend channels__key" aria-hidden="true">
        <li v-for="s in SERIES" :key="s.key"><span class="chart-key" :class="`chart-key--${s.key}`" />{{ s.label }}</li>
      </ul>
      <svg class="chart chart--channels" :viewBox="`0 0 ${channelChart.w} ${chart.height}`" role="img" :aria-label="`Leads from the website by channel. ${chart.summary}.`" xmlns="http://www.w3.org/2000/svg">
        <g v-for="g in chart.groups" :key="g.channel">
          <text class="chart__label chart__label--left" x="0" :y="n1(g.mid)" dominant-baseline="middle">{{ g.channel }}</text>
          <template v-for="b in g.bars" :key="b.key">
            <rect v-if="b.w > 0" :class="b.className" :x="labelW" :y="n1(b.y)" :width="n1(b.w)" :height="channelChart.bar">
              <title>{{ g.channel }}: {{ b.n }} {{ b.label.toLowerCase() }}</title>
            </rect>
            <text class="chart__value chart__value--small" :x="n1(labelW + b.w + 6)" :y="n1(b.y + channelChart.bar / 2)" dominant-baseline="middle">{{ b.n }}</text>
          </template>
        </g>
      </svg>
    </figure>
    <div class="desk-table-wrap channels__table" role="region" aria-label="The figures, by channel" tabindex="0">
      <table class="desk-table">
        <caption class="visually-hidden">Leads from the website by channel</caption>
        <thead>
          <tr>
            <th scope="col">Channel</th>
            <th scope="col" class="num">Leads</th>
            <th scope="col" class="num">Viewed</th>
            <th scope="col" class="num">Reserved</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in rows" :key="r.channel">
            <th scope="row">{{ r.channel }}</th>
            <td class="num">{{ r.leads }}</td>
            <td class="num">{{ r.viewed }}</td>
            <td class="num">{{ r.reserved }}</td>
          </tr>
        </tbody>
        <tfoot>
          <tr>
            <th scope="row">Every channel</th>
            <td class="num">{{ totals.leads }}</td>
            <td class="num">{{ totals.viewed }}</td>
            <td class="num">{{ totals.reserved }}</td>
          </tr>
        </tfoot>
      </table>
    </div>
  </div>
</template>

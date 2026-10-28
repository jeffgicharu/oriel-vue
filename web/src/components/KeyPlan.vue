<script setup lang="ts">
import { computed } from "vue";
import type { HomeDetail } from "@/api/client";

/**
 * The floor a home is on, from above with north up and the river beyond the
 * north edge, with the home picked out: the small plan an architect puts in
 * the corner of a sheet to show where a drawing belongs.
 */
const props = defineProps<{ floor: HomeDetail["floor"]; number: number; homeId: string }>();

const S = 9; // pixels to the metre
const k = computed(() => {
  const [xMin, zMin, xMax, zMax] = props.floor.plate;
  const ox = -xMin + 3;
  const oz = -zMin + 8;
  const x = (v: number) => (v + ox) * S;
  const z = (v: number) => (v + oz) * S;
  const w = (xMax - xMin + 6) * S;
  const h = (zMax - zMin + 14) * S;
  const f1 = (v: number) => v.toFixed(1);
  return {
    viewBox: `0 0 ${w.toFixed(0)} ${h.toFixed(0)}`,
    river: `M0 ${f1(z(zMin - 5.5))}C${f1(w * 0.3)} ${f1(z(zMin - 3.5))} ${f1(w * 0.65)} ${f1(z(zMin - 7))} ${f1(w)} ${f1(z(zMin - 5))}`,
    riverLabel: { x: f1(x(xMin)), y: f1(z(zMin - 6.6)) },
    core: props.floor.core ? { x: f1(x(props.floor.core[0])), y: f1(z(props.floor.core[1])), w: f1((props.floor.core[2] - props.floor.core[0]) * S), h: f1((props.floor.core[3] - props.floor.core[1]) * S) } : null,
    homes: props.floor.homes.map((p) => ({
      number: p.number,
      foot: { x: f1(x(p.foot[0])), y: f1(z(p.foot[1])), w: f1((p.foot[2] - p.foot[0]) * S), h: f1((p.foot[3] - p.foot[1]) * S) },
      bay: { x: f1(x(p.bay[0])), y: f1(z(p.bay[1])), w: f1((p.bay[2] - p.bay[0]) * S), h: f1((p.bay[3] - p.bay[1]) * S) },
      cx: f1(x((p.foot[0] + p.foot[2]) / 2)),
      cy: f1(z((p.foot[1] + p.foot[3]) / 2)),
    })),
    north: `translate(${f1(w - 16)} 26.0)`,
  };
});
</script>

<template>
  <svg class="keyplan" :viewBox="k.viewBox" role="img" :aria-label="`Where home ${homeId} sits on level ${floor.level}`">
    <path class="keyplan__river" :d="k.river" />
    <text class="keyplan__note" :x="k.riverLabel.x" :y="k.riverLabel.y">River</text>
    <rect v-if="k.core" class="keyplan__core" :x="k.core.x" :y="k.core.y" :width="k.core.w" :height="k.core.h" />
    <template v-for="p in k.homes" :key="p.number">
      <rect class="keyplan__home" :class="{ 'is-this': p.number === number }" :x="p.foot.x" :y="p.foot.y" :width="p.foot.w" :height="p.foot.h" />
      <rect class="keyplan__home keyplan__bay" :class="{ 'is-this': p.number === number }" :x="p.bay.x" :y="p.bay.y" :width="p.bay.w" :height="p.bay.h" />
      <text class="keyplan__num" :class="{ 'is-this': p.number === number }" :x="p.cx" :y="p.cy" text-anchor="middle" dominant-baseline="middle">{{ String(p.number).padStart(2, "0") }}</text>
    </template>
    <g class="keyplan__north" :transform="k.north"><path d="M0 -12L5 6L0 2L-5 6Z" /><text y="-16" text-anchor="middle">N</text></g>
  </svg>
</template>

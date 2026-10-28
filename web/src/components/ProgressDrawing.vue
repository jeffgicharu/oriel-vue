<script setup lang="ts">
import { computed } from "vue";
import type { BuildingModel } from "@/api/client";

/**
 * The south elevation drawn as the frame stands: storeys built solid, the one
 * under way hatched, the rest in dashed outline, and the crane above.
 */
const props = defineProps<{ model: BuildingModel; frame: number; storeys: number; label: string }>();

const K = 6; // pixels a metre
const LEFT = 70;

const d = computed(() => {
  const m = props.model;
  const plateWidth = m.plate[2] - m.plate[0];
  const width = plateWidth * K;
  const topLevel = m.levels[m.levels.length - 1]!.level;
  const top = m.roof + 1.2;
  const H = top * K + 70;
  const ground = H - 40;
  const y = (h: number) => ground - h * K;
  const f1 = (v: number) => (Math.round(v * 10) / 10).toString();
  const built = Math.floor(props.frame);
  const rising = props.frame > 0 && props.frame < props.storeys;

  const storeys = [];
  for (let level = 0; level <= topLevel + 1; level++) {
    const lv = m.levels[Math.min(level, topLevel)]!;
    const [y0, y1] = level <= topLevel ? [lv.y, lv.y + lv.h] : [m.roof, m.roof + 1.2];
    const inset = level >= topLevel ? m.setback * K : 0;
    const cls = level < built ? "pd-built" : level === built && props.frame < props.storeys ? "pd-now" : "pd-future";
    storeys.push({
      level,
      cls,
      x: f1(LEFT + inset),
      y: f1(y(y1)),
      w: f1(width - 2 * inset),
      h: f1((y1 - y0) * K),
      label: level <= topLevel && level % 2 === 0 ? `L${String(level).padStart(2, "0")}` : "",
      ly: f1(y((y0 + y1) / 2)),
    });
  }

  // The tower crane stands in the lift core and climbs with the frame.
  let crane = null;
  let today = null;
  if (rising) {
    const reach = m.levels[Math.min(built, topLevel)]!.y + 14;
    const cx = LEFT + width * 0.6;
    const jib = y(reach - 2);
    crane = [
      [cx, ground, cx, y(reach + 4)],
      [cx - 150, jib, cx + 60, jib],
      [cx, y(reach + 4), cx - 150, jib],
      [cx, y(reach + 4), cx + 60, jib],
    ].map((l) => l.map(f1));
    const hook = [cx - 100, jib, cx - 100, jib + 40].map(f1);
    const ty = f1(y((built <= topLevel ? m.levels[built]!.y : m.roof) + 1.6));
    crane.push(hook);
    today = { x1: f1(LEFT + width + 4), x2: f1(LEFT + width + 26), tx: f1(LEFT + width + 30), y: ty };
  }
  return { viewBox: `0 0 ${f1(LEFT + width + 120)} ${f1(H)}`, ground: { y: f1(ground), x2: f1(LEFT + width + 110) }, storeys, crane, today };
});
</script>

<template>
  <svg class="progress-drawing" :viewBox="d.viewBox" role="img" aria-labelledby="pd-title" xmlns="http://www.w3.org/2000/svg">
    <title id="pd-title">The frame today: {{ label }}</title>
    <defs>
      <pattern id="pd-hatch" width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(45)"><line x1="0" y1="0" x2="0" y2="6" /></pattern>
    </defs>
    <line class="pd-ground" x1="10" :y1="d.ground.y" :x2="d.ground.x2" :y2="d.ground.y" />
    <template v-for="s in d.storeys" :key="s.level">
      <rect :class="s.cls" :x="s.x" :y="s.y" :width="s.w" :height="s.h" />
      <text v-if="s.label" class="pd-level" :x="String(LEFT - 10)" :y="s.ly" text-anchor="end" dominant-baseline="middle">{{ s.label }}</text>
    </template>
    <g v-if="d.crane" class="pd-crane">
      <line v-for="(l, i) in d.crane" :key="i" :class="{ 'pd-hook': i === d.crane.length - 1 }" :x1="l[0]" :y1="l[1]" :x2="l[2]" :y2="l[3]" />
    </g>
    <g v-if="d.today" class="pd-today">
      <line :x1="d.today.x1" :y1="d.today.y" :x2="d.today.x2" :y2="d.today.y" />
      <text :x="d.today.tx" :y="d.today.y" dominant-baseline="middle">Today</text>
    </g>
  </svg>
</template>

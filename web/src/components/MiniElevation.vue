<script setup lang="ts">
import { computed } from "vue";
import { levelName } from "@/lib/format";

/** The tower's levels as bands with the home's level picked out, for the side of a home's page. */
const props = defineProps<{ level: number }>();

const TOP_LEVEL = 16;
const PENTHOUSE_LEVEL = 16;
const FIRST_HOME_LEVEL = 2;
const W = 120;
const FLOOR = 9;
const TOP = 12;

const bands = computed(() => {
  const out = [];
  for (let l = 0; l <= TOP_LEVEL; l++) {
    const y = TOP + (TOP_LEVEL - l) * FLOOR;
    const [x0, x1] = l === PENTHOUSE_LEVEL ? [16, W - 6] : [10, W];
    // Labels for the top, the ground and the home's own level, never two
    // close enough to touch.
    const near = l !== props.level && (Math.abs(l - props.level) === 1 || (props.level <= 2 && l === 0));
    const labelled = (l === props.level || l === TOP_LEVEL || l === 0) && !(near && l !== props.level);
    out.push({
      level: l,
      x: x0.toFixed(1),
      y: (y + 1).toFixed(1),
      w: (x1 - x0).toFixed(1),
      h: (FLOOR - 2).toFixed(1),
      cls: { "is-this": l === props.level, "is-shared": l < FIRST_HOME_LEVEL },
      label: labelled ? (l === 0 ? "G · Lobby" : levelName(l)) : "",
      ly: (y + FLOOR / 2).toFixed(1),
    });
  }
  return out;
});
const height = TOP + (TOP_LEVEL + 1) * FLOOR + 14;
const ground = `M0 ${(TOP + (TOP_LEVEL + 1) * FLOOR + 1).toFixed(1)}H${(W + 10).toFixed(1)}`;
</script>

<template>
  <svg class="minielev" :viewBox="`0 0 ${W + 70} ${height}`" role="img" :aria-label="`Level ${level} of ${TOP_LEVEL}`">
    <template v-for="b in bands" :key="b.level">
      <rect class="minielev__floor" :class="b.cls" :x="b.x" :y="b.y" :width="b.w" :height="b.h" />
      <text v-if="b.label" class="minielev__label" :class="{ 'is-this': b.level === level }" :x="(W + 8).toFixed(1)" :y="b.ly" dominant-baseline="middle">{{ b.label }}</text>
    </template>
    <path class="minielev__ground" :d="ground" />
  </svg>
</template>

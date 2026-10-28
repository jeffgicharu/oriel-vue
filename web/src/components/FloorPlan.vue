<script setup lang="ts">
import { computed } from "vue";
import type { Plan } from "@/api/client";
import { drawPlan } from "@/lib/floorplan";

const props = withDefaults(
  defineProps<{
    plan: Plan;
    /** Prefixes the drawing's ids, so several plans can share a page. */
    id?: string;
    /** Leaves out furniture and dimensions, for small drawings. */
    plain?: boolean;
    /** The direction of north on the drawing, in degrees clockwise from up. */
    north?: number;
    title?: string;
  }>(),
  { id: "", plain: false, north: 0, title: "" },
);

const drawing = computed(() => drawPlan(props.plan, { id: props.id, plain: props.plain, north: props.north }));
const label = computed(() => props.title || `Floor plan of type ${props.plan.type}`);
</script>

<template>
  <svg class="floorplan" :viewBox="drawing.viewBox" role="img" :aria-labelledby="`${drawing.id}-title`" xmlns="http://www.w3.org/2000/svg">
    <title :id="`${drawing.id}-title`">{{ label }}</title>
    <defs>
      <pattern :id="`${drawing.id}-wet`" width="14" height="14" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
        <path d="M0 0V14" class="fp-hatch" />
      </pattern>
      <pattern :id="`${drawing.id}-out`" width="22" height="22" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
        <path d="M0 0V22" class="fp-hatch-out" />
      </pattern>
    </defs>
    <g v-for="layer in drawing.layers" :key="layer.name" :class="`fp-${layer.name}`">
      <component :is="shape.tag" v-for="(shape, i) in layer.shapes" :key="i" v-bind="shape.attrs">{{ shape.text }}</component>
    </g>
  </svg>
</template>

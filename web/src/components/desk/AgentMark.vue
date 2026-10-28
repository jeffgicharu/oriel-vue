<script setup lang="ts">
import { computed } from "vue";
import { dotColour } from "@/lib/desk";

/**
 * Who a lead is with, where there is no room for the name: the agent's
 * initials in a ring of their colour. The colour tells the team apart at a
 * glance, the letters say who it is, and the name is there on hover and for
 * a screen reader.
 */
const props = defineProps<{
  name: string;
  colour: string;
  /** For a row that says the name itself, in its own place. */
  quiet?: boolean;
}>();
const initials = computed(() =>
  props.name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]!.toUpperCase())
    .join(""),
);
</script>

<template>
  <span class="agent-mark" :style="{ borderColor: dotColour(colour) }" :title="name" :aria-hidden="quiet ? 'true' : undefined">
    <span aria-hidden="true">{{ initials }}</span>
    <span v-if="!quiet" class="visually-hidden">With {{ name }}</span>
  </span>
</template>

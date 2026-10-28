<script setup lang="ts">
import { onMounted, ref } from "vue";

/**
 * One of the architectural drawings made from the building's own numbers
 * when the site is built (api: oriel-vue-api drawings). It is our own SVG,
 * never anything a visitor sent, so it goes into the page as it is. On a
 * phone a wide drawing pans, and starts on its most telling part.
 */
const props = withDefaults(defineProps<{ svg: string; label: string; panWidth?: number; pan?: number }>(), { panWidth: 0, pan: 0 });
const frame = ref<HTMLElement | null>(null);

onMounted(() => {
  const el = frame.value;
  if (!el || !props.panWidth) return;
  const room = el.scrollWidth - el.clientWidth;
  if (room > 0) el.scrollLeft = room * props.pan;
});
</script>

<template>
  <div v-if="panWidth" ref="frame" class="content-sheet__pan" :style="`--pan-width: ${panWidth}px`" tabindex="0" role="group" :aria-label="label" v-html="svg" />
  <div v-else class="content-sheet__drawing" v-html="svg" />
</template>

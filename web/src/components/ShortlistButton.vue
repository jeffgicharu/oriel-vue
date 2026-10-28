<script setup lang="ts">
import { computed } from "vue";
import { useShortlistStore } from "@/stores/shortlist";

const props = defineProps<{ id: string }>();
const shortlist = useShortlistStore();
const on = computed(() => shortlist.has(props.id));
</script>

<template>
  <button class="btn btn--line shortlist-toggle" type="button" :aria-pressed="on" @click="shortlist.toggle(id)">
    <svg class="heart" viewBox="0 0 20 18" aria-hidden="true"><path d="M10 16.5S2 11.6 2 6.2A4 4 0 0 1 10 4a4 4 0 0 1 8 2.2c0 5.4-8 10.3-8 10.3Z" /></svg>
    <span>{{ on ? "Shortlisted" : "Shortlist" }}</span>
  </button>
</template>

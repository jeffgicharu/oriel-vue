<script setup lang="ts">
defineProps<{ ok: boolean }>();
</script>

<template>
  <svg class="done-mark" :class="{ 'is-cancelled': !ok }" viewBox="0 0 64 64" aria-hidden="true">
    <rect x="4" y="4" width="56" height="56" pathLength="1" />
    <path v-if="ok" d="M18 33l9 9 19-20" pathLength="1" />
    <path v-else d="M22 22l20 20M42 22L22 42" pathLength="1" />
  </svg>
</template>

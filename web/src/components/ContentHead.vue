<script setup lang="ts">
defineProps<{ tag: string; title: string; lede: string }>();
</script>

<template>
  <header class="page-head content-head">
    <p class="sheet-tag label">{{ tag }}</p>
    <h1 class="h1">{{ title }}</h1>
    <p class="lede">{{ lede }}</p>
  </header>
</template>

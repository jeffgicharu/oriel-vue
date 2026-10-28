<script setup lang="ts">
// A desk page's heading, its lede, and the actions beside them.
defineProps<{ title: string; lede?: string }>();
defineSlots<{ default?(): unknown }>();
</script>

<template>
  <header class="desk-head">
    <div>
      <h1 class="desk-h1">{{ title }}</h1>
      <p v-if="lede" class="desk-lede">{{ lede }}</p>
    </div>
    <div v-if="$slots.default" class="desk-head__actions">
      <slot />
    </div>
  </header>
</template>

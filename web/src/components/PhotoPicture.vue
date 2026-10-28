<script setup lang="ts">
import { computed } from "vue";
import { photo } from "@/lib/photos";

const props = withDefaults(defineProps<{ name: string; alt: string; sizes: string; eager?: boolean }>(), { eager: false });
const src = computed(() => photo(props.name));
</script>

<template>
  <picture v-if="src" class="photo">
    <source type="image/avif" :srcset="src.avif" :sizes="sizes" />
    <img
      :src="src.fallback"
      :srcset="src.webp"
      :sizes="sizes"
      :width="src.width"
      :height="src.height"
      :alt="alt"
      :loading="eager ? 'eager' : 'lazy'"
      :fetchpriority="eager ? 'high' : undefined"
      decoding="async"
    />
  </picture>
</template>

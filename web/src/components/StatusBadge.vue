<script setup lang="ts">
import type { Status } from "@/api/client";
import { statusName } from "@/lib/homes";

defineProps<{ status: Status }>();
</script>

<template>
  <!-- The swatch carries the hatch the drawings use, so the status reads without colour. -->
  <span :class="`status status--${status}`"><span class="status__swatch" aria-hidden="true" />{{ statusName(status) }}</span>
</template>

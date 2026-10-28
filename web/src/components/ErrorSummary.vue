<script setup lang="ts">
import type { ApiError } from "@/api/client";

/**
 * What needs fixing, at the top of a form: each mistake, in the order of the
 * form, linked to its field; or a refusal that is not about any one field,
 * with a way to try again. Focus moves here when the form comes back.
 */
defineProps<{
  id: string;
  listed: { field: string; message: string; href: string }[];
  problem: ApiError | null;
}>();
const emit = defineEmits<{ retry: [] }>();

function jump(event: MouseEvent, href: string) {
  const el = document.querySelector<HTMLElement>(href);
  if (!el) return;
  event.preventDefault();
  el.scrollIntoView({ block: "center" });
  (el.matches("input, select, textarea, button") ? el : el.querySelector<HTMLElement>("input, select, textarea, button"))?.focus({ preventScroll: true });
}
</script>

<template>
  <div v-if="listed.length" :id="`${id}-errors`" class="notice notice--alert error-summary" role="alert" tabindex="-1">
    <strong>Please check the form.</strong>
    <ul class="error-summary__list" role="list">
      <li v-for="e in listed" :key="e.field"><a :href="e.href" @click="jump($event, e.href)">{{ e.message }}</a></li>
    </ul>
  </div>
  <div v-else-if="problem" :id="`${id}-errors`" class="notice notice--alert error-summary" role="alert" tabindex="-1">
    <strong>{{ problem.title }}</strong>
    <p v-if="problem.detail">{{ problem.detail }}</p>
    <p v-if="problem.kind !== 'rate-limited'">Nothing you typed has been lost. <button class="link-button" type="button" @click="emit('retry')">Try again</button></p>
  </div>
</template>

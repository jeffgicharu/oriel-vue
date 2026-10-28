<script setup lang="ts" generic="T">
import type { RequestState } from "@/api/request";

/**
 * Shows a read from the API in each of its states. While a new answer
 * loads, the last one stays on show and is marked busy, so the page does not
 * jump; with nothing to show yet, the loading slot stands in. A failure says
 * what went wrong and offers to try again.
 */
const props = withDefaults(
  defineProps<{
    state: RequestState<T>;
    retry: () => unknown;
    /** Is the answer empty, so the empty slot shows instead? */
    isEmpty?: (data: T) => boolean;
    loadingLabel?: string;
  }>(),
  { isEmpty: () => false, loadingLabel: "Loading…" },
);

defineSlots<{
  default(props: { data: T; busy: boolean }): unknown;
  loading?(): unknown;
  empty?(props: { data: T }): unknown;
  error?(props: { error: NonNullable<RequestState<T>["error"]> }): unknown;
}>();

function tryAgain() {
  void props.retry();
}
</script>

<template>
  <div v-if="state.status === 'error' && state.error" class="request-error notice notice--alert" role="alert">
    <slot name="error" :error="state.error">
      <p class="request-error__title">{{ state.error.title }}</p>
      <p v-if="state.error.detail" class="request-error__detail">{{ state.error.detail }}</p>
      <button class="btn btn--line btn--small" type="button" @click="tryAgain">Try again</button>
    </slot>
  </div>
  <template v-else-if="state.data !== null">
    <slot v-if="isEmpty(state.data)" name="empty" :data="state.data" />
    <slot v-else :data="state.data" :busy="state.status === 'loading'" />
  </template>
  <div v-else class="request-loading" aria-busy="true">
    <slot name="loading">
      <p class="muted">{{ loadingLabel }}</p>
    </slot>
  </div>
</template>

<script setup lang="ts">
// A field people never see; only robots fill it in, and the API then keeps nothing.
const model = defineModel<string>({ required: true });
</script>

<template>
  <div class="visually-hidden" aria-hidden="true">
    <label>Leave this empty <input v-model="model" type="text" name="website" tabindex="-1" autocomplete="off" /></label>
  </div>
</template>

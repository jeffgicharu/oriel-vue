<script setup lang="ts">
/**
 * A labelled input with its mistake or its hint beneath. The mistake is
 * tied to the input for screen readers, and the input is marked invalid.
 */
const props = withDefaults(
  defineProps<{
    id: string;
    label: string;
    type?: string;
    autocomplete?: string;
    hint?: string;
    error?: string;
    required?: boolean;
  }>(),
  { type: "text", autocomplete: "off", hint: "", error: "", required: true },
);
const model = defineModel<string>({ required: true });
const emit = defineEmits<{ touched: [] }>();

function input(event: Event) {
  model.value = (event.target as HTMLInputElement).value;
  if (props.error) emit("touched");
}
</script>

<template>
  <div class="field" :class="{ 'field--error': error }">
    <label class="field__label" :for="id">
      {{ label }}
      <span v-if="!required" class="label">Optional</span>
    </label>
    <input
      :id="id"
      class="input"
      :type="type"
      :value="model"
      :autocomplete="autocomplete"
      :required="required"
      :aria-invalid="error ? 'true' : undefined"
      :aria-describedby="error || hint ? `${id}-note` : undefined"
      @input="input"
    />
    <p v-if="error" :id="`${id}-note`" class="field__error">{{ error }}</p>
    <p v-else-if="hint" :id="`${id}-note`" class="field__hint">{{ hint }}</p>
  </div>
</template>

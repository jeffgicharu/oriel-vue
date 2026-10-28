<script setup lang="ts">
import { computed } from "vue";
import type { Home } from "@/api/client";
import { whatsappHref, whatsappMessage } from "@/lib/whatsapp";
import { useContactStore } from "@/stores/contact";
import { useFrameStore } from "@/stores/frame";

const props = withDefaults(defineProps<{ home?: Pick<Home, "id" | "title" | "price"> | null; label?: string; plain?: boolean }>(), {
  home: null,
  label: "Message us on WhatsApp",
  plain: false,
});

const frame = useFrameStore();
const contact = useContactStore();
const href = computed(() => whatsappHref(whatsappMessage(frame.siteUrl, props.home)));

// The tap is recorded with where the visitor came from, for the sales desk.
// It never holds up WhatsApp opening.
function tapped() {
  contact.tappedWhatsApp(props.home?.id);
}
</script>

<template>
  <a :class="plain ? 'link-arrow whatsapp-link' : 'btn btn--line whatsapp-link'" :href="href" target="_blank" rel="noopener" @click="tapped">
    <svg class="whatsapp-link__icon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 2.5a9.5 9.5 0 0 0-8.2 14.3L2.5 21.5l4.8-1.3A9.5 9.5 0 1 0 12 2.5Z" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" />
      <path d="M8.6 7.6c.2-.4.5-.4.8-.4h.6c.2 0 .4.1.5.4l.8 1.9c.1.3 0 .5-.1.7l-.6.7c-.1.1-.2.3 0 .5.4.7 1 1.4 1.7 1.9.6.5 1.2.8 1.7 1 .2.1.4 0 .5-.1l.7-.9c.2-.2.4-.3.7-.2l1.8.9c.3.1.4.3.4.5 0 .5-.2 1.2-.7 1.6-.5.4-1.2.7-2 .6-1.5-.2-3-1-4.3-2.2-1.4-1.3-2.4-2.8-2.7-4.3-.2-.8.1-1.8.6-2.6Z" fill="currentColor" />
    </svg>
    <span>{{ label }}</span>
    <span class="visually-hidden"> (opens WhatsApp)</span>
  </a>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { navItems } from "@/lib/nav";
import { useShortlistStore } from "@/stores/shortlist";

const shortlist = useShortlistStore();
const route = useRoute();
// The section and the header's style come from the route, which is settled
// before anything renders, so the server and the browser draw the same header.
const nav = computed(() => route.meta.nav ?? "");
const over = computed(() => route.meta.headerOver === true);

const open = ref(false);
const scrolled = ref(false);
const button = ref<HTMLButtonElement | null>(null);
const sheet = ref<HTMLElement | null>(null);

function setOpen(value: boolean) {
  open.value = value;
  document.body.classList.toggle("menu-open", value);
  if (value) requestAnimationFrame(() => sheet.value?.querySelector<HTMLElement>("a")?.focus({ preventScroll: true }));
}

function onKey(event: KeyboardEvent) {
  if (event.key === "Escape" && open.value) {
    setOpen(false);
    button.value?.focus();
  }
}

function onScroll() {
  scrolled.value = window.scrollY > 24;
}

let wide: MediaQueryList | undefined;
function onWide(query: MediaQueryListEvent) {
  if (query.matches) setOpen(false);
}

// Moving to another page closes the menu.
watch(() => route.fullPath, () => open.value && setOpen(false));

onMounted(() => {
  document.addEventListener("keydown", onKey);
  window.addEventListener("scroll", onScroll, { passive: true });
  onScroll();
  wide = window.matchMedia("(min-width: 1024px)");
  wide.addEventListener("change", onWide);
});

onBeforeUnmount(() => {
  document.removeEventListener("keydown", onKey);
  window.removeEventListener("scroll", onScroll);
  wide?.removeEventListener("change", onWide);
});
</script>

<template>
  <header class="site-header" :class="{ 'site-header--over': over, 'is-scrolled': over && scrolled }">
    <div class="wrap site-header__inner">
      <RouterLink class="wordmark" to="/">
        <span class="wordmark__name">ORIEL</span>
        <span class="wordmark__place label">Riverside · Nairobi</span>
        <span class="visually-hidden">, home</span>
      </RouterLink>
      <nav class="site-nav" aria-label="Main">
        <ul role="list">
          <li v-for="item in navItems" :key="item.key">
            <RouterLink :to="item.href" :aria-current="nav === item.key ? 'page' : undefined">{{ item.label }}</RouterLink>
          </li>
        </ul>
      </nav>
      <RouterLink
        :hidden="shortlist.count === 0"
        class="site-header__shortlist"
        :to="{ name: 'shortlist' }"
      >
        <svg class="heart" viewBox="0 0 20 18" aria-hidden="true"><path d="M10 16.5S2 11.6 2 6.2A4 4 0 0 1 10 4a4 4 0 0 1 8 2.2c0 5.4-8 10.3-8 10.3Z" /></svg>
        <span class="visually-hidden">Shortlist,</span>
        <span class="tnum">{{ shortlist.count }}</span>
      </RouterLink>
      <RouterLink class="btn btn--cobalt btn--small site-header__cta" :to="{ name: 'viewing-new' }">Book a viewing</RouterLink>
      <button ref="button" class="menu-button" type="button" :aria-expanded="open" aria-controls="menu-sheet" @click="setOpen(!open)">
        <span>{{ open ? "Close" : "Menu" }}</span>
        <span class="menu-button__lines" aria-hidden="true" />
      </button>
    </div>
  </header>
  <div id="menu-sheet" ref="sheet" class="menu-sheet" :inert="!open">
    <nav aria-label="Main">
      <ol role="list">
        <li v-for="item in navItems" :key="item.key">
          <RouterLink :to="item.href" :aria-current="nav === item.key ? 'page' : undefined">
            <span>{{ item.label }}</span>
            <span class="label">{{ item.sheet }}</span>
          </RouterLink>
        </li>
      </ol>
    </nav>
    <div class="menu-sheet__foot">
      <RouterLink class="btn btn--cobalt btn--wide" :to="{ name: 'viewing-new' }">Book a viewing</RouterLink>
      <RouterLink class="btn btn--line btn--wide" :to="{ name: 'enquire' }">Ask a question</RouterLink>
      <p class="label muted">Sales suite open Tuesday to Sunday, 10:00 to 17:00</p>
    </div>
  </div>
</template>

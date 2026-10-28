<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, type RouteLocationRaw } from "vue-router";
import RequestView from "@/components/RequestView.vue";
import DeskHeader from "@/components/desk/DeskHeader.vue";
import { useDeskHead } from "@/composables/useDeskHead";
import { useHttpStatus } from "@/composables/useHttpStatus";
import { useLiveEvents, type LiveEvent } from "@/composables/useLiveEvents";
import { dotColour } from "@/lib/desk";
import { useDeskStore } from "@/stores/desk";
import "@/styles/desk.css";

/**
 * The frame of every page of the sales desk: the menu, who is signed in, the
 * title bar on smaller screens, what the last change did, and the messages
 * that arrive live. Each page wraps its content in it.
 */
const props = defineProps<{
  /** The page's own title, for the tab and the title bar. */
  title: string;
  /** The menu item the page belongs to. */
  nav: string;
  /** Brings the page's data up to date after a change made elsewhere. */
  reload?: (event: LiveEvent) => unknown;
}>();
defineSlots<{ default(): unknown }>();

const route = useRoute();
const desk = useDeskStore();
const setStatus = useHttpStatus();

// A member who is not the manager, on a page for the manager alone, is told
// so here rather than sent away, as Oriel did.
const forbidden = computed(() => route.meta.manager === true && desk.session.data !== null && !desk.manager);
if (forbidden.value) setStatus(403);
useDeskHead(() => (forbidden.value ? "Managers only" : props.title));

interface NavItem {
  key: string;
  label: string;
  to: RouteLocationRaw;
  managerOnly?: boolean;
}

const items: NavItem[] = [
  { key: "dashboard", label: "Dashboard", to: { name: "desk" } },
  { key: "stack", label: "Stacking plan", to: { name: "desk-stack" } },
  { key: "leads", label: "Leads", to: { name: "desk-leads" } },
  { key: "viewings", label: "Viewings", to: { name: "desk-viewings" } },
  { key: "reservations", label: "Reservations", to: { name: "desk-reservations" } },
  { key: "pricing", label: "Pricing", to: { name: "desk-pricing" }, managerOnly: true },
  { key: "activity", label: "Activity", to: { name: "desk-activity" } },
];
const shown = computed(() => items.filter((item) => !item.managerOnly || desk.manager));

// The menu slides in on smaller screens.
const menuOpen = ref(false);
const side = ref<HTMLElement | null>(null);
const menuButton = ref<HTMLButtonElement | null>(null);
watch(() => route.fullPath, () => (menuOpen.value = false));

function onKey(e: KeyboardEvent) {
  if (e.key === "Escape" && menuOpen.value) {
    menuOpen.value = false;
    menuButton.value?.focus();
  }
}

function onClick(e: MouseEvent) {
  const target = e.target as Node | null;
  if (menuOpen.value && target && !side.value?.contains(target) && !menuButton.value?.contains(target)) menuOpen.value = false;
}

// Messages that arrive while the page is open, a few at a time.
interface Toast {
  id: number;
  text: string;
  to?: RouteLocationRaw;
  label?: string;
}
const toasts = ref<Toast[]>([]);
let toastId = 0;

function toast(text: string, to?: RouteLocationRaw, label?: string) {
  const id = ++toastId;
  toasts.value = [...toasts.value, { id, text, to, label }].slice(-3);
  window.setTimeout(() => (toasts.value = toasts.value.filter((t) => t.id !== id)), 9000);
}

useLiveEvents("/api/desk/events", (event) => {
  // The badge counts the requests waiting, which any decision changes.
  if (event.type === "reservation") void desk.loadSession(true);
  if (event.text) {
    if (event.type === "reservation") toast(event.text, { name: "desk-reservations" }, "Review it");
    else if (event.type === "lead") toast(event.text, { name: "desk-leads" }, "Leads");
    else toast(event.text);
  }
  void props.reload?.(event);
});

// The desk's clock moves on each minute once the page is running.
let clock: number | undefined;
onMounted(() => {
  desk.tick();
  clock = window.setInterval(() => desk.tick(), 60_000);
  document.addEventListener("keydown", onKey);
  document.addEventListener("click", onClick);
});
onBeforeUnmount(() => {
  window.clearInterval(clock);
  document.removeEventListener("keydown", onKey);
  document.removeEventListener("click", onClick);
});

async function signOut() {
  try {
    await desk.signOut();
  } catch {
    toast("You could not be signed out just now. Please try again.");
    return;
  }
  // A whole new page, so nothing the desk held stays in memory.
  window.location.assign("/desk/sign-in");
}
</script>

<template>
  <a class="skip-link" href="#main">Skip to content</a>
  <div class="desk-shell">
    <aside id="desk-side" ref="side" class="desk-side" :class="{ 'is-open': menuOpen }">
      <RouterLink class="desk-mark" :to="{ name: 'desk' }">
        <span class="desk-mark__name">ORIEL</span>
        <span class="desk-mark__sub">Sales desk</span>
      </RouterLink>
      <nav class="desk-nav" aria-label="Desk">
        <ul role="list">
          <li v-for="item in shown" :key="item.key">
            <RouterLink :to="item.to" :aria-current="nav === item.key ? 'page' : undefined">
              <span>{{ item.label }}</span>
              <span v-if="item.key === 'reservations'" class="desk-nav__badge" :class="{ 'is-empty': desk.waiting === 0 }"
                >{{ desk.waiting }}<span class="visually-hidden"> waiting</span></span
              >
            </RouterLink>
          </li>
        </ul>
      </nav>
      <div class="desk-side__foot">
        <div v-if="desk.staff" class="desk-me">
          <span class="desk-me__dot" :style="{ backgroundColor: dotColour(desk.staff.colour) }" aria-hidden="true">{{ desk.staff.initials }}</span>
          <span>
            <span class="desk-me__name">{{ desk.staff.name }}</span>
            <span class="desk-me__role">{{ desk.staff.title }}</span>
          </span>
        </div>
        <a class="desk-side__link" href="/" target="_blank" rel="noopener">The website ↗</a>
        <button class="desk-side__link" type="button" @click="signOut">Sign out</button>
      </div>
    </aside>
    <div class="desk-main">
      <header class="desk-bar">
        <button ref="menuButton" class="desk-bar__menu" type="button" aria-controls="desk-side" :aria-expanded="menuOpen ? 'true' : 'false'" @click="menuOpen = !menuOpen">
          Menu
        </button>
        <span class="desk-bar__title">{{ forbidden ? "Managers only" : title }}</span>
      </header>
      <main id="main" class="desk-content" tabindex="-1">
        <div v-if="desk.notice" class="desk-notice">{{ desk.notice }}</div>
        <!-- Always there, so a screen reader hears each notice as it arrives. -->
        <p class="visually-hidden" role="status">{{ desk.notice }}</p>
        <RequestView v-if="desk.session.data === null" :state="desk.session" :retry="() => desk.loadSession(true)" loading-label="Loading the desk…">
          <template #default />
        </RequestView>
        <template v-else-if="forbidden">
          <DeskHeader title="For the sales manager" lede="Prices are set by the sales manager. You can see every price on the stacking plan." />
          <p><RouterLink class="btn btn--line btn--small" :to="{ name: 'desk' }">Back to the dashboard</RouterLink></p>
        </template>
        <slot v-else />
      </main>
    </div>
  </div>
  <div class="desk-toasts" aria-live="polite">
    <div v-for="t in toasts" :key="t.id" class="desk-toast">
      <span>{{ t.text }}</span>
      <RouterLink v-if="t.to" :to="t.to">{{ t.label || "Open" }}</RouterLink>
    </div>
  </div>
</template>

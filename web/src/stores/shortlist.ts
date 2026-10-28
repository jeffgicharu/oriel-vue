// The shortlist lives in this browser only: a convenience, not a record. It
// is empty while a page renders on the server and fills from storage once the
// page is running in the browser.

import { defineStore } from "pinia";
import { computed, ref } from "vue";

const KEY = "oriel.shortlist";
export const SHORTLIST_MAX = 12;

function read(): string[] {
  try {
    const raw = localStorage.getItem(KEY);
    const list = raw ? (JSON.parse(raw) as unknown) : [];
    return Array.isArray(list) ? list.filter((x): x is string => typeof x === "string" && /^\d{4}$/.test(x)).slice(0, SHORTLIST_MAX) : [];
  } catch {
    return [];
  }
}

export const useShortlistStore = defineStore("shortlist", () => {
  const ids = ref<string[]>([]);
  const ready = ref(false);
  const count = computed(() => ids.value.length);

  function save() {
    try {
      localStorage.setItem(KEY, JSON.stringify(ids.value));
    } catch {
      // Storage may be unavailable, as in some private windows; the list
      // still works until the page is left.
    }
  }

  /** Reads the list from storage, and follows changes made in other tabs. */
  function restore() {
    if (ready.value) return;
    ids.value = read();
    ready.value = true;
    window.addEventListener("storage", (e) => {
      if (e.key === KEY) ids.value = read();
    });
  }

  function has(id: string) {
    return ids.value.includes(id);
  }

  function toggle(id: string) {
    ids.value = has(id) ? ids.value.filter((x) => x !== id) : [id, ...ids.value.filter((x) => x !== id)].slice(0, SHORTLIST_MAX);
    save();
  }

  function remove(id: string) {
    ids.value = ids.value.filter((x) => x !== id);
    save();
  }

  /** Up to two more homes from the shortlist to compare a home with. */
  function compareWith(id: string): string[] {
    return [id, ...ids.value.filter((x) => x !== id).slice(0, 2)];
  }

  return { ids, ready, count, restore, has, toggle, remove, compareWith };
});

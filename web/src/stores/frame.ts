// What the site's frame needs to know: the public address, today's date in
// Nairobi and the release, set by the server when it renders a page and
// carried to the browser with the rest of the state; and, for the page on
// show, which section it belongs to and its drawing-sheet number.

import { defineStore } from "pinia";
import { reactive, ref } from "vue";

export interface Sheet {
  number: string;
  name: string;
}

export const useFrameStore = defineStore("frame", () => {
  const siteUrl = ref("");
  const today = ref("");
  const release = ref("");
  const page = reactive({
    nav: "",
    sheet: { number: "A-000", name: "General arrangement" } as Sheet,
    // The header lies over the page's opening picture.
    headerOver: false,
    // The footer leaves out its invitation to book, where a visitor is
    // already booking or has just booked.
    quiet: false,
  });

  return { siteUrl, today, release, page };
});

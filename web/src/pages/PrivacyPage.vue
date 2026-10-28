<script setup lang="ts">
import { ref } from "vue";
import ContentHead from "@/components/ContentHead.vue";
import { isApiError } from "@/api/request";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { useContactStore } from "@/stores/contact";
import { useSiteStore } from "@/stores/site";

const site = useSiteStore();
const contact = useContactStore();
usePageData(() => site.loadPrivacy(true));

usePageMeta({
  title: "Privacy",
  description: "What the Oriel website keeps about you, where, and for how long, and how to delete what you have sent.",
  path: "/privacy",
  sheet: { number: "A-900", name: "Privacy" },
});

const deleting = ref(false);
const deleted = ref(false);
const failed = ref("");

async function deleteMine() {
  if (!window.confirm("Delete everything you have sent from this browser? Any viewing booked from it is cancelled.")) return;
  deleting.value = true;
  failed.value = "";
  try {
    await contact.deleteMine();
    deleted.value = true;
    await site.loadPrivacy(true);
  } catch (e) {
    failed.value = isApiError(e) ? e.detail || e.title : "That could not be done just now. Please try again.";
  } finally {
    deleting.value = false;
  }
}
</script>

<template>
  <div class="wrap content-page">
    <ContentHead tag="A-900 · Privacy" title="Your privacy." lede="What this website keeps about you, where, and for how long. In short: very little, nowhere else, and not for long." />
    <div class="prose legal">
      <h2 class="h3">What you tell us</h2>
      <p>
        When you book a viewing, request a reservation or ask a question, the details you type (your name, email address, phone number and anything you add) are saved so the booking and
        the sales desk can show them.
      </p>
      <p>
        They are saved in a private workspace that belongs to your browser alone. Nobody visiting from another browser can see it. It is deleted automatically when it has not been used for 24
        hours, and nothing in it is copied anywhere else, sold or used for marketing.
      </p>
      <h2 class="h3">Where you came from</h2>
      <p>
        So the sales team knows which of its campaigns work, this browser keeps a note of how you first reached the site and how you reached it this time: the campaign named in the link you
        followed, if any, the site that sent you, and the first page you saw. The note stays in your browser and is sent only with a form you send, or when you open WhatsApp from a home’s
        page. It holds nothing that identifies you, and it is forgotten after 90 days.
      </p>
      <h2 class="h3">Cookies and storage</h2>
      <p>
        The site sets one cookie, <span class="mono">oriel_copy</span>, which links your browser to its private workspace. It is set only once you send something, and it lasts seven days at
        most. The sales desk sets one more cookie while a member of staff is signed in.
      </p>
      <p>The homes you shortlist are kept in your browser’s own storage and never leave your device.</p>
      <p>
        There are no advertising or tracking cookies. The fonts, pictures and code are all served from this site, with one exception: Cloudflare, which delivers the site, adds a small
        script that counts visits and measures how fast pages load. It sets no cookies and does not follow you to other sites.
      </p>
      <h2 class="h3">Server records</h2>
      <p>
        Like every website, the server and Cloudflare keep short technical logs of requests (the address asked for, the time and your IP address) to keep the site running and safe. The
        server’s are kept for about two weeks.
      </p>
      <h2 class="h3">Your rights</h2>
      <p>Under Kenya’s Data Protection Act, 2019, you can ask what we hold about you, have it corrected or have it deleted. Everything you type is deleted within a day of your last visit, or straight away below.</p>
      <h2 id="yours" class="h3">Delete what you have sent</h2>
      <div v-if="deleted" class="notice" role="status">Everything you sent from this browser has been deleted.</div>
      <template v-else-if="site.privacy.data?.hasCopy">
        <p>This browser has a private workspace holding the bookings, reservations and questions you sent. You can delete it now instead of waiting a day. Any viewing or reservation in it goes with it.</p>
        <button class="btn btn--line" type="button" :disabled="deleting" @click="deleteMine">{{ deleting ? "Deleting…" : "Delete what I have sent" }}</button>
        <p v-if="failed" class="field__error" role="alert">{{ failed }}</p>
      </template>
      <p v-else-if="site.privacy.data">Nothing has been sent from this browser, so there is nothing to delete.</p>
      <p class="muted">Last updated 28 October 2026.</p>
    </div>
  </div>
</template>

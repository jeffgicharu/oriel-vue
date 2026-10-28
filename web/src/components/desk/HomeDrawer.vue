<script setup lang="ts">
import { computed, ref } from "vue";
import ErrorSummary from "@/components/ErrorSummary.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import type { DeskHomeDetail } from "@/api/desk";
import type { Status } from "@/api/client";
import { useForm } from "@/composables/useForm";
import { parseShillings, when } from "@/lib/desk";
import { kes, levelName, thousands } from "@/lib/format";
import { statusName } from "@/lib/homes";
import { useDeskHomesStore } from "@/stores/deskHomes";

/**
 * A home's drawer on the stacking plan: its status and price, its
 * reservation, its history, and for the sales manager the forms that
 * withhold or release it and give it a price of its own.
 */
const props = defineProps<{ detail: DeskHomeDetail; now: string }>();
const emit = defineEmits<{ close: [] }>();
const homes = useDeskHomesStore();
const home = computed(() => props.detail.home);
// What the last change here did, said inside the drawer, as Oriel did.
const done = ref("");

const withholding = computed(() => home.value.status === "available");

const statusForm = useForm({
  id: "home-status",
  values: { reason: "" },
  order: ["reason"],
  check: (v): Record<string, string> => (v.reason.trim() ? {} : { reason: "Please give a reason." }),
  send: async (v) => {
    done.value = "";
    const changed = await homes.setStatus(home.value.id, withholding.value ? "withheld" : "available", v.reason.trim());
    done.value = changed.notice;
    v.reason = "";
    return changed;
  },
});

const priceForm = useForm({
  id: "home-price",
  values: { price: home.value.overridden ? thousands(home.value.price) : "", reason: "", clear: false },
  order: ["price", "reason"],
  check: (v): Record<string, string> => {
    const errs: Record<string, string> = {};
    const price = parseShillings(v.price);
    if (!v.clear && (price === null || price <= 0)) errs.price = "Please give the price in shillings, such as 18,480,000.";
    if (!v.reason.trim()) errs.reason = "Please give a reason.";
    return errs;
  },
  send: async (v) => {
    done.value = "";
    const changed = await homes.setPrice(home.value.id, v.clear ? 0 : (parseShillings(v.price) ?? 0), v.reason.trim());
    done.value = changed.notice;
    v.reason = "";
    v.price = changed.home.overridden ? thousands(changed.home.price) : "";
    return changed;
  },
});

function historyLine(e: DeskHomeDetail["history"][number]): string {
  return e.kind === "status" ? `${statusName(e.from as Status)} → ${statusName(e.to as Status)}` : `${e.from} → ${e.to}`;
}
</script>

<template>
  <div class="desk-home" :data-home="home.id">
    <button class="desk-drawer__close" type="button" aria-label="Close" @click="emit('close')">×</button>
    <p class="label">{{ levelName(home.level) }} · Type {{ home.type }}</p>
    <h2 class="desk-h1">Home {{ home.id }}</h2>
    <p class="desk-lede">{{ home.title }} · faces {{ home.facings }} · {{ home.internal }} m²</p>
    <div v-if="done" class="desk-notice" role="status">{{ done }}</div>
    <dl class="desk-facts">
      <div>
        <dt class="label">Status</dt>
        <dd><StatusBadge :status="home.status" /></dd>
      </div>
      <div>
        <dt class="label">Price</dt>
        <dd class="tnum">
          {{ kes(home.price) }}
          <span v-if="home.overridden" class="desk-small">its own price; list {{ kes(home.listPrice) }}</span>
        </dd>
      </div>
      <div v-if="detail.reservation">
        <dt class="label">Reservation</dt>
        <dd>
          <RouterLink :to="{ name: 'desk-reservations', query: { tab: detail.reservation.group }, hash: `#${detail.reservation.ref}` }">{{ detail.reservation.ref }}</RouterLink>
          · {{ detail.reservation.name }} · {{ detail.reservation.statusName }}
        </dd>
      </div>
    </dl>
    <p class="desk-home__links">
      <a class="desk-link" :href="`/residences/${home.id}`" target="_blank" rel="noopener">Its page on the website ↗</a>
    </p>
    <template v-if="detail.canChange">
      <form v-if="home.status === 'available' || home.status === 'withheld'" class="desk-form desk-home__form" novalidate @submit.prevent="statusForm.submit">
        <h3 class="desk-h2">{{ withholding ? "Withhold from sale" : "Release for sale" }}</h3>
        <ErrorSummary id="home-status" :listed="statusForm.listed.value" :problem="statusForm.problem.value" @retry="statusForm.submit" />
        <div class="field" :class="{ 'field--error': statusForm.errors.value.reason }">
          <label class="field__label" for="home-status-reason">Reason</label>
          <input
            id="home-status-reason"
            v-model="statusForm.values.reason"
            class="input"
            type="text"
            maxlength="200"
            required
            :aria-invalid="statusForm.errors.value.reason ? 'true' : undefined"
            :aria-describedby="statusForm.errors.value.reason ? 'home-status-reason-note' : undefined"
            @input="statusForm.touched('reason')"
          />
          <p v-if="statusForm.errors.value.reason" id="home-status-reason-note" class="field__error">{{ statusForm.errors.value.reason }}</p>
        </div>
        <button class="btn btn--small" type="submit" :disabled="statusForm.status.value === 'sending'">{{ withholding ? "Withhold it" : "Release it" }}</button>
      </form>
      <form v-if="home.status !== 'sold' && home.status !== 'reserved'" class="desk-form desk-home__form" novalidate @submit.prevent="priceForm.submit">
        <h3 class="desk-h2">Its own price</h3>
        <ErrorSummary id="home-price" :listed="priceForm.listed.value" :problem="priceForm.problem.value" @retry="priceForm.submit" />
        <div class="desk-form__row">
          <div class="field" :class="{ 'field--error': priceForm.errors.value.price }">
            <label class="field__label" for="home-price-price">Price in KES</label>
            <input
              id="home-price-price"
              v-model="priceForm.values.price"
              class="input tnum"
              type="text"
              inputmode="numeric"
              :placeholder="thousands(home.listPrice)"
              :aria-invalid="priceForm.errors.value.price ? 'true' : undefined"
              :aria-describedby="priceForm.errors.value.price ? 'home-price-price-note' : undefined"
              @input="priceForm.touched('price')"
            />
            <p v-if="priceForm.errors.value.price" id="home-price-price-note" class="field__error">{{ priceForm.errors.value.price }}</p>
          </div>
        </div>
        <div class="field" :class="{ 'field--error': priceForm.errors.value.reason }">
          <label class="field__label" for="home-price-reason">Reason</label>
          <input
            id="home-price-reason"
            v-model="priceForm.values.reason"
            class="input"
            type="text"
            maxlength="200"
            required
            :aria-invalid="priceForm.errors.value.reason ? 'true' : undefined"
            :aria-describedby="priceForm.errors.value.reason ? 'home-price-reason-note' : undefined"
            @input="priceForm.touched('reason')"
          />
          <p v-if="priceForm.errors.value.reason" id="home-price-reason-note" class="field__error">{{ priceForm.errors.value.reason }}</p>
        </div>
        <div class="desk-form__row">
          <button class="btn btn--small" type="submit" :disabled="priceForm.status.value === 'sending'" @click="priceForm.values.clear = false">Set the price</button>
          <button v-if="home.overridden" class="btn btn--small btn--line" type="submit" :disabled="priceForm.status.value === 'sending'" @click="priceForm.values.clear = true">
            Back to the list price
          </button>
        </div>
      </form>
    </template>
    <p v-else class="desk-small">Only the sales manager can change a home’s status or price.</p>
    <section class="desk-home__history">
      <h3 class="desk-h2">History</h3>
      <p v-if="detail.history.length === 0" class="desk-empty">No changes yet.</p>
      <ol v-else class="desk-timeline" role="list">
        <li v-for="(e, i) in detail.history" :key="i">
          <span class="desk-timeline__when mono">{{ when(e.at, now) }}</span>
          <span class="desk-timeline__body">
            {{ historyLine(e) }}
            <span v-if="e.reason" class="desk-timeline__sub">{{ e.reason }}</span>
            <span v-if="e.staff" class="desk-timeline__sub">{{ e.staff }}</span>
          </span>
        </li>
      </ol>
    </section>
  </div>
</template>

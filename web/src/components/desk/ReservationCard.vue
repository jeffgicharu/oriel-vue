<script setup lang="ts">
import { computed } from "vue";
import ErrorSummary from "@/components/ErrorSummary.vue";
import type { DeskReservation, DeskReservationResult } from "@/api/desk";
import { useForm } from "@/composables/useForm";
import { lapses, parseShillings, when } from "@/lib/desk";
import { kes, thousands } from "@/lib/format";
import { useDeskHomesStore } from "@/stores/deskHomes";

/**
 * One reservation: the home and the buyer, where it stands, and for the sales
 * manager the decisions on it: confirm it and issue the letter of offer,
 * record the deposit, or decline it with a reason the buyer will see.
 */
const props = defineProps<{ r: DeskReservation; manager: boolean; now: string; target: boolean }>();
const emit = defineEmits<{ decided: [result: DeskReservationResult] }>();
const homes = useDeskHomesStore();
const code = computed(() => props.r.ref);
const open = computed(() => props.r.status === "requested" || props.r.status === "confirmed");

function decided(result: DeskReservationResult) {
  emit("decided", result);
  return result;
}

const confirmForm = useForm({
  id: `confirm-${props.r.ref}`,
  values: {},
  order: [],
  check: () => ({}),
  send: async () => decided(await homes.confirm(code.value)),
});

function confirmIt() {
  if (window.confirm(`Confirm reservation ${props.r.ref} and issue the letter of offer for home ${props.r.home}?`)) void confirmForm.submit();
}

const deposit = useForm({
  id: `deposit-${props.r.ref}`,
  values: { amount: thousands(props.r.suggestedDeposit) },
  order: ["amount"],
  check: (v): Record<string, string> => {
    const n = parseShillings(v.amount);
    return n !== null && n > 0 ? {} : { amount: "Please give the deposit in shillings, such as 1,848,000." };
  },
  send: async (v) => decided(await homes.recordDeposit(code.value, parseShillings(v.amount) ?? 0)),
});

const decline = useForm({
  id: `decline-${props.r.ref}`,
  values: { reason: "" },
  order: ["reason"],
  check: (v): Record<string, string> => {
    const reason = v.reason.trim();
    if ([...reason].length < 3) return { reason: "Please give the buyer a reason." };
    if ([...reason].length > 200) return { reason: "Please keep this under 200 characters." };
    return {};
  },
  send: async (v) => decided(await homes.decline(code.value, v.reason.trim())),
});

const refused = computed(() => confirmForm.problem.value !== null || deposit.listed.value.length > 0 || deposit.problem.value !== null);
</script>

<template>
  <li :id="r.ref" class="res-card" :class="{ 'is-target': target }">
    <div class="res-card__home">
      <RouterLink class="res-card__id mono" :to="{ name: 'desk-stack', query: { home: r.home } }" :aria-label="`Home ${r.home} on the stacking plan`">{{ r.home }}</RouterLink>
      <span class="res-card__ref mono">{{ r.ref }}</span>
    </div>
    <div class="res-card__main">
      <p class="res-card__buyer">
        <RouterLink :to="{ name: 'desk-lead', params: { id: String(r.leadId) } }">{{ r.name }}</RouterLink>
      </p>
      <p class="res-card__meta">{{ r.planName }} · <span class="tnum">{{ kes(r.price) }}</span> · {{ r.phone }}</p>
      <p v-if="r.note" class="res-card__note">“{{ r.note }}”</p>
      <p class="res-card__when">Requested {{ when(r.requested, now) }}<template v-if="r.agent"> · {{ r.agent }}</template></p>
    </div>
    <div class="res-card__state">
      <template v-if="r.status === 'requested'">
        <span class="res-card__hold" :class="{ 'is-soon': r.holdSecondsLeft < 12 * 3600 }">{{ lapses(r.holdSecondsLeft) }}</span>
      </template>
      <template v-else-if="r.status === 'confirmed'">
        <span class="res-card__offer">Offer <span class="mono">{{ r.offerNumber }}</span></span>
        <span class="desk-small">confirmed {{ when(r.decided, now) }} by {{ r.decidedBy }}</span>
        <a v-if="r.offer" class="desk-link" :href="r.offer" target="_blank" rel="noopener">Letter of offer (PDF)</a>
      </template>
      <template v-else-if="r.status === 'completed'">
        <span class="res-card__offer">Deposit <span class="tnum">{{ kes(r.depositAmount) }}</span></span>
        <span class="desk-small">received {{ when(r.deposit, now) }}</span>
        <a v-if="r.offer" class="desk-link" :href="r.offer" target="_blank" rel="noopener">Letter of offer (PDF)</a>
      </template>
      <template v-else>
        <span class="res-card__closed">{{ r.statusName }}</span>
        <span v-if="r.declineReason" class="desk-small">{{ r.declineReason }}</span>
      </template>
    </div>
    <div v-if="refused" class="res-card__problem">
      <ErrorSummary :id="`confirm-${r.ref}`" :listed="confirmForm.listed.value" :problem="confirmForm.problem.value" @retry="confirmIt" />
      <ErrorSummary :id="`deposit-${r.ref}`" :listed="deposit.listed.value" :problem="deposit.problem.value" @retry="deposit.submit" />
    </div>
    <div v-if="manager && open" class="res-card__actions">
      <form v-if="r.status === 'requested'" novalidate @submit.prevent="confirmIt">
        <button class="btn btn--cobalt btn--small" type="submit" :disabled="confirmForm.status.value === 'sending'">Confirm and issue the offer</button>
      </form>
      <form v-else class="res-card__deposit" novalidate @submit.prevent="deposit.submit">
        <label class="visually-hidden" :for="`deposit-${r.ref}-amount`">Deposit received, in KES</label>
        <input
          :id="`deposit-${r.ref}-amount`"
          v-model="deposit.values.amount"
          class="input tnum"
          name="amount"
          type="text"
          inputmode="numeric"
          :aria-invalid="deposit.errors.value.amount ? 'true' : undefined"
          :aria-describedby="deposit.errors.value.amount ? `deposit-${r.ref}-amount-note` : undefined"
          @input="deposit.touched('amount')"
        />
        <button class="btn btn--cobalt btn--small" type="submit" :disabled="deposit.status.value === 'sending'">Record the deposit</button>
        <p v-if="deposit.errors.value.amount" :id="`deposit-${r.ref}-amount-note`" class="field__error">{{ deposit.errors.value.amount }}</p>
      </form>
      <details class="res-card__decline">
        <summary>Decline</summary>
        <form novalidate @submit.prevent="decline.submit">
          <ErrorSummary :id="`decline-${r.ref}`" :listed="decline.listed.value" :problem="decline.problem.value" @retry="decline.submit" />
          <label class="field__label" :for="`decline-${r.ref}-reason`">Reason, which the buyer will see</label>
          <input
            :id="`decline-${r.ref}-reason`"
            v-model="decline.values.reason"
            class="input"
            name="reason"
            type="text"
            maxlength="200"
            required
            :aria-invalid="decline.errors.value.reason ? 'true' : undefined"
            :aria-describedby="decline.errors.value.reason ? `decline-${r.ref}-reason-note` : undefined"
            @input="decline.touched('reason')"
          />
          <p v-if="decline.errors.value.reason" :id="`decline-${r.ref}-reason-note`" class="field__error">{{ decline.errors.value.reason }}</p>
          <button class="btn btn--small btn--danger" type="submit" :disabled="decline.status.value === 'sending'">Decline and free the home</button>
        </form>
      </details>
    </div>
    <p v-else-if="!manager && open" class="desk-small res-card__actions">The sales manager confirms reservations and records deposits.</p>
  </li>
</template>

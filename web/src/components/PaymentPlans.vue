<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { Schedule } from "@/api/client";
import { isApiError } from "@/api/request";
import { kes, monthYear, percent, shortDate } from "@/lib/format";
import { useHomesStore } from "@/stores/homes";

/**
 * The three ways to pay for a home, and the schedule each gives. The plan
 * chosen is kept in the address, so a buyer can send the schedule on. On the
 * mortgage plan the buyer can try their own rate and term; each change asks
 * the API, which works the figures out the same way the letter of offer will.
 */
const props = defineProps<{ homeId: string; schedules: Schedule[] }>();
const homes = useHomesStore();
const route = useRoute();
const router = useRouter();

type Code = Schedule["plan"]["code"];
const chosen = computed<Code>(() => {
  const q = String(route.query.plan ?? "");
  return (props.schedules.some((s) => s.plan.code === q) ? q : "staged") as Code;
});

// The buyer's own mortgage estimate, once they change the rate or term.
const ownMortgage = ref<Schedule | null>(null);
const estimating = ref(false);
const estimateError = ref("");
watch(() => props.homeId, () => (ownMortgage.value = null));

const shown = computed(() => {
  if (chosen.value === "mortgage" && ownMortgage.value) return ownMortgage.value;
  return props.schedules.find((s) => s.plan.code === chosen.value) ?? props.schedules[0]!;
});
const total = computed(() => shown.value.instalments.reduce((sum, i) => sum + i.amount, 0));

function choose(code: Code) {
  void router.replace({ query: { ...route.query, plan: code === "staged" ? undefined : code }, hash: route.hash });
}

const rates = [1050, 1150, 1250, 1350, 1450, 1550, 1650];
const terms = [10, 15, 20, 25];

async function estimate(event: Event) {
  const form = (event.target as HTMLElement).closest("form")!;
  const data = new FormData(form);
  estimating.value = true;
  estimateError.value = "";
  try {
    ownMortgage.value = await homes.schedule(props.homeId, "mortgage", Number(data.get("rate")), Number(data.get("years")));
  } catch (e) {
    estimateError.value = isApiError(e) ? e.detail || e.title : "The estimate could not be worked out. Please try again.";
  } finally {
    estimating.value = false;
  }
}
</script>

<template>
  <form class="payplans" method="get" @submit.prevent>
    <fieldset class="payplans__choices">
      <legend class="visually-hidden">Payment plan</legend>
      <label v-for="s in schedules" :key="s.plan.code" class="choice payplan">
        <input type="radio" name="plan" :value="s.plan.code" :checked="s.plan.code === chosen" @change="choose(s.plan.code)" />
        <span>
          <strong>{{ s.plan.name }}</strong>
          <span class="muted">{{ s.plan.summary }}</span>
        </span>
      </label>
    </fieldset>
    <div id="schedule" class="schedule-card" :aria-busy="estimating">
      <div class="schedule-card__head">
        <div>
          <span class="label">{{ shown.plan.name }}</span>
          <p class="schedule-card__total tnum">{{ kes(shown.price) }}</p>
          <p v-if="shown.discount > 0" class="schedule-card__saving">{{ kes(shown.discount) }} off the list price of {{ kes(shown.listPrice) }}</p>
        </div>
      </div>
      <table class="pay-table">
        <caption class="visually-hidden">Payments on the {{ shown.plan.name }} plan</caption>
        <thead>
          <tr><th scope="col">Payment</th><th scope="col">Due</th><th scope="col" class="num">Share</th><th scope="col" class="num">Amount</th></tr>
        </thead>
        <tbody>
          <tr v-for="(i, n) in shown.instalments" :key="n">
            <th scope="row">
              {{ i.label }}
              <span v-if="i.note" class="pay-table__note">{{ i.note }}</span>
            </th>
            <td class="mono">{{ i.due ? shortDate(i.due) : "" }}</td>
            <td class="num mono">{{ percent(i.percentBP) }}</td>
            <td class="num tnum">{{ kes(i.amount) }}</td>
          </tr>
        </tbody>
        <tfoot>
          <tr><th scope="row" colspan="3">Total</th><td class="num tnum">{{ kes(total) }}</td></tr>
        </tfoot>
      </table>
      <div v-if="shown.mortgage" class="mortgage">
        <div class="mortgage__inputs" @change="estimate">
          <label class="field">
            <span class="field__label">Interest rate</span>
            <select class="select select--small" name="rate">
              <option v-for="r in rates" :key="r" :value="r" :selected="r === shown.mortgage.rateBP">{{ percent(r) }} a year</option>
            </select>
          </label>
          <label class="field">
            <span class="field__label">Term</span>
            <select class="select select--small" name="years">
              <option v-for="y in terms" :key="y" :value="y" :selected="y === shown.mortgage.years">{{ y }} years</option>
            </select>
          </label>
        </div>
        <p v-if="estimateError" class="field__error" role="alert">{{ estimateError }}</p>
        <p class="mortgage__result" aria-live="polite">
          <span class="label">Estimated repayment</span>
          <span class="mortgage__monthly tnum">{{ kes(shown.mortgage.monthly) }}</span>
          <span class="muted"
            >a month on a loan of {{ kes(shown.mortgage.loan) }}, from {{ monthYear(shown.mortgage.firstMonth) }}. An estimate only: your bank sets the terms.</span
          >
        </p>
      </div>
    </div>
  </form>
</template>

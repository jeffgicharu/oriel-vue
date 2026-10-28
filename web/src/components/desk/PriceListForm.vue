<script setup lang="ts">
import { nextTick } from "vue";
import ErrorSummary from "@/components/ErrorSummary.vue";
import type { DeskPriceList, PriceListInput } from "@/api/desk";
import { isApiError } from "@/api/request";
import { useForm } from "@/composables/useForm";
import { bpField, parsePercentBP, parseShillings, when } from "@/lib/desk";
import { kes, percent, thousands } from "@/lib/format";
import { useDeskHomesStore } from "@/stores/deskHomes";

/**
 * A draft price list: a base price for each type, a premium for each floor
 * above level 2 and one for a river view. Sending it shows what it would do
 * to every home not yet reserved or sold; nothing changes until it is published.
 */
const props = defineProps<{ current: DeskPriceList; types: { code: "A" | "B" | "C" | "D" | "P"; name: string }[]; now: string }>();
const homes = useDeskHomesStore();

type Code = "A" | "B" | "C" | "D" | "P";
const CODES: Code[] = ["A", "B", "C", "D", "P"];

// The API names a base price "base.B" and a premium "floorBP"; the form names them as Oriel's did.
const FIELD: Record<string, string> = { floorBP: "floor", riverBP: "river" };
function formField(key: string): string {
  return key.startsWith("base.") ? key.slice(5) : (FIELD[key] ?? key);
}

const form = useForm({
  id: "price",
  values: {
    A: thousands(props.current.base.A),
    B: thousands(props.current.base.B),
    C: thousands(props.current.base.C),
    D: thousands(props.current.base.D),
    P: thousands(props.current.base.P),
    floor: bpField(props.current.floorBP),
    river: bpField(props.current.riverBP),
    note: "",
  },
  order: ["A", "B", "C", "D", "P", "floor", "river", "note"],
  check: (v): Record<string, string> => {
    const errs: Record<string, string> = {};
    for (const code of CODES) {
      const p = parseShillings(v[code]);
      if (p === null || p <= 0) errs[code] = `Type ${code}: please give the base price in shillings.`;
      else if (p % 10_000 !== 0) errs[code] = `Type ${code}: prices are set to the nearest KES\u00a010,000.`;
    }
    if (parsePercentBP(v.floor) === null) errs.floor = "The floor premium should be a percentage, such as 1 or 0.8.";
    if (parsePercentBP(v.river) === null) errs.river = "The river premium should be a percentage, such as 4.";
    if ([...v.note.trim()].length > 120) errs.note = "Please keep this under 120 characters.";
    return errs;
  },
  send: async (v) => {
    const list: PriceListInput = {
      base: { A: parseShillings(v.A) ?? 0, B: parseShillings(v.B) ?? 0, C: parseShillings(v.C) ?? 0, D: parseShillings(v.D) ?? 0, P: parseShillings(v.P) ?? 0 },
      floorBP: parsePercentBP(v.floor) ?? 0,
      riverBP: parsePercentBP(v.river) ?? 0,
    };
    try {
      const preview = await homes.previewPrices(list, v.note.trim());
      // The preview opens at its result, as Oriel's did.
      await nextTick();
      document.getElementById("preview")?.scrollIntoView({ block: "start" });
      document.getElementById("preview-title")?.focus({ preventScroll: true });
      return preview;
    } catch (e) {
      if (isApiError(e)) throw { ...e, fields: Object.fromEntries(Object.entries(e.fields).map(([k, m]) => [formField(k), m])) };
      throw e;
    }
  },
});
</script>

<template>
  <form class="desk-panel desk-form pricing__form" novalidate @submit.prevent="form.submit">
    <div class="desk-panel__head">
      <h2 class="desk-h2">A new price list</h2>
      <span class="desk-small">In force: list {{ current.id }}, {{ when(current.published, now) }}</span>
    </div>
    <ErrorSummary id="price" :listed="form.listed.value" :problem="form.problem.value" @retry="form.submit" />
    <div class="pricing__wrap">
      <table class="desk-table pricing__bases">
        <thead>
          <tr>
            <th scope="col">Type</th>
            <th scope="col" class="num">In force</th>
            <th scope="col">New base price (KES)</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in types" :key="t.code">
            <th scope="row"><span class="mono">{{ t.code }}</span> {{ t.name }}</th>
            <td class="num">{{ kes(current.base[t.code]) }}</td>
            <td>
              <label class="visually-hidden" :for="`price-${t.code}`">New base price for type {{ t.code }}</label>
              <input
                :id="`price-${t.code}`"
                v-model="form.values[t.code]"
                class="input tnum"
                :class="{ 'is-wrong': form.errors.value[t.code] }"
                :name="t.code"
                type="text"
                inputmode="numeric"
                :aria-invalid="form.errors.value[t.code] ? 'true' : undefined"
                :aria-describedby="form.errors.value[t.code] ? `price-${t.code}-note` : undefined"
                @input="form.touched(t.code)"
              />
              <p v-if="form.errors.value[t.code]" :id="`price-${t.code}-note`" class="field__error">{{ form.errors.value[t.code] }}</p>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="field-grid">
      <div class="field" :class="{ 'field--error': form.errors.value.floor }">
        <label class="field__label" for="price-floor">Floor premium, % a floor <span class="label">Now {{ percent(current.floorBP) }}</span></label>
        <input
          id="price-floor"
          v-model="form.values.floor"
          class="input tnum"
          name="floor"
          type="text"
          inputmode="decimal"
          :aria-invalid="form.errors.value.floor ? 'true' : undefined"
          :aria-describedby="form.errors.value.floor ? 'price-floor-note' : undefined"
          @input="form.touched('floor')"
        />
        <p v-if="form.errors.value.floor" id="price-floor-note" class="field__error">{{ form.errors.value.floor }}</p>
      </div>
      <div class="field" :class="{ 'field--error': form.errors.value.river }">
        <label class="field__label" for="price-river">River premium, % <span class="label">Now {{ percent(current.riverBP) }}</span></label>
        <input
          id="price-river"
          v-model="form.values.river"
          class="input tnum"
          name="river"
          type="text"
          inputmode="decimal"
          :aria-invalid="form.errors.value.river ? 'true' : undefined"
          :aria-describedby="form.errors.value.river ? 'price-river-note' : undefined"
          @input="form.touched('river')"
        />
        <p v-if="form.errors.value.river" id="price-river-note" class="field__error">{{ form.errors.value.river }}</p>
      </div>
    </div>
    <div class="field" :class="{ 'field--error': form.errors.value.note }">
      <label class="field__label" for="price-note">What changed <span class="label">Optional</span></label>
      <input
        id="price-note"
        v-model="form.values.note"
        class="input"
        name="note"
        type="text"
        maxlength="120"
        placeholder="Such as: two-bedroom base up 2%"
        :aria-invalid="form.errors.value.note ? 'true' : undefined"
        :aria-describedby="form.errors.value.note ? 'price-note-note' : undefined"
        @input="form.touched('note')"
      />
      <p v-if="form.errors.value.note" id="price-note-note" class="field__error">{{ form.errors.value.note }}</p>
    </div>
    <button class="btn btn--small" type="submit" :disabled="form.status.value === 'sending'">Preview the change</button>
  </form>
</template>

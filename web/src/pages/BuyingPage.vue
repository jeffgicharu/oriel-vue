<script setup lang="ts">
import { computed } from "vue";
import ArrowIcon from "@/components/ArrowIcon.vue";
import ContentHead from "@/components/ContentHead.vue";
import RequestView from "@/components/RequestView.vue";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { buyingCosts, buyingSteps, EXAMPLE_PRICE, planRows, questions } from "@/lib/content";
import { kes, kesShort, percent } from "@/lib/format";
import { useSiteStore } from "@/stores/site";

const site = useSiteStore();
usePageData(() => Promise.all([site.loadExamples(EXAMPLE_PRICE), site.loadOverview()]));
const steps = computed(() => buyingSteps(site.overview.data?.completion ?? ""));

usePageMeta({
  title: "Buying off-plan",
  description: "How buying a home at Oriel works: reserving, the deposit and the agreement, three ways to pay, the costs beyond the price and the questions buyers ask most.",
  path: "/buying",
  sheet: { number: "A-600", name: "Buying" },
  nav: "buying",
});
</script>

<template>
  <div class="wrap content-page">
    <ContentHead tag="A-600 · Buying" title="Buying off-plan, step by step." lede="How a reservation works, the three ways to pay, what it costs beyond the price, and the questions buyers ask us most." />
  </div>
  <section class="content-section content-section--first" aria-labelledby="steps-title">
    <div class="wrap">
      <h2 id="steps-title" class="visually-hidden">The steps</h2>
      <ol class="steps-row" role="list">
        <li v-for="(s, i) in steps" :key="s.title" class="steps-row__item">
          <span class="steps-row__n mono">{{ i + 1 }}</span>
          <p class="steps-row__when label">{{ s.when }}</p>
          <h3 class="steps-row__title">{{ s.title }}</h3>
          <p class="steps-row__text">{{ s.text }}</p>
        </li>
      </ol>
    </div>
  </section>
  <section id="payment-plans" class="content-section content-section--rule" aria-labelledby="plans-title">
    <div class="wrap">
      <div class="content-section__head">
        <p class="sheet-tag label">A-601 · Payment plans</p>
        <h2 id="plans-title" class="h2">Three ways to pay.</h2>
        <p class="lede">Shown on a home priced at {{ kes(EXAMPLE_PRICE) }}. Every home’s page works out its own schedule to the shilling.</p>
      </div>
      <RequestView :state="site.examples" :retry="() => site.loadExamples(EXAMPLE_PRICE)" loading-label="Working out the plans…">
        <template #default="{ data }">
          <div class="plan-cards">
            <article v-for="s in data" :key="s.plan.code" class="plan-card">
              <header class="plan-card__head">
                <h3 class="h3">{{ s.plan.name }}</h3>
                <p class="muted">{{ s.plan.summary }}</p>
                <p class="plan-card__total tnum">{{ kes(s.price) }}</p>
                <p v-if="s.discount > 0" class="plan-card__saving">{{ kes(s.discount) }} off for paying early</p>
              </header>
              <ol class="plan-card__list" role="list">
                <li v-for="(row, i) in planRows(s)" :key="i">
                  <span class="plan-card__share mono">{{ percent(row.shareBP) }}</span>
                  <span class="plan-card__label">
                    {{ row.label }}
                    <span v-if="row.each > 0" class="plan-card__each">{{ kesShort(row.each) }} each</span>
                  </span>
                  <span class="plan-card__amount tnum">{{ kesShort(row.amount) }}</span>
                </li>
              </ol>
            </article>
          </div>
        </template>
      </RequestView>
      <div class="plan-links">
        <RouterLink class="link-arrow" :to="{ name: 'residences' }">Find a home and see its schedule <ArrowIcon /></RouterLink>
        <a class="link-arrow" href="/api/price-list.pdf">Download the price list (PDF) <ArrowIcon /></a>
      </div>
    </div>
  </section>
  <section class="content-section content-section--rule" aria-labelledby="costs-title">
    <div class="wrap content-split">
      <div class="content-split__text">
        <div class="content-section__head">
          <p class="sheet-tag label">A-602 · Costs</p>
          <h2 id="costs-title" class="h2">What you pay beyond the price.</h2>
        </div>
        <p class="muted">These are the costs a buyer usually meets on top of the price. Your advocate will confirm them for your purchase.</p>
      </div>
      <table class="costs">
        <caption class="visually-hidden">Costs of buying beyond the price</caption>
        <tbody>
          <tr v-for="[name, amount, note] in buyingCosts" :key="name">
            <th scope="row">{{ name }}<span class="costs__note">{{ note }}</span></th>
            <td>{{ amount }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
  <section id="questions" class="content-section content-section--rule" aria-labelledby="faq-title">
    <div class="wrap content-split">
      <div class="content-split__text">
        <div class="content-section__head">
          <p class="sheet-tag label">A-603 · Questions</p>
          <h2 id="faq-title" class="h2">What buyers ask us.</h2>
        </div>
        <RouterLink class="link-arrow" :to="{ name: 'enquire' }">Ask your own question <ArrowIcon /></RouterLink>
      </div>
      <div class="faq">
        <details v-for="[q, a] in questions" :key="q" class="faq__item">
          <summary>{{ q }}</summary>
          <p>{{ a }}</p>
        </details>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import PhotoPicture from "@/components/PhotoPicture.vue";
import ProgressDrawing from "@/components/ProgressDrawing.vue";
import RequestView from "@/components/RequestView.vue";
import ContentHead from "@/components/ContentHead.vue";
import type { Progress } from "@/api/client";
import { usePageData } from "@/composables/usePageData";
import { usePageMeta } from "@/composables/usePageMeta";
import { monthYear, shortDate, shortMonthYear } from "@/lib/format";
import { photo } from "@/lib/photos";
import { useSiteStore } from "@/stores/site";

const site = useSiteStore();
usePageData(() => Promise.all([site.loadProgress(), site.loadModel()]));
const g = computed(() => site.progress.data);

function title(p: Progress | null | undefined): string {
  if (!p) return "Construction progress.";
  if (p.complete) return "Complete.";
  if (p.frame > 0 && p.frame < p.storeys) return "Rising on Riverside.";
  if (p.frame >= p.storeys) return "Topped out.";
  return "Breaking ground.";
}

usePageMeta(() => ({
  title: "Construction progress",
  description: g.value
    ? `Oriel is ${g.value.percent}% built: ${g.value.current.toLowerCase()}, completing in ${monthYear(g.value.completion)}. The programme from the first pile to the keys, and photographs from the site.`
    : "Oriel's construction programme, from the first pile to the keys, and photographs from the site.",
  path: "/progress",
  sheet: { number: "A-500", name: "Progress" },
  nav: "progress",
}));

function sizes(i: number) {
  return i === 0 ? "(min-width: 1520px) 1408px, 92vw" : "(min-width: 1000px) 50vw, 100vw";
}
</script>

<template>
  <RequestView :state="site.progress" :retry="() => site.loadProgress(true)" loading-label="Loading the programme…">
    <template #default="{ data }">
      <div class="wrap content-page">
        <ContentHead
          tag="A-500 · Progress"
          :title="title(data)"
          lede="The building goes up in public. This page follows the programme, from the first pile to the keys, and is brought up to date as each stage is reached."
        />
        <dl class="content-facts progress-facts">
          <div>
            <dt class="label">Overall</dt>
            <dd class="tnum">{{ data.percent }}%</dd>
            <div class="progress-bar" role="presentation"><span :style="`width: ${data.percent}%`" /></div>
          </div>
          <div><dt class="label">On site now</dt><dd>{{ data.current }}</dd></div>
          <div><dt class="label">Completion</dt><dd>{{ shortMonthYear(data.completion) }}</dd></div>
          <div><dt class="label">As at</dt><dd>{{ shortDate(data.asOf) }}</dd></div>
        </dl>
      </div>
      <section class="content-section content-section--first" aria-labelledby="programme-title">
        <div class="wrap progress">
          <figure class="content-sheet progress__drawing">
            <ProgressDrawing v-if="site.model.data" :model="site.model.data" :frame="data.frame" :storeys="data.storeys" :label="data.current.replace('Frame at ', '')" />
            <figcaption class="content-sheet__caption"><span class="mono">A-501</span> South elevation. Storeys with their frame built are solid; the one under way is hatched.</figcaption>
          </figure>
          <div class="progress__programme">
            <h2 id="programme-title" class="h3">The programme</h2>
            <ol class="timeline" role="list">
              <li v-for="m in data.milestones" :key="m.title" :class="['timeline__item', `is-${m.status}`]">
                <span class="timeline__dot" aria-hidden="true" />
                <p class="timeline__when mono">{{ m.when }}</p>
                <p class="timeline__title">
                  {{ m.title }}
                  <span v-if="m.status === 'done'" class="visually-hidden">(done)</span>
                  <span v-else-if="m.status === 'now'" class="timeline__now label">Under way</span>
                </p>
                <p class="timeline__note">{{ m.note }}</p>
              </li>
            </ol>
          </div>
        </div>
      </section>
      <section v-if="data.photos.length && photo(data.photos[0]!.image)" class="content-section content-section--rule" aria-labelledby="photos-title">
        <div class="wrap">
          <div class="content-section__head">
            <p class="sheet-tag label">A-502 · From the site</p>
            <h2 id="photos-title" class="h2">Photographs, newest first.</h2>
          </div>
          <ul class="site-photos" role="list">
            <template v-for="(ph, i) in data.photos" :key="ph.image">
              <li v-if="photo(ph.image)" class="site-photo" :class="{ 'site-photo--lead': i === 0 }">
                <figure class="crop">
                  <PhotoPicture :name="ph.image" :alt="ph.caption" :sizes="sizes(i)" />
                  <figcaption><span class="mono">{{ shortDate(ph.taken) }}</span> {{ ph.caption }}</figcaption>
                </figure>
              </li>
            </template>
          </ul>
        </div>
      </section>
    </template>
  </RequestView>
</template>

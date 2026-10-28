<script setup lang="ts">
import PhotoPicture from "@/components/PhotoPicture.vue";
import { isPortrait, type Shot } from "@/lib/photos";

defineProps<{ shots: Shot[]; note?: string }>();

function sizes(i: number) {
  return i === 0 ? "(min-width: 1000px) 60vw, 100vw" : "(min-width: 1000px) 38vw, 100vw";
}
</script>

<template>
  <template v-if="shots.length">
    <div class="gallery" :class="{ 'gallery--portrait': shots.length > 1 && isPortrait(shots[1]!.name) }">
      <figure v-for="(s, i) in shots" :key="s.name" class="gallery__item">
        <PhotoPicture :name="s.name" :alt="s.alt" :sizes="sizes(i)" />
        <figcaption class="gallery__caption">{{ s.caption }}</figcaption>
      </figure>
    </div>
    <p v-if="note" class="gallery__note">{{ note }}</p>
  </template>
</template>

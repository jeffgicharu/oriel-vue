<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import ErrorSummary from "@/components/ErrorSummary.vue";
import { useDeskHead } from "@/composables/useDeskHead";
import { useForm } from "@/composables/useForm";
import { safeNext } from "@/router/guards";
import { useDeskStore } from "@/stores/desk";
import "@/styles/desk.css";

const route = useRoute("desk-sign-in");
const desk = useDeskStore();
useDeskHead("Sign in", "desk desk--login");

// Where to go once signed in: the page that asked, kept to the desk.
const next = computed(() => safeNext(route.query.next));

const form = useForm({
  id: "signin",
  values: { username: "", password: "" },
  order: ["username", "password"],
  check: (v): Record<string, string> => ({
    ...(v.username.trim() ? {} : { username: "Please give your username." }),
    ...(v.password ? {} : { password: "Please give your password." }),
  }),
  send: async (v) => {
    const signed = await desk.signIn(v.username, v.password);
    // A whole new page, drawn on the server as the member now signed in.
    window.location.assign(next.value);
    return signed;
  },
});
</script>

<template>
  <main id="main" class="desk-login" tabindex="-1">
    <a class="desk-mark desk-mark--dark" href="/">
      <span class="desk-mark__name">ORIEL</span>
      <span class="desk-mark__sub">Sales desk</span>
    </a>
    <form class="desk-login__card" method="post" action="/desk/sign-in" novalidate @submit.prevent="form.submit">
      <h1 class="desk-h1">Sign in</h1>
      <ErrorSummary id="signin" :listed="form.listed.value" :problem="null" />
      <div v-if="form.problem.value" id="signin-errors" class="notice notice--alert" role="alert" tabindex="-1">
        {{ form.problem.value.title }}<template v-if="form.problem.value.detail"> {{ form.problem.value.detail }}</template>
      </div>
      <div class="field" :class="{ 'field--error': form.errors.value.username }">
        <label class="field__label" for="signin-username">Username</label>
        <input
          id="signin-username"
          v-model="form.values.username"
          class="input"
          name="username"
          type="text"
          autocomplete="username"
          autocapitalize="none"
          spellcheck="false"
          required
          :aria-invalid="form.errors.value.username ? 'true' : undefined"
          :aria-describedby="form.errors.value.username ? 'signin-username-note' : undefined"
          @input="form.touched('username')"
        />
        <p v-if="form.errors.value.username" id="signin-username-note" class="field__error">{{ form.errors.value.username }}</p>
      </div>
      <div class="field" :class="{ 'field--error': form.errors.value.password }">
        <label class="field__label" for="signin-password">Password</label>
        <input
          id="signin-password"
          v-model="form.values.password"
          class="input"
          name="password"
          type="password"
          autocomplete="current-password"
          required
          :aria-invalid="form.errors.value.password ? 'true' : undefined"
          :aria-describedby="form.errors.value.password ? 'signin-password-note' : undefined"
          @input="form.touched('password')"
        />
        <p v-if="form.errors.value.password" id="signin-password-note" class="field__error">{{ form.errors.value.password }}</p>
      </div>
      <button class="btn btn--cobalt btn--wide" type="submit" :disabled="form.status.value === 'sending' || form.status.value === 'sent'">
        {{ form.status.value === "sending" || form.status.value === "sent" ? "Signing in…" : "Sign in" }}
      </button>
    </form>
    <p class="desk-login__foot"><a href="/">Back to the website</a></p>
  </main>
</template>

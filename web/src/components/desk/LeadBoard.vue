<script setup lang="ts">
import Sortable, { type SortableEvent } from "sortablejs";
import { nextTick, onBeforeUnmount, onMounted, ref } from "vue";
import type { DeskLeads } from "@/api/desk";
import { isApiError } from "@/api/request";
import { ago, dayLabel, dotColour, isStage, type Stage } from "@/lib/desk";
import { useDeskLeadsStore } from "@/stores/deskLeads";

/**
 * The pipeline board: drag a lead between stages, or within one, and the
 * desk saves where it landed. Every card also has a Move button, so the same
 * moves can be made with a keyboard or a single tap. Moving to Lost asks for
 * a reason first.
 */
const props = defineProps<{ columns: DeskLeads["columns"]; now: string }>();
const emit = defineEmits<{ busy: [busy: boolean]; failed: [] }>();
const leads = useDeskLeadsStore();

type Column = DeskLeads["columns"][number];
type Lead = Column["leads"][number];

const board = ref<HTMLElement | null>(null);
const lists = ref<HTMLElement[]>([]);
const dialog = ref<HTMLDialogElement | null>(null);
const reason = ref("");
const saving = ref(0);
// The card whose Move buttons are open, and what the last move did, read out.
const menuFor = ref(0);
const said = ref("");
let sortables: Sortable[] = [];
// Where the dragged card sat, so it can be put back before Vue moves it.
let origin: { list: HTMLElement; next: Node | null } | null = null;

function idOf(el: Element | null | undefined): number {
  return Number((el as HTMLElement | null)?.dataset?.id ?? 0);
}

function askReason(): Promise<string | null> {
  const d = dialog.value;
  if (!d) return Promise.resolve(window.prompt("Why was this lead lost?"));
  reason.value = "";
  d.showModal();
  return new Promise((resolve) => {
    d.addEventListener("close", () => resolve(d.returnValue === "ok" && reason.value.trim() ? reason.value.trim() : null), { once: true });
  });
}

/** The reason a move needs: none, unless it is to Lost. Null when it was not given. */
async function reasonFor(fromStage: string, stage: Stage): Promise<string | null> {
  return stage === "lost" && fromStage !== "lost" ? askReason() : "";
}

/**
 * Moves a lead in the board's data, between the leads with the ids after
 * and before (0 for an end of the column), then saves where it went.
 */
async function place(id: number, fromStage: string, stage: Stage, after: number, before: number, why: string): Promise<boolean> {
  const source = props.columns.find((c) => c.stage === fromStage);
  const target = props.columns.find((c) => c.stage === stage);
  const i = source?.leads.findIndex((l) => l.id === id) ?? -1;
  if (!source || !target || i < 0) return false;
  const [lead] = source.leads.splice(i, 1);
  if (!lead) return false;
  lead.stage = stage;
  const at = after ? target.leads.findIndex((l) => l.id === after) + 1 : 0;
  target.leads.splice(at, 0, lead);

  saving.value = id;
  try {
    await leads.move(id, stage, after, before, why);
    return true;
  } catch (e) {
    window.alert(isApiError(e) ? e.title : "The lead could not be moved.");
    emit("failed");
    return false;
  } finally {
    saving.value = 0;
  }
}

async function dropped(event: SortableEvent) {
  const card = event.item;
  const from = event.from;
  const to = event.to;
  const start = origin;
  origin = null;
  if (from === to && event.oldIndex === event.newIndex) {
    emit("busy", false);
    return;
  }
  const stage = to.dataset.stage ?? "";
  const fromStage = from.dataset.stage ?? "";
  const id = idOf(card);
  const after = idOf(card.previousElementSibling);
  const before = idOf(card.nextElementSibling);
  // Sortable moved the card itself. It goes back where it was, and the data
  // moves instead, so what Vue shows and what it holds stay the same.
  if (start) start.list.insertBefore(card, start.next);
  if (!isStage(stage) || !id) {
    emit("busy", false);
    return;
  }
  const why = await reasonFor(fromStage, stage);
  if (why !== null) await place(id, fromStage, stage, after, before, why);
  emit("busy", false);
}

function moveButton(id: number): HTMLButtonElement | null {
  return board.value?.querySelector<HTMLButtonElement>(`.lead-card[data-id="${id}"] .lead-card__move`) ?? null;
}

function toggleMenu(id: number) {
  menuFor.value = menuFor.value === id ? 0 : id;
}

function closeMenu(id: number) {
  menuFor.value = 0;
  moveButton(id)?.focus();
}

/** One place up or down the column. The buttons stay open for the next step. */
async function shift(col: Column, i: number, by: -1 | 1) {
  const lead = col.leads[i];
  const j = i + by;
  if (!lead || j < 0 || j >= col.leads.length) return;
  // Its neighbours once it has moved.
  const rest = col.leads.filter((l) => l.id !== lead.id);
  const after = rest[j - 1]?.id ?? 0;
  const before = rest[j]?.id ?? 0;
  emit("busy", true);
  if (await place(lead.id, col.stage, col.stage, after, before, "")) {
    said.value = `${lead.name} moved ${by < 0 ? "up" : "down"}, ${j + 1} of ${col.leads.length} in ${col.title}.`;
  }
  emit("busy", false);
}

/** To the top of another stage. */
async function moveTo(col: Column, lead: Lead, to: Column) {
  if (!isStage(to.stage)) return;
  const why = await reasonFor(col.stage, to.stage);
  if (why === null) return;
  const before = to.leads[0]?.id ?? 0;
  emit("busy", true);
  const moved = await place(lead.id, col.stage, to.stage, 0, before, why);
  emit("busy", false);
  menuFor.value = 0;
  if (!moved) return;
  said.value = `${lead.name} moved to ${to.title}.`;
  await nextTick();
  moveButton(lead.id)?.focus();
}

onMounted(() => {
  sortables = lists.value.map((list) =>
    Sortable.create(list, {
      group: "leads",
      animation: 160,
      // Pointer-driven dragging behaves the same with a mouse, a pen or a finger.
      forceFallback: true,
      fallbackOnBody: true,
      fallbackTolerance: 4,
      delay: 120,
      delayOnTouchOnly: true,
      ghostClass: "is-ghost",
      chosenClass: "is-chosen",
      // Links and the Move buttons are pressed, never dragged.
      filter: "a, button",
      preventOnFilter: false,
      onStart: (event) => {
        menuFor.value = 0;
        origin = { list: event.from, next: event.item.nextSibling };
        emit("busy", true);
      },
      onEnd: (event) => void dropped(event),
    }),
  );
});

onBeforeUnmount(() => {
  for (const s of sortables) s.destroy();
  sortables = [];
});
</script>

<template>
  <div ref="board" class="board">
    <section v-for="col in columns" :key="col.stage" class="board__col" :aria-label="col.title">
      <header class="board__head">
        <h2 class="board__title">{{ col.title }}</h2>
        <span class="board__count tnum">{{ col.leads.length + col.more }}</span>
      </header>
      <ol ref="lists" class="board__cards" role="list" :data-stage="col.stage">
        <li v-for="(l, i) in col.leads" :key="l.id" class="lead-card" :class="{ 'is-saving': saving === l.id }" :data-id="l.id">
          <span class="lead-card__top">
            <RouterLink class="lead-card__name" :to="{ name: 'desk-lead', params: { id: String(l.id) } }">{{ l.name }}</RouterLink>
            <button
              class="lead-card__move"
              type="button"
              :aria-expanded="menuFor === l.id"
              :aria-controls="menuFor === l.id ? `moves-${l.id}` : undefined"
              @click="toggleMenu(l.id)"
            >
              <span class="visually-hidden">Move {{ l.name }}</span>
              <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="3" cy="8" r="1.4" /><circle cx="8" cy="8" r="1.4" /><circle cx="13" cy="8" r="1.4" /></svg>
            </button>
          </span>
          <span v-if="l.interest" class="lead-card__interest">{{ l.interest }}</span>
          <span class="lead-card__meta">
            <span class="desk-dot" :style="{ backgroundColor: dotColour(l.agentColour) }" :title="l.agent" />
            <span v-if="l.followUpOn && l.stage !== 'won' && l.stage !== 'lost'" class="lead-card__follow" :class="{ 'is-late': l.overdue }">{{ dayLabel(l.followUpOn) }}</span>
            <span class="lead-card__age">{{ ago(l.stageChanged, now) }}</span>
          </span>
          <div v-if="menuFor === l.id" :id="`moves-${l.id}`" class="lead-card__moves" role="group" :aria-label="`Move ${l.name}`" @keydown.esc.stop="closeMenu(l.id)">
            <button class="lead-card__step" type="button" :aria-disabled="i === 0" @click="shift(col, i, -1)">Up</button>
            <button class="lead-card__step" type="button" :aria-disabled="i === col.leads.length - 1" @click="shift(col, i, 1)">Down</button>
            <template v-for="c in columns" :key="c.stage">
              <button v-if="c.stage !== col.stage" class="lead-card__step" type="button" @click="moveTo(col, l, c)">To {{ c.title }}</button>
            </template>
          </div>
        </li>
      </ol>
      <RouterLink v-if="col.more > 0" class="board__more desk-link" :to="{ name: 'desk-leads', query: { view: 'table', stage: col.stage } }">and {{ col.more }} more</RouterLink>
    </section>
  </div>
  <p class="visually-hidden" role="status">{{ said }}</p>
  <dialog ref="dialog" class="desk-dialog" aria-labelledby="lost-title">
    <form class="desk-form" method="dialog">
      <h2 id="lost-title" class="desk-h2">Why was this lead lost?</h2>
      <div class="field">
        <label class="field__label" for="lost-reason">Reason</label>
        <input id="lost-reason" v-model="reason" class="input" name="reason" type="text" maxlength="200" required />
      </div>
      <div class="desk-form__row">
        <button class="btn btn--small" type="submit" value="ok">Move to Lost</button>
        <button class="btn btn--small btn--line" type="submit" value="cancel" formnovalidate>Cancel</button>
      </div>
    </form>
  </dialog>
</template>

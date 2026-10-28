// The desk's leads: the table and the pipeline board for the filters in the
// address, each lead's page, and what the team does to a lead.

import { defineStore } from "pinia";
import { reactive } from "vue";
import { unwrap, useApi } from "@/api/client";
import type { DeskLeadPage, DeskLeads } from "@/api/desk";
import { idle, load, type RequestState } from "@/api/request";
import type { Stage } from "@/lib/desk";

/** The leads list's filters, as the address holds them. */
export interface LeadQuery {
  view: "board" | "table";
  stage: Stage | "";
  agent: number;
  q: string;
  source: string;
  sort: "recent" | "follow" | "name";
}

export interface NewLeadValues extends Record<string, unknown> {
  name: string;
  phone: string;
  email: string;
  source: string;
  interest: string;
  buying: "home" | "investment";
  agent: number;
  note: string;
}

/** The query sent to the API: the board takes only the agent, as Oriel's did. */
export function leadApiQuery(f: LeadQuery) {
  if (f.view === "board") return { view: "board" as const, ...(f.agent ? { agent: f.agent } : {}) };
  return {
    view: "table" as const,
    ...(f.stage ? { stage: f.stage } : {}),
    ...(f.agent ? { agent: f.agent } : {}),
    ...(f.q ? { q: f.q } : {}),
    ...(f.source ? { source: f.source } : {}),
    ...(f.sort !== "recent" ? { sort: f.sort } : {}),
  };
}

export const useDeskLeadsStore = defineStore("deskLeads", () => {
  const api = useApi();
  const list = reactive(idle<DeskLeads>());
  const pages = reactive<Record<string, RequestState<DeskLeadPage>>>({});

  function loadList(f: LeadQuery, force = false) {
    const query = leadApiQuery(f);
    const key = new URLSearchParams(Object.entries(query).map(([k, v]) => [k, String(v)])).toString();
    return load(list, key, () => unwrap(api.GET("/api/desk/leads", { params: { query } })), force);
  }

  function page(id: string): RequestState<DeskLeadPage> {
    pages[id] ??= idle<DeskLeadPage>();
    return pages[id];
  }

  function loadLead(id: string, force = false) {
    return load(page(id), id, () => unwrap(api.GET("/api/desk/leads/{id}", { params: { path: { id: Number(id) } } })), force);
  }

  /** Keeps the lead's page as a change left it, so it shows at once. */
  function settle(changed: DeskLeadPage): DeskLeadPage {
    const id = String(changed.lead.id);
    Object.assign(page(id), { status: "ready", key: id, data: changed, error: null });
    return changed;
  }

  const path = (id: string) => ({ path: { id: Number(id) } });

  async function addNote(id: string, body: string) {
    return settle(await unwrap(api.POST("/api/desk/leads/{id}/notes", { params: path(id), body: { body } })));
  }

  async function setStage(id: string, stage: Stage, reason: string) {
    return settle(await unwrap(api.POST("/api/desk/leads/{id}/stage", { params: path(id), body: { stage, reason } })));
  }

  async function setFollowUp(id: string, day: string) {
    return settle(await unwrap(api.POST("/api/desk/leads/{id}/follow-up", { params: path(id), body: { day } })));
  }

  async function assign(id: string, agent: number) {
    return settle(await unwrap(api.POST("/api/desk/leads/{id}/assign", { params: path(id), body: { agent } })));
  }

  /** A card dropped on the board, between the cards now above and below it. */
  async function move(id: number, stage: Stage, after: number, before: number, reason: string) {
    await unwrap(api.POST("/api/desk/leads/{id}/move", { params: { path: { id } }, body: { stage, after, before, reason } }));
  }

  async function addLead(v: NewLeadValues) {
    const body = {
      name: v.name,
      phone: v.phone,
      email: v.email,
      source: v.source,
      interest: v.interest,
      buying: v.buying,
      note: v.note,
      ...(v.agent ? { agent: v.agent } : {}),
    };
    return settle(await unwrap(api.POST("/api/desk/leads", { body })));
  }

  return { list, pages, loadList, page, loadLead, addNote, setStage, setFollowUp, assign, move, addLead };
});

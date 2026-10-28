// What visitors send: questions, viewings and taps on WhatsApp. Each carries
// where the visitor came from, for the sales desk.

import { defineStore } from "pinia";
import { reactive } from "vue";
import { unwrap, useApi, type Calendar, type Viewing } from "@/api/client";
import { idle, load, type RequestState } from "@/api/request";
import { currentSource } from "@/lib/leadSource";

export interface EnquiryValues extends Record<string, unknown> {
  name: string;
  email: string;
  phone: string;
  interest: string;
  message: string;
  website: string;
  home: string;
}

export interface ViewingValues extends Record<string, unknown> {
  name: string;
  email: string;
  phone: string;
  party: number;
  notes: string;
  slot: string;
  website: string;
  home: string;
}

export const useContactStore = defineStore("contact", () => {
  const api = useApi();
  const calendar = reactive(idle<Calendar>());
  const viewings = reactive<Record<string, RequestState<Viewing>>>({});

  function loadCalendar(force = false) {
    return load(calendar, "calendar", () => unwrap(api.GET("/api/viewings/calendar")), force);
  }

  function viewing(ref: string): RequestState<Viewing> {
    viewings[ref] ??= idle<Viewing>();
    return viewings[ref];
  }

  function loadViewing(ref: string, force = false) {
    return load(viewing(ref), ref, () => unwrap(api.GET("/api/viewings/{ref}", { params: { path: { ref } } })), force);
  }

  async function enquire(v: EnquiryValues) {
    return unwrap(
      api.POST("/api/enquiries", {
        body: { name: v.name, email: v.email, phone: v.phone, interest: v.interest, message: v.message, website: v.website, home: v.home, source: currentSource() },
      }),
    );
  }

  async function bookViewing(v: ViewingValues): Promise<Viewing> {
    const booked = await unwrap(
      api.POST("/api/viewings", {
        body: { name: v.name, email: v.email, phone: v.phone, party: v.party, notes: v.notes, slot: v.slot, website: v.website, home: v.home, source: currentSource() },
      }),
    );
    const state = viewing(booked.ref);
    Object.assign(state, { status: "ready", key: booked.ref, data: booked, error: null });
    return booked;
  }

  async function cancelViewing(ref: string): Promise<Viewing> {
    const cancelled = await unwrap(api.POST("/api/viewings/{ref}/cancel", { params: { path: { ref } } }));
    Object.assign(viewing(ref), { status: "ready", key: ref, data: cancelled, error: null });
    void loadCalendar(true);
    return cancelled;
  }

  /** Records a tap on a WhatsApp link. It never holds up WhatsApp opening. */
  function tappedWhatsApp(home?: string) {
    void api.POST("/api/whatsapp", { body: { home: home ?? "", source: currentSource() }, keepalive: true }).catch(() => undefined);
  }

  async function deleteMine() {
    await unwrap(api.POST("/api/privacy/delete", {}));
    for (const key of Object.keys(viewings)) delete viewings[key];
  }

  return { calendar, viewings, loadCalendar, viewing, loadViewing, enquire, bookViewing, cancelViewing, tappedWhatsApp, deleteMine };
});

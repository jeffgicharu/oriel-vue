// Reservations: a request to hold a home, and its page, which updates the
// moment the sales team confirms it.

import { defineStore } from "pinia";
import { reactive } from "vue";
import { unwrap, useApi, type Reservation } from "@/api/client";
import { idle, load, type RequestState } from "@/api/request";
import { currentSource } from "@/lib/leadSource";

export interface ReserveValues extends Record<string, unknown> {
  name: string;
  email: string;
  phone: string;
  plan: string;
  buying: "home" | "investment";
  note: string;
  agree: boolean;
  website: string;
}

export const useReservationsStore = defineStore("reservations", () => {
  const api = useApi();
  const byRef = reactive<Record<string, RequestState<Reservation>>>({});

  function reservation(ref: string): RequestState<Reservation> {
    byRef[ref] ??= idle<Reservation>();
    return byRef[ref];
  }

  function loadReservation(ref: string, force = false) {
    return load(reservation(ref), ref, () => unwrap(api.GET("/api/reservations/{ref}", { params: { path: { ref } } })), force);
  }

  async function reserve(home: string, v: ReserveValues): Promise<Reservation> {
    const made = await unwrap(
      api.POST("/api/reservations", {
        body: { name: v.name, email: v.email, phone: v.phone, website: v.website, home, plan: v.plan, buying: v.buying, note: v.note, agree: v.agree, source: currentSource() },
      }),
    );
    Object.assign(reservation(made.ref), { status: "ready", key: made.ref, data: made, error: null });
    return made;
  }

  return { byRef, reservation, loadReservation, reserve };
});

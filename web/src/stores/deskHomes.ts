// The desk's homes and their sale: the stacking plan and each home's drawer,
// the reservations and the manager's decisions on them, and the price lists.

import { defineStore } from "pinia";
import { reactive, ref } from "vue";
import { unwrap, useApi } from "@/api/client";
import type { DeskHomeDetail, DeskPricePreview, DeskPricing, DeskReservations, DeskStack, PriceListInput, ReservationGroup } from "@/api/desk";
import { idle, load, type RequestState } from "@/api/request";

export const RESERVATION_GROUPS: ReservationGroup[] = ["waiting", "confirmed", "completed", "closed"];

export const useDeskHomesStore = defineStore("deskHomes", () => {
  const api = useApi();
  const stack = reactive(idle<DeskStack>());
  const homes = reactive<Record<string, RequestState<DeskHomeDetail>>>({});
  const reservations = reactive(idle<DeskReservations>());
  const pricing = reactive(idle<DeskPricing>());
  // The preview of a draft price list, until it is published or redrawn.
  const preview = ref<DeskPricePreview | null>(null);

  function loadStack(force = false) {
    return load(stack, "stack", () => unwrap(api.GET("/api/desk/stack")), force);
  }

  function home(id: string): RequestState<DeskHomeDetail> {
    homes[id] ??= idle<DeskHomeDetail>();
    return homes[id];
  }

  function loadHome(id: string, force = false) {
    return load(home(id), id, () => unwrap(api.GET("/api/desk/homes/{id}", { params: { path: { id } } })), force);
  }

  /** Keeps a home's drawer as a change left it, and brings the plan up to date. */
  function settleHome(changed: DeskHomeDetail): DeskHomeDetail {
    Object.assign(home(changed.home.id), { status: "ready", key: changed.home.id, data: changed, error: null });
    void loadStack(true);
    return changed;
  }

  async function setStatus(id: string, status: "withheld" | "available", reason: string) {
    return settleHome(await unwrap(api.POST("/api/desk/homes/{id}/status", { params: { path: { id } }, body: { status, reason } })));
  }

  /** Gives a home its own price; 0 puts it back on the list price. */
  async function setPrice(id: string, price: number, reason: string) {
    return settleHome(await unwrap(api.POST("/api/desk/homes/{id}/price", { params: { path: { id } }, body: { price, reason } })));
  }

  function loadReservations(group: ReservationGroup, force = false) {
    return load(reservations, group, () => unwrap(api.GET("/api/desk/reservations", { params: { query: { group } } })), force);
  }

  const ref_ = (r: string) => ({ path: { ref: r } });

  function confirm(r: string) {
    return unwrap(api.POST("/api/desk/reservations/{ref}/confirm", { params: ref_(r) }));
  }

  function decline(r: string, reason: string) {
    return unwrap(api.POST("/api/desk/reservations/{ref}/decline", { params: ref_(r), body: { reason } }));
  }

  function recordDeposit(r: string, amount: number) {
    return unwrap(api.POST("/api/desk/reservations/{ref}/deposit", { params: ref_(r), body: { amount } }));
  }

  function loadPricing(force = false) {
    return load(pricing, "pricing", () => unwrap(api.GET("/api/desk/pricing")), force);
  }

  async function previewPrices(list: PriceListInput, note: string) {
    preview.value = await unwrap(api.POST("/api/desk/pricing/preview", { body: { list, note } }));
    return preview.value;
  }

  async function publishPrices(list: PriceListInput, note: string) {
    const published = await unwrap(api.POST("/api/desk/pricing", { body: { list, note } }));
    Object.assign(pricing, { status: "ready", key: "pricing", data: published, error: null });
    preview.value = null;
    return published;
  }

  return {
    stack,
    homes,
    reservations,
    pricing,
    preview,
    loadStack,
    home,
    loadHome,
    setStatus,
    setPrice,
    loadReservations,
    confirm,
    decline,
    recordDeposit,
    loadPricing,
    previewPrices,
    publishPrices,
  };
});

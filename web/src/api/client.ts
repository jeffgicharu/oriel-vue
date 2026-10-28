// The typed client for Oriel's API. Its types are generated from
// api/openapi.yaml (pnpm api-types), the same contract the Go API is tested
// against, so a field renamed on one side fails the build on the other.

import createClient, { type Client } from "openapi-fetch";
import { inject, type InjectionKey } from "vue";
import type { components, paths } from "./schema";

export type Schemas = components["schemas"];
export type Home = Schemas["Home"];
export type HomeDetail = Schemas["HomeDetail"];
export type HomeList = Schemas["HomeList"];
export type TypeSummary = Schemas["TypeSummary"];
export type TypeDetail = Schemas["TypeDetail"];
export type Schedule = Schemas["Schedule"];
export type Plan = Schemas["Plan"];
export type Overview = Schemas["Overview"];
export type Progress = Schemas["Progress"];
export type BuildingModel = Schemas["BuildingModel"];
export type Calendar = Schemas["Calendar"];
export type Viewing = Schemas["Viewing"];
export type Reservation = Schemas["Reservation"];
export type Source = Schemas["Source"];
export type Touch = Schemas["Touch"];
export type Status = Schemas["Status"];
export type Rect = Schemas["Rect"];

export type Api = Client<paths>;

/** A call that failed, as a page shows it. */
export interface ApiError {
  /** The HTTP status, or 0 when the API could not be reached at all. */
  status: number;
  /** The problem's name: validation, slot-taken, busy, rate-limited, not-found, server or network. */
  kind: string;
  title: string;
  detail: string;
  /** What to say beside each form field that needs another look. */
  fields: Record<string, string>;
}

/** Turns what a failed call gave back into an ApiError. */
export function toApiError(error: unknown, response?: Response): ApiError {
  const p = (error ?? {}) as Partial<Schemas["Problem"]>;
  if (response && typeof p === "object" && typeof p.status === "number") {
    return {
      status: p.status,
      kind: String(p.type ?? "").split("/").pop() || "server",
      title: p.title ?? "Something went wrong",
      detail: p.detail ?? "",
      fields: p.errors ?? {},
    };
  }
  if (response) {
    return { status: response.status, kind: "server", title: "Something went wrong", detail: "Please try again in a moment.", fields: {} };
  }
  return {
    status: 0,
    kind: "network",
    title: "Oriel could not be reached",
    detail: "Check your connection and try again.",
    fields: {},
  };
}

/** Is the error the API being unreachable, rather than an answer? */
export function isNetworkError(e: ApiError): boolean {
  return e.status === 0;
}

type Result<T> = { data: T; error?: never; response: Response } | { data?: never; error: unknown; response: Response };

/** Awaits one call and returns its data, or throws an ApiError. */
export async function unwrap<T>(call: Promise<Result<T>>): Promise<Exclude<T, undefined>> {
  let result: Result<T>;
  try {
    result = await call;
  } catch {
    throw toApiError(undefined);
  }
  if (result.error !== undefined || !result.response.ok) throw toApiError(result.error, result.response);
  return result.data as Exclude<T, undefined>;
}

export function createApi(options: { baseUrl: string; fetch?: (request: Request) => Promise<Response> }): Api {
  return createClient<paths>({ baseUrl: options.baseUrl, ...(options.fetch ? { fetch: options.fetch } : {}) });
}

export const apiKey: InjectionKey<Api> = Symbol("api");

/** The API client of the app being rendered: one per request on the server. */
export function useApi(): Api {
  const api = inject(apiKey);
  if (!api) throw new Error("useApi() was called outside an app that provides the API");
  return api;
}

// The state of one read from the API, as a store keeps it: idle, loading,
// ready or failed, with the data from the last good answer. It is plain data,
// so it travels from the server's render to the browser with the rest of the
// store and the browser does not ask again for what the server already had.

import type { ApiError } from "./client";
import { toApiError } from "./client";

export type RequestStatus = "idle" | "loading" | "ready" | "error";

export interface RequestState<T> {
  status: RequestStatus;
  /** What the request was for, such as a filter's query; a new key loads afresh. */
  key: string | null;
  data: T | null;
  error: ApiError | null;
}

export function idle<T>(): RequestState<T> {
  return { status: "idle", key: null, data: null, error: null };
}

/**
 * Does the state already hold the answer for this key? An answer is data, a
 * read under way, or a refusal that will not change by asking again, such
 * as "there is no home 9999". A failure that might pass, such as the API
 * being busy, is asked again.
 */
export function settled<T>(state: RequestState<T>, key: string): boolean {
  if (state.key !== key) return false;
  if (state.status === "ready" || state.status === "loading") return true;
  return state.status === "error" && state.error !== null && state.error.status >= 400 && state.error.status < 500 && state.error.status !== 429;
}

/**
 * Runs a read into a request state. A read for an answer the state already
 * holds is skipped unless forced, so a page the server rendered is taken
 * over in the browser exactly as it arrived. The data from the last answer
 * stays on show while a new one loads, so a filter change never blanks the
 * list. If a newer read starts before this one answers, this one's answer
 * is dropped.
 */
export async function load<T>(state: RequestState<T>, key: string, read: () => Promise<T>, force = false): Promise<void> {
  if (!force && settled(state, key)) return;
  state.key = key;
  state.status = "loading";
  state.error = null;
  try {
    const data = await read();
    if (state.key !== key) return;
    state.data = data;
    state.status = "ready";
  } catch (e) {
    if (state.key !== key) return;
    state.error = isApiError(e) ? e : toApiError(undefined);
    state.status = "error";
  }
}

export function isApiError(e: unknown): e is ApiError {
  return typeof e === "object" && e !== null && "status" in e && "kind" in e && "fields" in e;
}

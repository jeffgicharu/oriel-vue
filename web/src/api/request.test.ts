import { describe, expect, it, vi } from "vitest";
import { idle, load } from "./request";

describe("a read's state", () => {
  it("goes from loading to ready, and does not ask again for what it has", async () => {
    const state = idle<number>();
    const read = vi.fn().mockResolvedValue(42);
    const pending = load(state, "a", read);
    expect(state.status).toBe("loading");
    await pending;
    expect(state).toMatchObject({ status: "ready", key: "a", data: 42 });
    await load(state, "a", read);
    expect(read).toHaveBeenCalledTimes(1);
    await load(state, "a", read, true);
    expect(read).toHaveBeenCalledTimes(2);
  });

  it("keeps the last answer on show while a new one loads", async () => {
    const state = idle<string>();
    await load(state, "one", () => Promise.resolve("first"));
    let finish!: (v: string) => void;
    const pending = load(state, "two", () => new Promise((r) => (finish = r)));
    expect(state.status).toBe("loading");
    expect(state.data).toBe("first");
    finish("second");
    await pending;
    expect(state.data).toBe("second");
  });

  it("drops an answer that arrives after a newer read started", async () => {
    const state = idle<string>();
    let slow!: (v: string) => void;
    const old = load(state, "old", () => new Promise((r) => (slow = r)));
    await load(state, "new", () => Promise.resolve("new"));
    slow("old");
    await old;
    expect(state.data).toBe("new");
  });

  it("keeps the API's error for the page to show, with a way to try again", async () => {
    const state = idle<string>();
    await load(state, "x", () => Promise.reject({ status: 503, kind: "busy", title: "Busy just now", detail: "", fields: {} }));
    expect(state.status).toBe("error");
    expect(state.error?.kind).toBe("busy");
    await load(state, "x", () => Promise.resolve("ok"));
    expect(state).toMatchObject({ status: "ready", data: "ok", error: null });
  });
});

import { describe, expect, it, vi } from "vitest";
import type { ApiError } from "@/api/client";
import { useForm } from "./useForm";

const refusal = (status: number, kind: string, fields: Record<string, string> = {}): ApiError => ({ status, kind, title: "Some answers need another look", detail: "", fields });

function makeForm(send: () => Promise<unknown>) {
  return useForm({
    id: "test",
    values: { name: "", email: "" },
    order: ["name", "email"],
    check: (v): Record<string, string> => (v.name ? {} : { name: "Please tell us your name." }),
    send,
  });
}

describe("a form's mistakes and its sending", () => {
  it("does not send while the browser's own checks fail, and lists them in the form's order", async () => {
    const send = vi.fn();
    const form = makeForm(send);
    await form.submit();
    expect(send).not.toHaveBeenCalled();
    expect(form.status.value).toBe("failed");
    expect(form.listed.value).toEqual([{ field: "name", message: "Please tell us your name.", href: "#test-name" }]);
  });

  it("shows the API's 422 field errors beside their fields", async () => {
    const form = makeForm(() => Promise.reject(refusal(422, "validation", { email: "That email address doesn’t look right." })));
    form.values.name = "Amani";
    await form.submit();
    expect(form.errors.value).toEqual({ email: "That email address doesn’t look right." });
    expect(form.problem.value).toBeNull();
  });

  it("shows a refusal that is not about a field at the top", async () => {
    const form = makeForm(() => Promise.reject(refusal(503, "busy")));
    form.values.name = "Amani";
    await form.submit();
    expect(form.errors.value).toEqual({});
    expect(form.problem.value?.kind).toBe("busy");
  });

  it("clears a field's mistake as soon as it is changed", async () => {
    const form = makeForm(() => Promise.resolve());
    await form.submit();
    form.touched("name");
    expect(form.errors.value).toEqual({});
  });

  it("keeps the answer once sent", async () => {
    const form = makeForm(() => Promise.resolve({ firstName: "Amani" }));
    form.values.name = "Amani";
    await form.submit();
    expect(form.status.value).toBe("sent");
    expect(form.result.value).toEqual({ firstName: "Amani" });
  });
});

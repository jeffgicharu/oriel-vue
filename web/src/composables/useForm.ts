import { computed, nextTick, reactive, ref } from "vue";
import type { ApiError } from "@/api/client";
import { isApiError } from "@/api/request";
import type { FieldErrors } from "@/lib/validation";

export type FormStatus = "idle" | "sending" | "sent" | "failed";

/**
 * A form's values, its mistakes and its sending. Mistakes found in the
 * browser and those the API sends back in a 422 are shown the same way:
 * listed at the top, each linked to its field, and beside the field itself.
 * A refusal that is not about a field (busy, too many forms, no connection)
 * is shown at the top with a way to try again.
 */
export function useForm<V extends Record<string, unknown>, R>(options: {
  /** The id the error summary and fields are named from, such as "enquiry". */
  id: string;
  values: V;
  /** The browser's own checks, the same as the API's. */
  check: (values: V) => FieldErrors;
  send: (values: V) => Promise<R>;
  /** The order fields appear in, for the summary. */
  order: (keyof V & string)[];
}) {
  const values = reactive(options.values) as V;
  const errors = ref<FieldErrors>({});
  const status = ref<FormStatus>("idle");
  const problem = ref<ApiError | null>(null);
  const result = ref<R | null>(null);

  const listed = computed(() =>
    Object.entries(errors.value)
      .sort(([a], [b]) => options.order.indexOf(a) - options.order.indexOf(b))
      .map(([field, message]) => ({ field, message, href: `#${options.id}-${field}` })),
  );

  function fieldId(field: keyof V & string) {
    return `${options.id}-${field}`;
  }

  /** Clears a field's mistake as soon as it is changed. */
  function touched(field: keyof V & string) {
    if (errors.value[field]) {
      const next = { ...errors.value };
      delete next[field];
      errors.value = next;
    }
  }

  async function focusSummary() {
    await nextTick();
    document.getElementById(`${options.id}-errors`)?.focus();
  }

  async function submit(): Promise<R | null> {
    if (status.value === "sending") return null;
    problem.value = null;
    const found = options.check(values);
    errors.value = found;
    if (Object.keys(found).length > 0) {
      status.value = "failed";
      await focusSummary();
      return null;
    }
    status.value = "sending";
    try {
      const r = await options.send(values);
      result.value = r as never;
      status.value = "sent";
      return r;
    } catch (e) {
      status.value = "failed";
      const err = isApiError(e) ? e : null;
      if (err && Object.keys(err.fields).length > 0) {
        errors.value = err.fields;
      } else {
        problem.value = err;
      }
      await focusSummary();
      return null;
    }
  }

  return { values, errors, listed, status, problem, result, fieldId, touched, submit };
}

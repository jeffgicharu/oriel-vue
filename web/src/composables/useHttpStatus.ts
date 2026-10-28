import { useSSRContext } from "vue";

/**
 * Sets the status the page is sent with, such as 404 for a home that does
 * not exist. It only means something while the server renders; in the
 * browser the page has already arrived.
 */
export function useHttpStatus() {
  const ctx = import.meta.env.SSR ? useSSRContext<{ status?: number }>() : undefined;
  return (status: number) => {
    if (ctx) ctx.status = status;
  };
}

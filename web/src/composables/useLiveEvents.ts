import { onBeforeUnmount, onMounted } from "vue";

export interface LiveEvent {
  type: string;
  id?: string;
  status?: string;
  text?: string;
}

/**
 * Listens to the API's live updates while the component is on the page. The
 * stream ends itself every few minutes and the browser reconnects; the
 * browser also reconnects after a dropped connection, waiting as the stream
 * asks. Nothing happens on the server, where there is nothing to listen to.
 */
export function useLiveEvents(path: string, onChange: (event: LiveEvent) => void) {
  let source: EventSource | undefined;

  onMounted(() => {
    if (!("EventSource" in window)) return;
    source = new EventSource(path);
    source.addEventListener("change", (event) => {
      try {
        onChange(JSON.parse((event as MessageEvent<string>).data) as LiveEvent);
      } catch {
        // Anything unexpected on the stream is left alone.
      }
    });
  });

  onBeforeUnmount(() => source?.close());
}

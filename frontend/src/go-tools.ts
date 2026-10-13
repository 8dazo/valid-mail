import type { ToolResult } from "./toolkit";

type Pending = {
  resolve: (result: ToolResult) => void;
  reject: (error: Error) => void;
};
type Engine = {
  worker: Worker;
  ready: Promise<void>;
  rejectReady: (error: Error) => void;
  pending: Map<number, Pending>;
};
let engine: Engine | null = null;
let requestId = 0;
function stop(current: Engine, error: Error) {
  current.worker.terminate();
  current.rejectReady(error);
  for (const pending of current.pending.values()) pending.reject(error);
  current.pending.clear();
  if (engine === current) engine = null;
}
function getEngine(): Engine {
  if (engine) return engine;
  const worker = new Worker(new URL("./go-worker.ts", import.meta.url), {
    type: "module",
  });
  let resolveReady!: () => void;
  let rejectReady!: (error: Error) => void;
  const ready = new Promise<void>((resolve, reject) => {
    resolveReady = resolve;
    rejectReady = reject;
  });
  const current: Engine = { worker, ready, rejectReady, pending: new Map() };
  engine = current;
  const startup = setTimeout(
    () =>
      stop(
        current,
        new Error("The local toolkit took too long to load. Please retry."),
      ),
    60_000,
  );
  worker.onmessage = (
    event: MessageEvent<{
      type: string;
      id?: number;
      result?: ToolResult;
      error?: string;
    }>,
  ) => {
    const data = event.data;
    if (data.type === "ready") {
      clearTimeout(startup);
      resolveReady();
    } else if (data.type === "error") {
      clearTimeout(startup);
      stop(
        current,
        new Error(data.error || "The local toolkit could not start."),
      );
    } else if (data.type === "result" && data.id !== undefined) {
      const pending = current.pending.get(data.id);
      if (!pending) return;
      current.pending.delete(data.id);
      if (data.error || !data.result)
        pending.reject(new Error(data.error || "No result was returned."));
      else pending.resolve(data.result);
    }
  };
  worker.onerror = () => {
    clearTimeout(startup);
    stop(
      current,
      new Error("The local toolkit could not start. Reload the page or retry."),
    );
  };
  return current;
}
export async function runGoTool(
  tool: string,
  values: Record<string, string>,
  signal?: AbortSignal,
): Promise<ToolResult> {
  if (signal?.aborted) throw new DOMException("Cancelled", "AbortError");
  for (const value of Object.values(values)) {
    if (new TextDecoder().decode(new TextEncoder().encode(value)) !== value)
      throw new Error(
        "The input contains an invalid Unicode sequence. Use valid UTF-8 text.",
      );
  }
  const current = getEngine();
  const id = ++requestId;
  return new Promise<ToolResult>((resolve, reject) => {
    let budget: ReturnType<typeof setTimeout> | undefined;
    const finish = (error?: Error, result?: ToolResult) => {
      if (budget) clearTimeout(budget);
      signal?.removeEventListener("abort", cancel);
      current.pending.delete(id);
      if (error) reject(error);
      else resolve(result!);
    };
    const cancel = () => {
      finish(new DOMException("Cancelled", "AbortError"));
      stop(current, new DOMException("Cancelled", "AbortError"));
    };
    signal?.addEventListener("abort", cancel, { once: true });
    current.ready.then(
      () => {
        if (signal?.aborted || engine !== current) {
          cancel();
          return;
        }
        budget = setTimeout(
          () =>
            stop(
              current,
              new Error(
                "This input exceeded the processing budget. Try a simpler or smaller example.",
              ),
            ),
          tool === "regex" ? 3000 : 12_000,
        );
        current.pending.set(id, {
          resolve: (result) => finish(undefined, result),
          reject: (error) => finish(error),
        });
        current.worker.postMessage({
          id,
          request: JSON.stringify({ tool, values }),
        });
      },
      (error: Error) => finish(error),
    );
  });
}

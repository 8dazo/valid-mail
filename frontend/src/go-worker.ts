/// <reference lib="webworker" />
import manifest from "./generated/go-runtime.json";

type GoRuntime = {
  importObject: WebAssembly.Imports;
  run: (instance: WebAssembly.Instance) => Promise<void>;
};
type Globals = typeof globalThis & {
  Go: new () => GoRuntime;
  __wemeGoRun?: (request: string) => string;
};
const context = self as unknown as DedicatedWorkerGlobalScope;
const globals = globalThis as Globals;
const initialize = (async () => {
  await import(/* @vite-ignore */ manifest.runtime);
  const runtime = new globals.Go();
  const response = await fetch(manifest.wasm);
  if (!response.ok)
    throw new Error("The local toolkit could not be downloaded.");
  const { instance } = await WebAssembly.instantiate(
    await response.arrayBuffer(),
    runtime.importObject,
  );
  void runtime.run(instance).catch(() =>
    context.postMessage({
      type: "error",
      error: "The local toolkit stopped. Please retry.",
    }),
  );
  for (let attempt = 0; attempt < 100 && !globals.__wemeGoRun; attempt++) {
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
  if (!globals.__wemeGoRun)
    throw new Error("The local toolkit could not start.");
  context.postMessage({ type: "ready" });
})();
initialize.catch((error: unknown) =>
  context.postMessage({
    type: "error",
    error:
      error instanceof Error
        ? error.message
        : "The local toolkit could not start.",
  }),
);
context.onmessage = async (
  event: MessageEvent<{ id: number; request: string }>,
) => {
  try {
    await initialize;
    const response = JSON.parse(globals.__wemeGoRun!(event.data.request));
    context.postMessage({ type: "result", id: event.data.id, ...response });
  } catch {
    context.postMessage({
      type: "result",
      id: event.data.id,
      error: "This input could not be processed. Try a smaller example.",
    });
  }
};

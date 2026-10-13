import type { HLJSApi } from "highlight.js";
let initialized: Promise<HLJSApi> | undefined;
export function highlighter(): Promise<HLJSApi> {
  initialized ??= (async () => {
    const [
      { default: hljs },
      { default: json },
      { default: yaml },
      { default: xml },
      { default: css },
      { default: javascript },
      { default: bash },
      { default: go },
      { default: ini },
      { default: markdown },
    ] = await Promise.all([
      import("highlight.js/lib/core"),
      import("highlight.js/lib/languages/json"),
      import("highlight.js/lib/languages/yaml"),
      import("highlight.js/lib/languages/xml"),
      import("highlight.js/lib/languages/css"),
      import("highlight.js/lib/languages/javascript"),
      import("highlight.js/lib/languages/bash"),
      import("highlight.js/lib/languages/go"),
      import("highlight.js/lib/languages/ini"),
      import("highlight.js/lib/languages/markdown"),
    ]);
    for (const [name, language] of Object.entries({
      json,
      yaml,
      xml,
      css,
      javascript,
      bash,
      go,
      ini,
      markdown,
    }))
      hljs.registerLanguage(name, language);
    hljs.registerAliases(["html", "svg"], { languageName: "xml" });
    hljs.registerAliases(["sh", "shell"], { languageName: "bash" });
    hljs.registerAliases(["js", "ts", "typescript"], {
      languageName: "javascript",
    });
    hljs.registerAliases(["toml"], { languageName: "ini" });
    return hljs;
  })();
  return initialized;
}

import { useEffect, useRef, useState } from "react";
import { Check, Copy, ChevronLeft, ChevronRight } from "lucide-react";
import { highlighter } from "./syntax";
import type { ToolResult } from "./toolkit";

export function CodeBlock({
  text,
  language = "text",
  wrap = false,
}: {
  text: string;
  language?: string;
  wrap?: boolean;
}) {
  const [highlighted, setHighlighted] = useState<{
    source: string;
    html: string;
  } | null>(null);
  useEffect(() => {
    let active = true;
    if (language !== "text" && text.length <= 100_000) {
      highlighter()
        .then((hljs) => {
          if (active && hljs.getLanguage(language))
            setHighlighted({
              source: text,
              html: hljs.highlight(text, { language, ignoreIllegals: true })
                .value,
            });
        })
        .catch(() => {});
    }
    return () => {
      active = false;
    };
  }, [text, language]);
  return (
    <pre className={`output-code ${wrap ? "wrap-code" : ""}`} tabIndex={0}>
      {highlighted?.source === text ? (
        <code dangerouslySetInnerHTML={{ __html: highlighted.html }} />
      ) : (
        <code>{text}</code>
      )}
    </pre>
  );
}
export function MarkdownPreview({ html }: { html: string }) {
  const container = useRef<HTMLDivElement>(null);
  useEffect(() => {
    let active = true;
    const blocks = container.current?.querySelectorAll(
      "pre code[class*='language-']",
    );
    if (blocks?.length)
      highlighter()
        .then((hljs) => {
          if (!active) return;
          blocks.forEach((block) => {
            const language = block.className.match(/language-([\w+-]+)/)?.[1];
            if (language && hljs.getLanguage(language))
              block.innerHTML = hljs.highlight(block.textContent ?? "", {
                language,
                ignoreIllegals: true,
              }).value;
          });
        })
        .catch(() => {});
    return () => {
      active = false;
    };
  }, [html]);
  return (
    <div
      ref={container}
      className="markdown-preview"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
export function DataTable({
  data,
}: {
  data: NonNullable<ToolResult["table"]>;
}) {
  const [page, setPage] = useState(0);
  const pageSize = 25;
  const pages = Math.max(1, Math.ceil(data.rows.length / pageSize));
  useEffect(() => setPage(0), [data]);
  const rows = data.rows.slice(page * pageSize, (page + 1) * pageSize);
  return (
    <div className="data-view">
      <div className="data-table-scroll" tabIndex={0} aria-label="Result table">
        <table>
          <thead>
            <tr>
              {data.columns.map((column, index) => (
                <th key={`${index}-${column}`} scope="col">
                  {column}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, index) => (
              <tr key={page * pageSize + index}>
                {row.map((cell, column) => (
                  <td key={column}>{cell}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
        {!rows.length && <p className="table-empty">No matching rows.</p>}
      </div>
      <div className="table-pagination">
        <span>
          {data.rows.length} row{data.rows.length === 1 ? "" : "s"}
          {pages > 1 ? ` · Page ${page + 1} of ${pages}` : ""}
        </span>
        {pages > 1 && (
          <div>
            <button
              className="icon-button small"
              disabled={page === 0}
              onClick={() => setPage(page - 1)}
              aria-label="Previous rows"
            >
              <ChevronLeft size={16} />
            </button>
            <button
              className="icon-button small"
              disabled={page + 1 === pages}
              onClick={() => setPage(page + 1)}
              aria-label="Next rows"
            >
              <ChevronRight size={16} />
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
export function GeneratedValues({
  text,
  onError,
}: {
  text: string;
  onError: (message: string) => void;
}) {
  const [copied, setCopied] = useState<number | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );
  async function copy(value: string, index: number) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(index);
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(null), 2000);
    } catch {
      onError(
        "Clipboard access is unavailable. Select the value to copy it manually.",
      );
    }
  }
  return (
    <ol className="generated-values">
      {text
        .split("\n")
        .filter(Boolean)
        .map((value, index) => (
          <li key={`${index}-${value}`}>
            <code>{value}</code>
            <button
              className="icon-button small"
              aria-label={`Copy value ${index + 1}`}
              onClick={() => copy(value, index)}
            >
              {copied === index ? <Check size={15} /> : <Copy size={15} />}
            </button>
          </li>
        ))}
    </ol>
  );
}

import { Component, useCallback, useEffect, useRef, useState } from "react";
import type { ErrorInfo, ReactNode, MouseEvent } from "react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import {
  ArrowRight,
  ArrowUpRight,
  ArrowLeft,
  Search,
  Star,
  ShieldCheck,
  Sparkles,
  GitFork as Github,
  X,
  Menu,
  ChevronDown,
  Check,
  Copy,
  Download,
  RotateCcw,
  LoaderCircle,
  AlertCircle,
  MailCheck,
  Braces,
  QrCode,
  KeyRound,
  Phone,
  Link,
  Network,
  FileJson,
  CodeXml,
  Table,
  Fingerprint,
  CalendarClock,
  FileText,
  GitCompareArrows,
  ScanLine,
  Hash,
  Binary,
  Clock,
  CaseSensitive,
  SearchCheck,
  Columns2,
  Rows3,
  FileUp,
  WrapText,
  Maximize2,
  LockKeyhole,
  Command,
  ExternalLink,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { categories, initialValues, tools } from "./catalog";
import type { Category, Tool } from "./catalog";
import { readList, saveList, storageKeys } from "./storage";
import type { ToolResult } from "./toolkit";
import {
  CodeBlock,
  DataTable,
  GeneratedValues,
  MarkdownPreview,
} from "./ResultViews";
import { LandingIntro, LandingOutro, StudioFooter } from "./Landing";
import "./fonts.css";
import "@fontsource/instrument-serif/latin-400.css";
import "@fontsource/instrument-serif/latin-400-italic.css";

const iconMap: Record<string, LucideIcon> = {
  MailCheck,
  Braces,
  QrCode,
  KeyRound,
  Phone,
  Link,
  Network,
  FileJson,
  CodeXml,
  Table,
  Fingerprint,
  CalendarClock,
  FileText,
  GitCompareArrows,
  ScanLine,
  Hash,
  Binary,
  Clock,
  CaseSensitive,
  SearchCheck,
  Columns2,
  Rows3,
  FileUp,
  WrapText,
  Maximize2,
};
const currentRoute = () => {
  const path = location.pathname;
  return path.startsWith("/tools/")
    ? (tools.find((tool) => `/tools/${tool.id}` === path)?.id ?? "not-found")
    : path === "/finder"
      ? "finder"
      : path === "/"
        ? null
        : "not-found";
};

function Brand({ onClick }: { onClick?: () => void }) {
  return (
    <a
      className="brand"
      href="/"
      onClick={(event) => {
        if (onClick && !event.metaKey && !event.ctrlKey) {
          event.preventDefault();
          onClick();
        }
      }}
      aria-label="WeMe home"
    >
      <img src="/weme-mark.svg" width="34" height="38" alt="" />
      <span>WeMe</span>
    </a>
  );
}
function ToolIcon({ tool, size = 23 }: { tool: Tool; size?: number }) {
  const Icon = iconMap[tool.icon] ?? Braces;
  return (
    <span className={`tool-icon ${tool.color}`}>
      <Icon size={size} strokeWidth={1.65} />
    </span>
  );
}
function ToolCard({
  tool,
  favorite,
  onFavorite,
  onOpen,
  index,
}: {
  tool: Tool;
  favorite: boolean;
  onFavorite: () => void;
  onOpen: (event: MouseEvent<HTMLAnchorElement>) => void;
  index: number;
}) {
  const reduced = useReducedMotion();
  return (
    <motion.article
      className={`tool-card card-${tool.color}`}
      initial={reduced ? false : { opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, delay: Math.min(index * 0.025, 0.25) }}
    >
      <div className="card-top">
        <ToolIcon tool={tool} />
        <button
          className={`favorite ${favorite ? "selected" : ""}`}
          aria-label={`${favorite ? "Remove" : "Add"} ${tool.name} ${favorite ? "from" : "to"} favorites`}
          aria-pressed={favorite}
          onClick={onFavorite}
        >
          <Star size={17} fill={favorite ? "currentColor" : "none"} />
        </button>
      </div>
      <a href={`/tools/${tool.id}`} onClick={onOpen} className="card-link">
        <h3>{tool.name}</h3>
        <p>{tool.description}</p>
        <div className="card-bottom">
          <span>{tool.server ? "Server assisted" : "Runs locally"}</span>
          <ArrowUpRight size={18} />
        </div>
      </a>
    </motion.article>
  );
}

function ToolWorkspace({
  tool,
  favorite,
  onFavorite,
  onBack,
  onRun,
}: {
  tool: Tool;
  favorite: boolean;
  onFavorite: () => void;
  onBack: () => void;
  onRun: () => void;
}) {
  const [values, setValues] = useState(() => initialValues(tool));
  const [result, setResult] = useState<ToolResult | null>(null);
  const [error, setError] = useState("");
  const [running, setRunning] = useState(false);
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState("");
  const [outputView, setOutputView] = useState<"preview" | "raw">("preview");
  const [stacked, setStacked] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const [wrapCode, setWrapCode] = useState(false);
  const [inputError, setInputError] = useState("");
  const [inputFile, setInputFile] = useState("");
  const [cancelled, setCancelled] = useState(false);
  const fileInput = useRef<HTMLInputElement | null>(null);
  const controller = useRef<AbortController | null>(null);
  const generation = useRef(0);
  const copyTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(
    () => () => {
      generation.current++;
      controller.current?.abort();
      if (copyTimer.current) clearTimeout(copyTimer.current);
    },
    [],
  );
  const reset = () => {
    generation.current++;
    controller.current?.abort();
    setValues(initialValues(tool));
    setResult(null);
    setError("");
    setRunning(false);
    setCopied(false);
    setCopyError("");
    setInputError("");
    setInputFile("");
    setCancelled(false);
  };
  const execute = async (event?: React.FormEvent) => {
    event?.preventDefault();
    if (running) return;
    controller.current = new AbortController();
    const run = ++generation.current;
    const timeout = setTimeout(
      () => controller.current?.abort(),
      ["finder", "email"].includes(tool.id)
        ? 45_000
        : tool.engine === "go"
          ? 75_000
          : 30_000,
    );
    setRunning(true);
    setCancelled(false);
    setError("");
    setCopyError("");
    setResult(null);
    setCopied(false);
    try {
      const { runTool } = await import("./toolkit");
      const output = await runTool(tool.id, values, controller.current.signal);
      if (generation.current === run) {
        setResult(output);
        setOutputView("preview");
        onRun();
      }
    } catch (err) {
      if (generation.current === run)
        setError(
          err instanceof Error
            ? err.name === "AbortError" || err.name === "TimeoutError"
              ? "The request timed out. Please try again."
              : err.message
            : String(err),
        );
    } finally {
      clearTimeout(timeout);
      if (generation.current === run) setRunning(false);
    }
  };
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(result?.text ?? "");
      setCopyError("");
      setCopied(true);
      if (copyTimer.current) clearTimeout(copyTimer.current);
      copyTimer.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopyError(
        "Clipboard access is unavailable. Select the output to copy it manually.",
      );
    }
  };
  const download = () => {
    if (!result) return;
    const body = result.binary
      ? Uint8Array.from(atob(result.binary.base64), (char) =>
          char.charCodeAt(0),
        )
      : result.mime === "image/png" &&
          result.image?.startsWith("data:image/png;base64,")
        ? Uint8Array.from(atob(result.image.split(",")[1]), (char) =>
            char.charCodeAt(0),
          )
        : result.text;
    const url = URL.createObjectURL(
      new Blob([body], { type: result.mime ?? "text/plain;charset=utf-8" }),
    );
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download =
      result.binary?.filename ?? result.filename ?? `${tool.id}-result.txt`;
    anchor.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };
  const updateValues = (
    update: React.SetStateAction<Record<string, string>>,
  ) => {
    generation.current++;
    controller.current?.abort();
    setRunning(false);
    setResult(null);
    setError("");
    setCopyError("");
    setCancelled(false);
    setInputError("");
    setValues(update);
  };
  const loadFile = async (file?: File) => {
    if (!file) return;
    if (file.size > 400_000) {
      setInputError("Choose a UTF-8 text file smaller than 400 KB.");
      return;
    }
    try {
      const bytes = new Uint8Array(await file.arrayBuffer());
      if (tool.id === "gzip" && file.name.toLowerCase().endsWith(".gz")) {
        if (bytes.length > 75_000)
          throw new Error(
            "Use a gzip file smaller than 75 KB; decoded text is limited to 1 MB.",
          );
        let binary = "";
        for (const byte of bytes) binary += String.fromCharCode(byte);
        updateValues((previous) => ({
          ...previous,
          input: btoa(binary),
          mode: "decompress",
        }));
        setInputFile(file.name);
        return;
      }
      const text = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
      if (text.length > 100_000)
        throw new Error("Keep imported text under 100,000 characters.");
      updateValues((previous) => ({ ...previous, input: text }));
      setInputFile(file.name);
    } catch (error) {
      setInputError(
        error instanceof Error &&
          (error.message.includes("100,000") || error.message.includes("gzip"))
          ? error.message
          : "This file is not UTF-8 text. Try a text, Markdown, configuration, or PEM file.",
      );
    }
  };
  const cancelRun = () => {
    generation.current++;
    controller.current?.abort();
    setRunning(false);
    setCancelled(true);
    setError("");
  };
  const generated =
    ["password", "id", "ulid"].includes(tool.id) &&
    !(tool.id === "ulid" && values.mode === "decode");
  const hasPreview = !!(
    result &&
    (result.html ||
      result.diff ||
      result.image ||
      result.table ||
      result.swatches ||
      result.segments ||
      result.smtp ||
      generated)
  );
  const language =
    result?.language ??
    (result?.mime?.includes("json") ||
    result?.filename?.endsWith(".json") ||
    /^[\s]*[\[{]/.test(result?.text ?? "")
      ? "json"
      : result?.filename?.endsWith(".yaml")
        ? "yaml"
        : result?.filename?.endsWith(".xml")
          ? "xml"
          : "text");
  const hasTextInput = tool.fields.some(
    (field) => field.key === "input" && field.type === "textarea",
  );
  return (
    <main
      className={`workspace container workspace-${tool.layout ?? "compact"} ${expanded ? "workspace-expanded" : ""}`}
      id="main-content"
    >
      <button className="back-link" onClick={onBack}>
        <ArrowLeft size={16} /> All tools
      </button>
      <div className="workspace-heading">
        <div>
          <div className="eyebrow-label">
            {tool.category} <span> / </span>{" "}
            {tool.server ? "Server assisted" : "Private by design"}
          </div>
          <h1>
            {tool.name}
            <span>.</span>
          </h1>
          <p>{tool.description}</p>
        </div>
        <button
          className={`favorite large ${favorite ? "selected" : ""}`}
          onClick={onFavorite}
          aria-label={favorite ? "Remove from favorites" : "Add to favorites"}
          aria-pressed={favorite}
        >
          <Star size={20} fill={favorite ? "currentColor" : "none"} />
        </button>
      </div>
      <div className="workspace-toolbar" aria-label="Workspace controls">
        <span>
          {tool.server
            ? "Server-assisted check"
            : "Your input stays in this browser"}
        </span>
        <div>
          <button
            className="text-button"
            onClick={() => setStacked(!stacked)}
            aria-pressed={stacked}
          >
            {stacked ? <Columns2 size={15} /> : <Rows3 size={15} />}
            {stacked ? "Side by side" : "Stack panels"}
          </button>
          <button
            className="text-button"
            onClick={() => setExpanded(!expanded)}
            aria-pressed={expanded}
          >
            <Maximize2 size={15} />
            {expanded ? "Standard width" : "More room"}
          </button>
        </div>
      </div>
      <div className={`workspace-grid ${stacked ? "stacked-panels" : ""}`}>
        <form
          className="input-panel"
          onSubmit={execute}
          onKeyDown={(event) => {
            if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
              event.preventDefault();
              void execute();
            }
          }}
        >
          <div className="panel-heading">
            <span>Your input</span>
            <button
              type="button"
              className="text-button"
              onClick={() => {
                reset();
                setValues({ ...initialValues(tool), ...tool.example });
              }}
            >
              <Sparkles size={14} /> Try example
            </button>
          </div>
          {hasTextInput && (
            <div className="input-file-bar">
              <button
                className="text-button"
                type="button"
                onClick={() => fileInput.current?.click()}
              >
                <FileUp size={14} />
                Import text file
              </button>
              <input
                ref={fileInput}
                type="file"
                hidden
                aria-label="Import input file"
                accept={
                  tool.id === "gzip"
                    ? ".gz,.txt,.md,.json,.csv"
                    : ".txt,.md,.markdown,.json,.yaml,.yml,.xml,.csv,.html,.css,.js,.toml,.ini,.pem,.crt"
                }
                onChange={(event) => {
                  void loadFile(event.target.files?.[0]);
                  event.target.value = "";
                }}
              />
              <span>{inputFile || "UTF-8 · up to 100,000 characters"}</span>
            </div>
          )}
          {inputError && (
            <p className="input-file-error" role="alert">
              {inputError}
            </p>
          )}
          <div className="fields">
            {tool.fields
              .filter(
                (field) =>
                  !field.when || values[field.when[0]] === field.when[1],
              )
              .map((field) => (
                <label className="field" key={field.key}>
                  <span>{field.label}</span>
                  {field.type === "textarea" ? (
                    <textarea
                      aria-label={field.label}
                      value={values[field.key]}
                      placeholder={field.placeholder}
                      maxLength={100_000}
                      onChange={(event) =>
                        updateValues((previous) => ({
                          ...previous,
                          [field.key]: event.target.value,
                        }))
                      }
                      spellCheck={false}
                      autoComplete="off"
                    />
                  ) : field.type === "select" ? (
                    <div className="select-wrap">
                      <select
                        aria-label={field.label}
                        value={values[field.key]}
                        onChange={(event) =>
                          updateValues((previous) => ({
                            ...previous,
                            [field.key]: event.target.value,
                          }))
                        }
                      >
                        {field.options?.map(([value, label]) => (
                          <option key={value} value={value}>
                            {label}
                          </option>
                        ))}
                      </select>
                      <ChevronDown size={15} />
                    </div>
                  ) : (
                    <input
                      aria-label={field.label}
                      type={field.type ?? "text"}
                      value={values[field.key]}
                      placeholder={field.placeholder}
                      maxLength={100_000}
                      onChange={(event) =>
                        updateValues((previous) => ({
                          ...previous,
                          [field.key]: event.target.value,
                        }))
                      }
                      autoComplete="off"
                      spellCheck={false}
                    />
                  )}
                  {field.type === "textarea" && (
                    <span className="editor-meta">
                      {(values[field.key] ?? "").length.toLocaleString()} /
                      100,000 characters
                    </span>
                  )}
                  {field.helper && (
                    <small className="field-helper">{field.helper}</small>
                  )}
                </label>
              ))}
          </div>
          <div className="input-actions">
            <button className="button primary" type="submit" disabled={running}>
              {running ? (
                <>
                  <LoaderCircle size={16} className="spin" /> Working…
                </>
              ) : (
                <>
                  {tool.action}
                  <ArrowRight size={17} />
                </>
              )}
            </button>
            {running && (
              <button className="text-button" type="button" onClick={cancelRun}>
                Cancel
              </button>
            )}
            <button
              className="icon-button"
              type="button"
              onClick={reset}
              aria-label="Reset input and output"
            >
              <RotateCcw size={17} />
            </button>
          </div>
          <div className="run-shortcut">Ctrl / ⌘ + Enter to run</div>
          <div className="privacy-note">
            <LockKeyhole size={14} />
            <span>
              {tool.note ??
                "Your input stays in this browser. No upload, no account, no tracking."}
            </span>
          </div>
          {tool.source && (
            <a
              className="source-link"
              href={tool.source}
              target="_blank"
              rel="noopener noreferrer"
            >
              <Github size={15} /> Built with {tool.library}
              <ExternalLink size={12} />
            </a>
          )}
        </form>
        <section
          className="output-panel"
          aria-live="polite"
          aria-busy={running}
        >
          <div className="panel-heading">
            <span>Your result</span>
            {result && (
              <div className="output-actions">
                {hasPreview && (
                  <button
                    className="text-button"
                    onClick={() =>
                      setOutputView(outputView === "raw" ? "preview" : "raw")
                    }
                  >
                    {outputView === "raw"
                      ? "Result view"
                      : result.html
                        ? "HTML source"
                        : "Raw output"}
                  </button>
                )}
                {(!hasPreview || outputView === "raw") && (
                  <button
                    className={`icon-button small ${wrapCode ? "active" : ""}`}
                    onClick={() => setWrapCode(!wrapCode)}
                    aria-label="Wrap output lines"
                    aria-pressed={wrapCode}
                  >
                    <WrapText size={16} />
                  </button>
                )}
                <button
                  className="icon-button small"
                  onClick={copy}
                  aria-label={copied ? "Copied result" : "Copy result"}
                >
                  {copied ? <Check size={16} /> : <Copy size={16} />}
                </button>
                <button
                  className="icon-button small"
                  onClick={download}
                  aria-label="Download result"
                >
                  <Download size={16} />
                </button>
              </div>
            )}
          </div>
          {error ? (
            <div className="error-message" role="alert">
              <AlertCircle size={22} />
              <h3>Let’s try that again.</h3>
              <p>{error}</p>
            </div>
          ) : result ? (
            <div className="result-body">
              {copyError && (
                <p className="result-notice" role="status">
                  {copyError}
                </p>
              )}
              {result.title && <h2>{result.title}</h2>}
              {result.smtp && (
                <section
                  className="smtp-report"
                  aria-label="SMTP verification evidence"
                >
                  <div className="smtp-confidence">
                    <span>Verdict confidence</span>
                    <strong>
                      {result.smtp.confidence === null
                        ? "Not measured"
                        : `${result.smtp.confidence}%`}
                    </strong>
                  </div>
                  {result.smtp.confidence !== null && (
                    <progress
                      aria-label="Verdict confidence"
                      value={result.smtp.confidence}
                      max={100}
                    />
                  )}
                  <p>{result.smtp.evidence}</p>
                  <small>
                    Confidence reflects the evidence for this verdict. It does
                    not guarantee delivery or identify the mailbox owner.
                  </small>
                  <dl>
                    {result.smtp.details.map((detail) => (
                      <div key={detail.label}>
                        <dt>{detail.label}</dt>
                        <dd>{detail.value}</dd>
                      </div>
                    ))}
                  </dl>
                </section>
              )}
              {result.facts && (
                <div className="facts">
                  {result.facts.map((fact) => (
                    <div key={fact.label}>
                      <span>{fact.label}</span>
                      <strong className={fact.tone ?? ""}>{fact.value}</strong>
                    </div>
                  ))}
                </div>
              )}
              {result.notice && (
                <p className="result-notice">
                  <ShieldCheck size={16} />
                  <span>{result.notice}</span>
                </p>
              )}
              {outputView === "preview" && result.image ? (
                <div
                  className={`qr-result ${tool.id === "barcode" ? "barcode-result" : ""}`}
                >
                  <img
                    src={result.image}
                    alt={result.imageAlt ?? "Generated QR code"}
                  />
                </div>
              ) : outputView === "preview" && result.html ? (
                <MarkdownPreview html={result.html} />
              ) : outputView === "preview" && result.diff ? (
                <div className="diff-result">
                  {result.diff.map((part, index) =>
                    part.added ? (
                      <ins key={index}>{part.value}</ins>
                    ) : part.removed ? (
                      <del key={index}>{part.value}</del>
                    ) : (
                      <span key={index}>{part.value}</span>
                    ),
                  )}
                </div>
              ) : outputView === "preview" &&
                (result.table || result.swatches || result.segments) ? (
                <>
                  {result.swatches && (
                    <div className="color-swatches">
                      {result.swatches.map((swatch) => (
                        <div key={swatch.name}>
                          <div
                            className="color-chip"
                            style={{ backgroundColor: swatch.color }}
                          />
                          <strong>{swatch.color}</strong>
                          <span>{swatch.name}</span>
                        </div>
                      ))}
                    </div>
                  )}
                  {result.segments && (
                    <div
                      className="regex-preview"
                      aria-label="Highlighted matches"
                    >
                      {result.segments.map((segment, index) =>
                        segment.matched ? (
                          <mark key={index}>{segment.value || "∅"}</mark>
                        ) : (
                          <span key={index}>{segment.value}</span>
                        ),
                      )}
                    </div>
                  )}
                  {result.table && <DataTable data={result.table} />}
                </>
              ) : outputView === "preview" && generated ? (
                <GeneratedValues text={result.text} onError={setCopyError} />
              ) : outputView === "preview" && result.smtp ? null : (
                <CodeBlock
                  text={result.text}
                  language={language}
                  wrap={wrapCode}
                />
              )}
              <div className="result-footer">
                <span>{result.mime ?? "Plain text"}</span>
                <span>
                  {new TextEncoder()
                    .encode(result.text)
                    .length.toLocaleString()}{" "}
                  bytes · ready to copy or download
                </span>
              </div>
            </div>
          ) : (
            <div className="output-empty">
              <div className="empty-illustration">
                {running ? (
                  <LoaderCircle size={30} className="spin" />
                ) : (
                  <ToolIcon tool={tool} size={32} />
                )}
              </div>
              <h3>
                {running
                  ? "A little work, a little magic."
                  : cancelled
                    ? "Ready when you are."
                    : "Good things start here."}
              </h3>
              <p>
                {running
                  ? tool.engine === "go"
                    ? "Preparing your local workspace. The first run can take longer; later tools reuse it."
                    : "Your result will appear in a moment."
                  : "Add your input or try an example.\nWe’ll take care of the rest."}
              </p>
            </div>
          )}
        </section>
      </div>
      <div className="workspace-footnote">
        <ShieldCheck size={15} />
        <span>
          {tool.server
            ? "Transparent results. No promises about mailbox ownership."
            : "No sign-up. No server upload. Just a useful tool."}
        </span>
      </div>
    </main>
  );
}

function AppContent() {
  const [route, setRoute] = useState(currentRoute);
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState<
    Category | "All tools" | "Favorites" | "Recently used"
  >("All tools");
  const [favorites, setFavorites] = useState(() =>
    readList(storageKeys.favorites),
  );
  const [recent, setRecent] = useState(() => readList(storageKeys.recent));
  const [mobileMenu, setMobileMenu] = useState(false);
  const searchInput = useRef<HTMLInputElement>(null);
  const tool = tools.find((item) => item.id === route);
  const reduced = useReducedMotion();
  useEffect(() => {
    const onPop = () => {
      setRoute(currentRoute());
      setMobileMenu(false);
    };
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);
  useEffect(() => {
    document.title = tool
      ? `${tool.name} — WeMe`
      : "WeMe — Small tools. Big possibilities.";
  }, [tool]);
  useEffect(() => {
    saveList(storageKeys.favorites, favorites);
  }, [favorites]);
  useEffect(() => {
    saveList(storageKeys.recent, recent);
  }, [recent]);
  const navigate = useCallback((id: string | null) => {
    history.pushState({}, "", id ? `/tools/${id}` : "/");
    setRoute(id);
    setMobileMenu(false);
    window.scrollTo({ top: 0, behavior: "instant" });
  }, []);
  const goSection = (id: string) => {
    if (route) navigate(null);
    setMobileMenu(false);
    setTimeout(
      () =>
        document
          .getElementById(id)
          ?.scrollIntoView({ behavior: reduced ? "instant" : "smooth" }),
      30,
    );
  };
  useEffect(() => {
    const shortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        if (route) navigate(null);
        setTimeout(() => {
          document.getElementById("tools")?.scrollIntoView();
          searchInput.current?.focus();
        }, 30);
      }
    };
    window.addEventListener("keydown", shortcut);
    return () => window.removeEventListener("keydown", shortcut);
  }, [route, navigate]);
  const favorite = (id: string) =>
    setFavorites((previous) =>
      previous.includes(id)
        ? previous.filter((item) => item !== id)
        : [...previous, id],
    );
  const filtered = tools.filter(
    (item) =>
      (category === "All tools" ||
        (category === "Favorites" && favorites.includes(item.id)) ||
        (category === "Recently used" && recent.includes(item.id)) ||
        item.category === category) &&
      `${item.name} ${item.description} ${item.library}`
        .toLowerCase()
        .includes(query.toLowerCase()),
  );
  if (category === "Recently used")
    filtered.sort((a, b) => recent.indexOf(a.id) - recent.indexOf(b.id));
  const clickTool = (id: string) => (event: MouseEvent<HTMLAnchorElement>) => {
    if (
      !event.ctrlKey &&
      !event.metaKey &&
      !event.shiftKey &&
      event.button === 0
    ) {
      event.preventDefault();
      navigate(id);
    }
  };
  return (
    <>
      <a className="skip-link" href="#main-content">
        Skip to content
      </a>
      <header className="site-header">
        <Brand onClick={() => navigate(null)} />
        <nav
          className={mobileMenu ? "nav open" : "nav"}
          aria-label="Main navigation"
        >
          <button onClick={() => goSection("tools")}>Tools</button>
          <button onClick={() => goSection("how-it-works")}>
            How it works
          </button>
          <button onClick={() => goSection("open-source")}>Open source</button>
          <button onClick={() => goSection("about")}>About</button>
        </nav>
        <div className="header-actions">
          <button
            className="button light header-cta"
            onClick={() => goSection("tools")}
          >
            Explore tools <ArrowRight size={18} />
          </button>
          <button
            className="menu-toggle icon-button"
            onClick={() => setMobileMenu(!mobileMenu)}
            aria-label={mobileMenu ? "Close navigation" : "Open navigation"}
            aria-expanded={mobileMenu}
          >
            {mobileMenu ? <X size={22} /> : <Menu size={22} />}
          </button>
        </div>
      </header>
      <AnimatePresence mode="wait">
        {tool ? (
          <motion.div
            key={tool.id}
            initial={reduced ? false : { opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15 }}
          >
            <ToolWorkspace
              tool={tool}
              favorite={favorites.includes(tool.id)}
              onFavorite={() => favorite(tool.id)}
              onBack={() => navigate(null)}
              onRun={() =>
                setRecent((previous) =>
                  [tool.id, ...previous.filter((id) => id !== tool.id)].slice(
                    0,
                    8,
                  ),
                )
              }
            />
          </motion.div>
        ) : route === "not-found" ? (
          <main id="main-content" className="not-found container">
            <span className="eyebrow-label">404 / A little off course</span>
            <h1>This island isn’t on the map.</h1>
            <p>Let’s get you back to something useful.</p>
            <button className="button primary" onClick={() => navigate(null)}>
              Back to WeMe
              <ArrowRight size={18} />
            </button>
          </main>
        ) : (
          <motion.main
            key="home"
            id="main-content"
            initial={reduced ? false : { opacity: 0 }}
            animate={{ opacity: 1 }}
          >
            <section className="hero" aria-labelledby="hero-title">
              <img
                className="hero-landscape"
                src="/images/weme-coast.webp"
                alt="Pixelated turquoise Mediterranean cove framed by pine-covered limestone cliffs"
                fetchPriority="high"
              />
              <div className="hero-copy">
                <motion.div
                  initial={reduced ? false : { opacity: 0, y: 16 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.6 }}
                >
                  <span className="hero-badge">
                    Your everyday utility toolkit
                  </span>
                  <h1 id="hero-title">
                    Less friction.
                    <br />
                    More creating.
                  </h1>
                  <p>
                    Little tools for the things that slow you down.
                    <br className="desktop-break" /> Validate, convert, and
                    create. All in one calm place.
                  </p>
                  <button
                    className="button primary hero-cta"
                    onClick={() => goSection("tools")}
                  >
                    Meet your toolkit <ArrowRight size={19} />
                  </button>
                  <div className="hero-meta">
                    <span>No sign-up needed</span>
                    <span className="meta-dot" />
                    <span>Free to use</span>
                  </div>
                </motion.div>
              </div>
              <div className="coast-caption">
                <span>Small tools. A little breathing room.</span>
                <span>01 — The everyday collection</span>
              </div>
            </section>
            <section
              className="library-strip"
              aria-label="Built with open-source libraries"
            >
              <div className="strip-label">
                Built on good
                <br />
                foundations.
              </div>
              <div className="marquee">
                <div className="marquee-track">
                  {[0, 1].map((copy) => (
                    <div
                      className="marquee-group"
                      key={copy}
                      aria-hidden={copy === 1 ? true : undefined}
                    >
                      {[
                        { name: "AfterShip", mark: "↗" },
                        { name: "Goldmark", mark: "M↓" },
                        { name: "go-yaml", mark: "◇" },
                        { name: "Nano ID", mark: "#" },
                        { name: "regexp2", mark: "{ }" },
                        { name: "React", mark: "✳" },
                        { name: "Go", mark: "→" },
                      ].map((library) => (
                        <span className="library-wordmark" key={library.name}>
                          <b>{library.mark}</b>
                          {library.name}
                        </span>
                      ))}
                    </div>
                  ))}
                </div>
              </div>
            </section>
            <LandingIntro
              count={tools.length}
              localCount={tools.filter((item) => !item.server).length}
              onOpen={navigate}
            />
            <section
              className="tool-directory container"
              id="tools"
              aria-labelledby="tools-title"
            >
              <div className="section-heading">
                <div>
                  <span className="eyebrow-label">
                    A LITTLE HELP GOES A LONG WAY
                  </span>
                  <h2 id="tools-title">
                    A good tool for <em>every little thing.</em>
                  </h2>
                  <p>Find the one you need. Make something happen.</p>
                </div>
                <span className="tool-count">
                  {tools.length} tools, endless possibilities <ArrowDownMark />
                </span>
              </div>
              <div className="directory-controls">
                <div className="search-wrap">
                  <Search size={18} />
                  <input
                    ref={searchInput}
                    type="search"
                    placeholder="Find a tool…"
                    aria-label="Search tools"
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                  />
                  <kbd>
                    <Command size={11} /> K
                  </kbd>
                </div>
                <div className="directory-shortcuts">
                  <button
                    className={category === "Favorites" ? "active" : ""}
                    onClick={() =>
                      setCategory(
                        category === "Favorites" ? "All tools" : "Favorites",
                      )
                    }
                  >
                    <Star size={15} /> Favorites
                    {favorites.length > 0 && <span>{favorites.length}</span>}
                  </button>
                  <button
                    className={category === "Recently used" ? "active" : ""}
                    onClick={() =>
                      setCategory(
                        category === "Recently used"
                          ? "All tools"
                          : "Recently used",
                      )
                    }
                  >
                    <Clock size={15} /> Recent
                  </button>
                </div>
              </div>
              <div
                className="category-tabs"
                role="group"
                aria-label="Filter tools by category"
              >
                {(["All tools", ...categories] as const).map((name) => (
                  <button
                    key={name}
                    className={category === name ? "active" : ""}
                    onClick={() => setCategory(name)}
                    aria-pressed={category === name}
                  >
                    {name}
                    {name === "All tools" && <span>{tools.length}</span>}
                  </button>
                ))}
              </div>
              <div className="tool-grid">
                {filtered.map((item, index) => (
                  <ToolCard
                    key={item.id}
                    tool={item}
                    index={index}
                    favorite={favorites.includes(item.id)}
                    onFavorite={() => favorite(item.id)}
                    onOpen={clickTool(item.id)}
                  />
                ))}
              </div>
              {filtered.length === 0 && (
                <div className="directory-empty">
                  <Search size={28} />
                  <h3>
                    {category === "Favorites"
                      ? "Keep your favorites close."
                      : category === "Recently used"
                        ? "Your next shortcut starts here."
                        : "No tools found."}
                  </h3>
                  <p>
                    {category === "Favorites"
                      ? "Tap the star on any tool to save it here."
                      : category === "Recently used"
                        ? "Tools you use will appear here. Your inputs are never saved."
                        : "Try a different search or explore all tools."}
                  </p>
                  <button
                    className="text-button"
                    onClick={() => {
                      setQuery("");
                      setCategory("All tools");
                    }}
                  >
                    Explore all tools <ArrowRight size={15} />
                  </button>
                </div>
              )}
              <div className="directory-note">
                <ShieldCheck size={16} />{" "}
                {tools.filter((item) => !item.server).length} tools run right in
                your browser. Your data stays yours.
              </div>
            </section>
            <LandingOutro
              count={tools.length}
              localCount={tools.filter((item) => !item.server).length}
              onSection={goSection}
            />
          </motion.main>
        )}
      </AnimatePresence>
      <StudioFooter
        brand={<Brand onClick={() => navigate(null)} />}
        onSection={goSection}
        onOpen={navigate}
      />
    </>
  );
}
function ArrowDownMark() {
  return <span aria-hidden="true">↙</span>;
}
class ErrorBoundary extends Component<
  { children: ReactNode },
  { failed: boolean }
> {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("WeMe interface error", error.name, info.componentStack);
  }
  render() {
    return this.state.failed ? (
      <main className="not-found container">
        <Brand />
        <h1>A little hiccup.</h1>
        <p>
          Refresh the page to get your tools back. Your inputs aren’t stored.
        </p>
        <button className="button primary" onClick={() => location.reload()}>
          Refresh WeMe
        </button>
      </main>
    ) : (
      this.props.children
    );
  }
}
export default function App() {
  return (
    <ErrorBoundary>
      <AppContent />
    </ErrorBoundary>
  );
}

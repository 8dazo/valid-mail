import { useEffect, useRef, useState } from "react";
import type { ChangeEvent, DragEvent, MouseEvent, PointerEvent } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  ChevronLeft,
  ChevronRight,
  Download,
  FilePlus2,
  FileText,
  Highlighter,
  ImagePlus,
  Layers,
  LoaderCircle,
  MousePointer2,
  PenLine,
  RotateCw,
  Search,
  ShieldAlert,
  Signature,
  Sparkles,
  Trash2,
  Type,
  UploadCloud,
  WandSparkles,
  X,
} from "lucide-react";

type PdfAction =
  | "replace"
  | "restyle"
  | "text"
  | "sign"
  | "watermark"
  | "highlight"
  | "rectangle"
  | "rotate"
  | "extract"
  | "reorder"
  | "merge"
  | "fill"
  | "redact";
type PdfSection = "edit" | "annotate" | "pages" | "forms";
type PageInfo = { number: number; width: number; height: number };
type TextRun = {
  text: string;
  x: number;
  y: number;
  size: number;
  font: string;
};
type FormField = { name: string; type: string; value: string };
type Inspection = {
  success: boolean;
  page_count: number;
  pages: PageInfo[];
  runs: TextRun[];
  fields: FormField[];
  encrypted: boolean;
  has_signatures: boolean;
};
const sections: { id: PdfSection; label: string; description: string }[] = [
  {
    id: "edit",
    label: "Edit text",
    description: "Find and replace text already in your PDF",
  },
  {
    id: "annotate",
    label: "Annotate & sign",
    description: "Add text, signatures, notes, and marks",
  },
  {
    id: "pages",
    label: "Organize pages",
    description: "Reorder, rotate, extract and merge",
  },
  {
    id: "forms",
    label: "Forms & redaction",
    description: "Fill fields and permanently remove text",
  },
];
const actions: Record<
  PdfSection,
  { id: PdfAction; label: string; icon: typeof Type }[]
> = {
  edit: [
    { id: "replace", label: "Replace text", icon: Search },
    { id: "restyle", label: "Change formatting", icon: Type },
  ],
  annotate: [
    { id: "text", label: "Add text", icon: Type },
    { id: "sign", label: "Sign PDF", icon: Signature },
    { id: "watermark", label: "Watermark", icon: Layers },
    { id: "highlight", label: "Highlight", icon: Highlighter },
    { id: "rectangle", label: "Outline box", icon: MousePointer2 },
  ],
  pages: [
    { id: "rotate", label: "Rotate", icon: RotateCw },
    { id: "extract", label: "Extract pages", icon: FileText },
    { id: "reorder", label: "Rearrange", icon: Layers },
    { id: "merge", label: "Merge PDFs", icon: FilePlus2 },
  ],
  forms: [
    { id: "fill", label: "Fill PDF form", icon: PenLine },
    { id: "redact", label: "Redact text", icon: ShieldAlert },
  ],
};
const maxBytes = 12 * 1024 * 1024;
const placeable: PdfAction[] = ["text", "sign", "highlight", "rectangle"];
const sizeLabel = (bytes: number) =>
  bytes < 1024 * 1024
    ? `${Math.round(bytes / 1024)} KB`
    : `${(bytes / 1024 / 1024).toFixed(1)} MB`;

function useFileURL(blob: Blob | null) {
  const [url, setURL] = useState<string | null>(null);
  useEffect(() => {
    if (!blob) {
      setURL(null);
      return;
    }
    const link = URL.createObjectURL(blob);
    setURL(link);
    return () => URL.revokeObjectURL(link);
  }, [blob]);
  return url;
}
function loadPDFError(error: unknown) {
  return error instanceof Error
    ? error.message
    : "The PDF could not be processed.";
}
async function readError(response: Response) {
  try {
    const json = await response.json();
    return json.error || "The operation could not be completed.";
  } catch {
    return `Server returned HTTP ${response.status}`;
  }
}

export default function PDFStudio({
  onBack,
  onRun,
}: {
  onBack: () => void;
  onRun: () => void;
}) {
  const [file, setFile] = useState<File | null>(null);
  const [second, setSecond] = useState<File | null>(null);
  const [info, setInfo] = useState<Inspection | null>(null);
  const [page, setPage] = useState(1);
  const [section, setSection] = useState<PdfSection>("edit");
  const [action, setAction] = useState<PdfAction>("replace");
  const [loading, setLoading] = useState(false);
  const [inspecting, setInspecting] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [output, setOutput] = useState<Blob | null>(null);
  const [outputName, setOutputName] = useState("weme-edited.pdf");
  const [showOutput, setShowOutput] = useState(false);
  const [preview, setPreview] = useState<Blob | null>(null);
  const [searchText, setSearchText] = useState("");
  const [newText, setNewText] = useState("");
  const [style, setStyle] = useState("regular");
  const [fontSize, setFontSize] = useState("18");
  const [color, setColor] = useState("#463a62");
  const [x, setX] = useState("72");
  const [y, setY] = useState("100");
  const [width, setWidth] = useState("180");
  const [height, setHeight] = useState("38");
  const [allPages, setAllPages] = useState(false);
  const [pages, setPages] = useState("1");
  const [rotation, setRotation] = useState("90");
  const [password, setPassword] = useState("");
  const [formValues, setFormValues] = useState<Record<string, string>>({});
  const [signatureMode, setSignatureMode] = useState<"draw" | "type">("draw");
  const [signatureFile, setSignatureFile] = useState<File | null>(null);
  const canvas = useRef<HTMLCanvasElement>(null);
  const drawing = useRef(false);
  const upload = useRef<HTMLInputElement>(null);
  const rawPreviewURL = useFileURL(file);
  const imageURL = useFileURL(preview);
  const outputURL = useFileURL(output);
  const currentSize = info?.pages[page - 1];

  useEffect(() => {
    if (!file) {
      setInfo(null);
      setPreview(null);
      return;
    }
    const controller = new AbortController();
    const request = new FormData();
    request.set("pdf", file);
    request.set("page", String(page));
    request.set("password", password);
    setInspecting(true);
    setError("");
    void fetch("/api/pdf/inspect", {
      method: "POST",
      body: request,
      signal: controller.signal,
    })
      .then(async (response) => {
        if (!response.ok) throw new Error(await readError(response));
        return (await response.json()) as Inspection;
      })
      .then((details) => {
        setInfo(details);
        setFormValues((current) => {
          const next = { ...current };
          for (const field of details.fields) {
            if (!(field.name in next)) next[field.name] = field.value;
          }
          return next;
        });
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) setError(loadPDFError(cause));
      })
      .finally(() => {
        if (!controller.signal.aborted) setInspecting(false);
      });
    return () => controller.abort();
  }, [file, page, password]);

  useEffect(() => {
    if (!file || !info) {
      setPreview(null);
      return;
    }
    const controller = new AbortController();
    const request = new FormData();
    request.set("pdf", file);
    request.set("page", String(page));
    request.set("password", password);
    void fetch("/api/pdf/preview", {
      method: "POST",
      body: request,
      signal: controller.signal,
    })
      .then(async (response) => {
        if (!response.ok) throw new Error("Preview unavailable");
        return response.blob();
      })
      .then((blob) => {
        if (!controller.signal.aborted) setPreview(blob);
      })
      .catch(() => {
        if (!controller.signal.aborted) setPreview(null);
      });
    return () => controller.abort();
  }, [file, page, info, password]);

  function choose(fileCandidate: File | null) {
    if (!fileCandidate) return;
    if (
      fileCandidate.size > maxBytes ||
      !fileCandidate.name.toLowerCase().endsWith(".pdf")
    ) {
      setError("Choose a PDF file smaller than 12 MB.");
      return;
    }
    setPage(1);
    setInfo(null);
    setPreview(null);
    setOutput(null);
    setShowOutput(false);
    setError("");
    setNotice("");
    setPassword("");
    setFormValues({});
    setPages("1");
    setFile(fileCandidate);
  }
  function onDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    choose(event.dataTransfer.files[0] ?? null);
  }
  function place(event: MouseEvent<HTMLImageElement>) {
    if (!currentSize || !placeable.includes(action) || showOutput) return;
    const b = event.currentTarget.getBoundingClientRect();
    setX(
      Math.round(
        ((event.clientX - b.left) / b.width) * currentSize.width,
      ).toString(),
    );
    setY(
      Math.round(
        ((event.clientY - b.top) / b.height) * currentSize.height,
      ).toString(),
    );
  }
  function pointer(
    event: PointerEvent<HTMLCanvasElement>,
    kind: "down" | "move" | "up",
  ) {
    const target = canvas.current;
    const ctx = target?.getContext("2d");
    if (!target || !ctx) return;
    if (kind === "up") {
      drawing.current = false;
      return;
    }
    if (kind === "move" && !drawing.current) return;
    const rect = target.getBoundingClientRect();
    const px = ((event.clientX - rect.left) / rect.width) * target.width;
    const py = ((event.clientY - rect.top) / rect.height) * target.height;
    ctx.strokeStyle = "#3f315d";
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctx.lineWidth = 3;
    if (kind === "down") {
      target.setPointerCapture(event.pointerId);
      drawing.current = true;
      ctx.beginPath();
      ctx.moveTo(px, py);
    } else {
      ctx.lineTo(px, py);
      ctx.stroke();
    }
  }
  async function runAction() {
    if (!file || loading) return;
    setError("");
    setNotice("");
    setLoading(true);
    const request = new FormData();
    request.set("pdf", file);
    request.set("page", String(page));
    request.set("operation", action);
    request.set("password", password);
    request.set("find", searchText);
    request.set("text", newText);
    request.set("style", style);
    request.set("size", fontSize);
    request.set("color", color);
    request.set("x", x);
    request.set("y", y);
    request.set("width", width);
    request.set("height", height);
    request.set("all_pages", String(allPages));
    request.set("pages", pages);
    request.set("degrees", rotation);
    request.set("fields", JSON.stringify(formValues));
    request.set("signature_mode", signatureMode);
    if (action === "merge" && second) request.set("pdf2", second);
    if (action === "sign" && signatureMode === "draw") {
      let image: Blob | null = signatureFile;
      if (!image && canvas.current) {
        image = await new Promise<Blob | null>((resolve) =>
          canvas.current?.toBlob(resolve, "image/png"),
        );
      }
      if (image) request.set("signature", image, "signature.png");
    }
    try {
      const response = await fetch("/api/pdf/edit", {
        method: "POST",
        body: request,
      });
      if (!response.ok) throw new Error(await readError(response));
      const blob = await response.blob();
      if (blob.size < 8 || blob.type !== "application/pdf")
        throw new Error("The server did not return a PDF.");
      setOutput(blob);
      setOutputName(`weme-${action}.pdf`);
      setShowOutput(true);
      setNotice(
        "Your edited PDF is ready. Preview, download, or continue editing.",
      );
      onRun();
    } catch (cause) {
      setError(loadPDFError(cause));
    } finally {
      setLoading(false);
    }
  }

  const visibleActions = actions[section];
  const marks =
    info?.runs
      .filter((run) =>
        run.text.toLowerCase().includes(searchText.toLowerCase()),
      )
      .slice(0, 12) ?? [];

  return (
    <main className="pdf-studio container" id="main-content">
      <div className="pdf-topline">
        <button className="pdf-text-button" onClick={onBack}>
          <ArrowLeft size={16} /> All tools
        </button>
        <span className="pdf-private">
          <ShieldAlert size={15} /> Processed on WeMe's Go server · Files not
          retained
        </span>
      </div>
      <div className="pdf-heading">
        <div>
          <span className="pdf-eyebrow">
            <Sparkles size={14} /> DOCUMENT STUDIO / PDF
          </span>
          <h1>
            Make your PDF <em>your own.</em>
          </h1>
          <p>
            Edit existing text, add a handwritten signature, organize pages, and
            finish with a polished PDF.
          </p>
        </div>
        <div className="pdf-heading-mark" aria-hidden="true">
          <FileText size={47} strokeWidth={1.15} />
          <PenLine size={23} />
        </div>
      </div>

      {!file ? (
        <div
          className="pdf-drop"
          onDrop={onDrop}
          onDragOver={(event) => event.preventDefault()}
        >
          <input
            ref={upload}
            type="file"
            accept=".pdf,application/pdf"
            className="pdf-hidden"
            onChange={(e) => choose(e.target.files?.[0] ?? null)}
          />
          <div className="pdf-drop-icon">
            <UploadCloud size={34} />
          </div>
          <h2>Drop a PDF to start creating</h2>
          <p>Drag your document here, or select one from your device.</p>
          <button
            className="pdf-primary"
            onClick={() => upload.current?.click()}
          >
            <FilePlus2 size={17} /> Choose PDF
          </button>
          <small>PDF files up to 12 MB · 120 pages · No account required</small>
          {error && (
            <p className="pdf-error" role="alert">
              {error}
            </p>
          )}
        </div>
      ) : (
        <div className="pdf-layout">
          <aside className="pdf-panel">
            <div className="pdf-file">
              <div className="pdf-file-icon">
                <FileText size={21} />
              </div>
              <div className="pdf-file-name">
                <strong title={file.name}>{file.name}</strong>
                <span>
                  {sizeLabel(file.size)} · {info?.page_count ?? "…"} pages
                </span>
              </div>
              <button
                aria-label="Change PDF"
                title="Change PDF"
                onClick={() => upload.current?.click()}
              >
                <FilePlus2 size={18} />
              </button>
              <input
                ref={upload}
                type="file"
                accept=".pdf,application/pdf"
                className="pdf-hidden"
                onChange={(event) => choose(event.target.files?.[0] ?? null)}
              />
            </div>
            <div
              className="pdf-tool-sections"
              role="tablist"
              aria-label="PDF editor section"
            >
              {sections.map((item) => (
                <button
                  key={item.id}
                  role="tab"
                  aria-selected={section === item.id}
                  className={section === item.id ? "active" : ""}
                  onClick={() => {
                    setSection(item.id);
                    setAction(actions[item.id][0].id);
                    setShowOutput(false);
                  }}
                >
                  {item.label}
                </button>
              ))}
            </div>
            <div className="pdf-control-body">
              <span className="pdf-caption">
                {sections.find((item) => item.id === section)?.description}
              </span>
              <div className="pdf-operation-grid">
                {visibleActions.map((item) => {
                  const Icon = item.icon;
                  return (
                    <button
                      key={item.id}
                      className={action === item.id ? "selected" : ""}
                      onClick={() => {
                        setAction(item.id);
                        setShowOutput(false);
                      }}
                    >
                      <Icon size={18} />
                      <span>{item.label}</span>
                    </button>
                  );
                })}
              </div>
              {["replace", "restyle", "redact"].includes(action) && (
                <label className="pdf-label">
                  Find existing text
                  <input
                    value={searchText}
                    onChange={(event) => setSearchText(event.target.value)}
                    placeholder="Exact phrase in the PDF"
                  />
                </label>
              )}
              {(action === "replace" ||
                action === "text" ||
                action === "watermark" ||
                (action === "sign" && signatureMode === "type")) && (
                <label className="pdf-label">
                  {action === "replace"
                    ? "Replace with"
                    : action === "sign"
                      ? "Signature name"
                      : "Text to add"}
                  <textarea
                    rows={action === "replace" ? 3 : 2}
                    value={newText}
                    onChange={(event) => setNewText(event.target.value)}
                    placeholder={
                      action === "replace"
                        ? "Your new text"
                        : "Type something beautiful…"
                    }
                  />
                </label>
              )}
              {action === "sign" && (
                <div className="pdf-signature">
                  <div className="pdf-segment">
                    <button
                      className={signatureMode === "draw" ? "active" : ""}
                      onClick={() => setSignatureMode("draw")}
                    >
                      Draw / upload
                    </button>
                    <button
                      className={signatureMode === "type" ? "active" : ""}
                      onClick={() => setSignatureMode("type")}
                    >
                      Type
                    </button>
                  </div>
                  {signatureMode === "draw" && (
                    <>
                      <canvas
                        ref={canvas}
                        width={660}
                        height={180}
                        aria-label="Draw your handwritten signature"
                        className="pdf-sign-pad"
                        onPointerDown={(e) => pointer(e, "down")}
                        onPointerMove={(e) => pointer(e, "move")}
                        onPointerUp={(e) => pointer(e, "up")}
                        onPointerCancel={(e) => pointer(e, "up")}
                      />
                      <div className="pdf-sign-actions">
                        <button
                          onClick={() => {
                            const c = canvas.current;
                            c?.getContext("2d")?.clearRect(
                              0,
                              0,
                              c.width,
                              c.height,
                            );
                            setSignatureFile(null);
                          }}
                        >
                          <Trash2 size={13} /> Clear
                        </button>
                        <label>
                          <ImagePlus size={13} />{" "}
                          {signatureFile
                            ? signatureFile.name
                            : "Upload PNG / JPEG"}
                          <input
                            type="file"
                            accept="image/png,image/jpeg"
                            onChange={(e: ChangeEvent<HTMLInputElement>) =>
                              setSignatureFile(e.target.files?.[0] ?? null)
                            }
                          />
                        </label>
                      </div>
                    </>
                  )}
                  <p className="pdf-hint">
                    This creates a visible signature mark, not a
                    certificate-backed digital signature.
                  </p>
                </div>
              )}
              {["replace", "restyle"].includes(action) && searchText && (
                <div className="pdf-found">
                  <span>
                    {marks.length > 0
                      ? "Text found on this page"
                      : "No visible text runs matched on this page"}
                  </span>
                  {marks.map((run, index) => (
                    <button
                      key={index}
                      title={run.font}
                      onClick={() => {
                        setX(String(Math.round(run.x)));
                        setY(String(Math.round(run.y)));
                      }}
                    >
                      <Search size={12} />
                      {run.text.slice(0, 85)}{" "}
                      <small>{Math.round(run.size)}pt</small>
                    </button>
                  ))}
                  <small>Scanned PDFs may not contain selectable text.</small>
                </div>
              )}
              {["restyle", "text", "watermark"].includes(action) ||
              (action === "sign" && signatureMode === "type") ? (
                <div className="pdf-options-two">
                  <label className="pdf-label">
                    Type style
                    <select
                      value={style}
                      onChange={(e) => setStyle(e.target.value)}
                    >
                      <option value="regular">Sans serif</option>
                      <option value="bold">Bold</option>
                      <option value="italic">Italic</option>
                      <option value="bolditalic">Bold italic</option>
                      <option value="serif">Serif</option>
                    </select>
                  </label>
                  <label className="pdf-label">
                    Size (pt)
                    <input
                      type="number"
                      min="6"
                      max="120"
                      value={fontSize}
                      onChange={(e) => setFontSize(e.target.value)}
                    />
                  </label>
                  <label className="pdf-label">
                    Text color
                    <input
                      type="color"
                      value={color}
                      onChange={(e) => setColor(e.target.value)}
                    />
                  </label>
                </div>
              ) : null}
              {placeable.includes(action) && (
                <div className="pdf-position">
                  <strong>Position on page</strong>
                  <p>
                    Click on the page preview to place your mark, or enter
                    coordinates.
                  </p>
                  <div className="pdf-options-two">
                    <label className="pdf-label">
                      X (pt)
                      <input
                        type="number"
                        value={x}
                        onChange={(e) => setX(e.target.value)}
                      />
                    </label>
                    <label className="pdf-label">
                      Y (pt)
                      <input
                        type="number"
                        value={y}
                        onChange={(e) => setY(e.target.value)}
                      />
                    </label>
                    {(action === "highlight" ||
                      action === "rectangle" ||
                      (action === "sign" && signatureMode === "draw")) && (
                      <>
                        <label className="pdf-label">
                          Width (pt)
                          <input
                            type="number"
                            value={width}
                            onChange={(e) => setWidth(e.target.value)}
                          />
                        </label>
                        <label className="pdf-label">
                          Height (pt)
                          <input
                            type="number"
                            value={height}
                            onChange={(e) => setHeight(e.target.value)}
                          />
                        </label>
                      </>
                    )}
                  </div>
                </div>
              )}
              {action === "rotate" && (
                <label className="pdf-label">
                  Page rotation
                  <select
                    value={rotation}
                    onChange={(e) => setRotation(e.target.value)}
                  >
                    <option value="90">90° clockwise</option>
                    <option value="180">180°</option>
                    <option value="270">270° clockwise</option>
                    <option value="0">Reset to 0°</option>
                  </select>
                </label>
              )}
              {(action === "extract" || action === "reorder") && (
                <label className="pdf-label">
                  {action === "extract" ? "Pages to extract" : "New page order"}
                  <input
                    value={pages}
                    onChange={(e) => setPages(e.target.value)}
                    placeholder={action === "extract" ? "1-3, 5, 8" : "3, 1, 2"}
                  />
                  <span className="pdf-hint">
                    Comma-separated page numbers and ranges, e.g. 3, 1-2
                  </span>
                </label>
              )}
              {action === "merge" && (
                <label className="pdf-label pdf-file-select">
                  Add another PDF
                  <input
                    type="file"
                    accept=".pdf,application/pdf"
                    onChange={(e) => setSecond(e.target.files?.[0] ?? null)}
                  />
                  <small>
                    {second
                      ? `${second.name} · ${sizeLabel(second.size)}`
                      : "The second PDF is appended to this one."}
                  </small>
                </label>
              )}
              {action === "fill" &&
                (info?.fields.length ? (
                  <div className="pdf-form-fields">
                    {info.fields
                      .filter((field) => field.type !== "signature")
                      .map((field) => (
                        <label className="pdf-label" key={field.name}>
                          {field.name}
                          <input
                            value={formValues[field.name] ?? ""}
                            onChange={(e) =>
                              setFormValues((old) => ({
                                ...old,
                                [field.name]: e.target.value,
                              }))
                            }
                          />
                        </label>
                      ))}
                  </div>
                ) : (
                  <p className="pdf-hint">
                    This file has no interactive form fields. Use “Add text” for
                    a non-fillable PDF.
                  </p>
                ))}
              {action === "redact" && (
                <p className="pdf-caution">
                  <ShieldAlert size={17} /> This permanently removes matching
                  selectable text and strips metadata. Save a copy first. It
                  cannot remove text embedded inside scanned images.
                </p>
              )}
              {["text", "watermark", "rotate", "replace", "restyle"].includes(
                action,
              ) && (
                <label className="pdf-check">
                  <input
                    type="checkbox"
                    checked={allPages}
                    onChange={(e) => setAllPages(e.target.checked)}
                  />{" "}
                  Apply to all pages
                </label>
              )}
              {(info?.encrypted || info?.has_signatures) && (
                <p className="pdf-caution">
                  <ShieldAlert size={17} />{" "}
                  {info.has_signatures
                    ? "Editing a digitally signed document may invalidate its certificate signature."
                    : "This PDF is encrypted; its restrictions may prevent some edits."}
                </p>
              )}
              <button
                className="pdf-primary pdf-apply"
                disabled={
                  loading || inspecting || (action === "merge" && !second)
                }
                onClick={() => void runAction()}
              >
                {loading ? (
                  <LoaderCircle className="pdf-spin" size={18} />
                ) : (
                  <WandSparkles size={18} />
                )}
                {loading ? "Processing PDF…" : "Apply & create PDF"}{" "}
                <ArrowRight size={17} />
              </button>
              {error && (
                <p className="pdf-error" role="alert">
                  {error}
                </p>
              )}
              {notice && (
                <p className="pdf-success" role="status">
                  <Check size={16} />
                  {notice}
                </p>
              )}
            </div>
          </aside>
          <div className="pdf-canvas-column">
            <div className="pdf-canvas-header">
              <div>
                <span className="pdf-caption">DOCUMENT PREVIEW</span>
                <strong>{showOutput ? "Your edited PDF" : file.name}</strong>
              </div>
              <div className="pdf-canvas-tools">
                {output && (
                  <button onClick={() => setShowOutput(!showOutput)}>
                    {showOutput ? "Original" : "Result"}
                  </button>
                )}
                {output && outputURL && (
                  <a
                    className="pdf-download"
                    href={outputURL}
                    download={outputName}
                  >
                    <Download size={16} /> Download
                  </a>
                )}
              </div>
            </div>
            <div className="pdf-preview-stage">
              {inspecting && (
                <div className="pdf-overlay">
                  <LoaderCircle className="pdf-spin" /> Reading your PDF…
                </div>
              )}
              {showOutput && outputURL ? (
                <iframe
                  className="pdf-pdf-frame"
                  src={outputURL}
                  title="Edited PDF preview"
                />
              ) : imageURL ? (
                <div className="pdf-preview-sheet">
                  <img
                    src={imageURL}
                    alt={`Rendered page ${page}`}
                    onClick={place}
                    className={placeable.includes(action) ? "placeable" : ""}
                  />
                  {placeable.includes(action) && currentSize && (
                    <span
                      className="pdf-placement-pin"
                      aria-hidden="true"
                      style={{
                        left: `${(Number(x) / currentSize.width) * 100}%`,
                        top: `${(Number(y) / currentSize.height) * 100}%`,
                      }}
                    />
                  )}
                </div>
              ) : rawPreviewURL ? (
                <iframe
                  className="pdf-pdf-frame"
                  src={rawPreviewURL}
                  title="Original PDF viewer"
                />
              ) : null}
            </div>
            <div className="pdf-pagination">
              <div className="pdf-page-switch">
                <button
                  disabled={page === 1}
                  onClick={() => setPage(page - 1)}
                  aria-label="Previous page"
                >
                  <ChevronLeft size={18} />
                </button>
                <span>
                  Page <strong>{page}</strong> / {info?.page_count ?? "—"}
                </span>
                <button
                  disabled={!info || page >= info.page_count}
                  onClick={() => setPage(page + 1)}
                  aria-label="Next page"
                >
                  <ChevronRight size={18} />
                </button>
              </div>
              {currentSize && (
                <span>
                  {Math.round(currentSize.width)} ×{" "}
                  {Math.round(currentSize.height)} pt
                </span>
              )}
            </div>
            <div className="pdf-bottom-help">
              <div>
                <MousePointer2 size={19} />
                <span>
                  <strong>Click to position</strong> signatures, text, and
                  highlights on the page.
                </span>
              </div>
              <div>
                <ShieldAlert size={19} />
                <span>
                  Files are uploaded for processing and deleted after each
                  request.
                </span>
              </div>
            </div>
            {output && (
              <div className="pdf-next">
                <div>
                  <Check size={19} />
                  <span>
                    <strong>Looks good?</strong>
                    <small>Keep editing or save the new PDF.</small>
                  </span>
                </div>
                <button
                  onClick={() => {
                    choose(
                      new File([output], outputName, {
                        type: "application/pdf",
                      }),
                    );
                    setNotice("Edited PDF loaded for another pass.");
                  }}
                >
                  Continue editing <ArrowRight size={15} />
                </button>
              </div>
            )}
          </div>
        </div>
      )}
      <footer className="pdf-footer">
        <span>
          Powered by{" "}
          <a
            href="https://github.com/SalvioniDigitalSolutions/gopdf"
            target="_blank"
            rel="noreferrer"
          >
            gopdf
          </a>{" "}
          · Pure Go PDF operations
        </span>
        <span>
          Visual signatures are not cryptographic digital signatures · Advanced
          edits depend on the PDF's embedded fonts
        </span>
      </footer>
    </main>
  );
}

import { tools } from "./catalog";
export type ToolResult = {
  text: string;
  title?: string;
  filename?: string;
  mime?: string;
  image?: string;
  imageAlt?: string;
  binary?: { base64: string; mime: string; filename: string };
  language?: string;
  table?: { columns: string[]; rows: string[][] };
  swatches?: { name: string; color: string }[];
  segments?: { value: string; matched?: boolean }[];
  html?: string;
  diff?: { value: string; added?: boolean; removed?: boolean }[];
  facts?: { label: string; value: string; tone?: "good" | "warn" | "bad" }[];
  notice?: string;
  smtp?: {
    confidence: number | null;
    evidence: string;
    details: { label: string; value: string }[];
  };
};
const pretty = (value: unknown) => JSON.stringify(value, null, 2);
const MAX_INPUT = 100_000;

export async function runTool(
  id: string,
  values: Record<string, string>,
  signal?: AbortSignal,
): Promise<ToolResult> {
  const input = values.input ?? "";
  if (Object.values(values).some((value) => value.length > MAX_INPUT))
    throw new Error("Please keep each input under 100,000 characters.");
  if (
    ![
      "id",
      "password",
      "hash",
      "base64",
      "diff",
      "markdown",
      "ulid",
      "bcrypt",
      "text-stats",
      "regex",
      "hmac",
      "gzip",
    ].includes(id) &&
    !input.trim()
  )
    throw new Error("Add your input first, or try the example.");
  if (tools.find((tool) => tool.id === id)?.engine === "go") {
    const { runGoTool } = await import("./go-tools");
    return runGoTool(id, values, signal);
  }
  switch (id) {
    case "finder": {
      if (!values.company?.trim() && !values.domain?.trim())
        throw new Error("Add a company name or domain.");
      const response = await fetch("/api/find-work-email", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: input.trim(),
          company: values.company?.trim(),
          domain: values.domain?.trim(),
        }),
        signal: signal ?? AbortSignal.timeout(45_000),
      });
      const data = await response.json();
      if (!response.ok || !data.success)
        throw new Error(
          data.error || "Discovery is unavailable. Please try again.",
        );
      return {
        table: {
          columns: ["Candidate email", "Domain check", "Evidence"],
          rows: (data.candidates ?? []).map(
            (candidate: {
              email: string;
              validity?: { status?: string };
              evidence?: string;
              source?: string;
            }) => [
              candidate.email,
              candidate.validity?.status ?? "Not checked",
              candidate.evidence ?? candidate.source ?? "Pattern suggestion",
            ],
          ),
        },
        text: pretty(data),
        filename: "work-email-candidates.json",
        mime: "application/json",
        title: "Company email candidates",
        facts: [
          { label: "Domain", value: data.domain ?? "Unknown" },
          {
            label: "Public emails",
            value: String(data.observed_email_count ?? 0),
          },
          { label: "Pages read", value: String(data.pages_fetched ?? 0) },
          { label: "Pattern", value: data.pattern ?? "Not established" },
        ],
        notice: data.note,
      };
    }
    case "email": {
      const response = await fetch("/api/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: input.trim(),
          smtp: values.depth !== "domain",
        }),
        signal: signal ?? AbortSignal.timeout(45_000),
      });
      const data = await response.json();
      if (!response.ok || !data.success)
        throw new Error(
          data.error || "Email verification is unavailable. Please try again.",
        );
      const v = data.verification;
      const status = data.validity?.status ?? "unknown";
      const reach = data.reachability;
      const confidence = Number(reach?.confidence);
      const mailboxTitle =
        reach?.status === "valid"
          ? "Mailbox reachable"
          : reach?.status === "invalid"
            ? "Mailbox rejected"
            : reach?.status === "risky"
              ? "Mailbox needs review"
              : "Address valid; mailbox unconfirmed";
      return {
        text: pretty(data),
        filename: "email-report.json",
        mime: "application/json",
        title:
          status === "unknown"
            ? "Verification inconclusive"
            : status === "invalid"
              ? "Email invalid"
              : mailboxTitle,
        smtp: {
          confidence:
            reach?.status !== "unknown" && Number.isFinite(confidence)
              ? Math.max(0, Math.min(100, confidence))
              : null,
          evidence:
            reach?.evidence ??
            "Mailbox verification did not provide conclusive evidence.",
          details: [
            {
              label: "SMTP check",
              value: data.smtp_attempted
                ? "Attempted"
                : data.smtp_enabled
                  ? "No connection attempted"
                  : "Disabled",
            },
            {
              label: "Route",
              value: data.smtp_attempted
                ? data.smtp_via_proxy
                  ? "SOCKS5 proxy"
                  : "Direct SMTP"
                : "Not established",
            },
            {
              label: "Provider / MX",
              value:
                [reach?.provider, reach?.mx_host].filter(Boolean).join(" / ") ||
                "Not established",
            },
            {
              label: "Server reply",
              value: reach?.target?.code
                ? `${reach.target.code} ${reach.target.enhanced_code ?? ""} ${reach.target.message ?? ""}`.trim()
                : "No recipient reply",
            },
            {
              label: "Catch-all",
              value:
                reach?.catch_all === true
                  ? "Yes — individual mailbox unconfirmed"
                  : reach?.catch_all === false
                    ? "No"
                    : "Not established",
            },
            {
              label: "Routes / negative confirmations",
              value: `${reach?.attempts ?? 0} / ${reach?.negative_consensus ?? 0}`,
            },
          ],
        },
        facts: [
          {
            label: "Address syntax",
            value: v?.syntax?.valid ? "Valid" : "Invalid",
            tone: v?.syntax?.valid ? "good" : "bad",
          },
          {
            label: "Domain validity",
            value: status,
            tone:
              status === "valid"
                ? "good"
                : status === "invalid"
                  ? "bad"
                  : "warn",
          },
          {
            label: "MX records",
            value:
              status === "unknown"
                ? "Unavailable"
                : v?.has_mx_records
                  ? "Found"
                  : v?.disposable
                    ? "Not checked"
                    : "Not found",
            tone:
              status === "unknown"
                ? "warn"
                : v?.has_mx_records
                  ? "good"
                  : "warn",
          },
          {
            label: "Disposable",
            value: v?.disposable ? "Yes" : "No",
            tone: v?.disposable ? "warn" : "good",
          },
          { label: "Role account", value: v?.role_account ? "Yes" : "No" },
          {
            label: "Checked at",
            value: data.checked_at
              ? new Date(data.checked_at).toLocaleString()
              : "Not supplied",
          },
          {
            label: "Duration",
            value:
              data.duration_ms !== undefined
                ? `${(data.duration_ms / 1000).toFixed(2)} seconds`
                : "Not supplied",
          },
          {
            label: "Mailbox",
            value: data.reachability?.status ?? "Unknown",
            tone: "warn",
          },
        ],
        notice: data.warning || data.validity?.evidence,
      };
    }
    default:
      throw new Error("This tool is not available.");
  }
}

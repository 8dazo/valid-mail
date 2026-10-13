export type ToolResult = {
  text: string;
  title?: string;
  filename?: string;
  mime?: string;
  image?: string;
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
function integer(
  value: string | undefined,
  fallback: number,
  min: number,
  max: number,
) {
  const n = value === undefined || value === "" ? fallback : Number(value);
  if (!Number.isInteger(n) || n < min || n > max)
    throw new Error(`Enter a whole number between ${min} and ${max}.`);
  return n;
}
function chooseIndex(max: number) {
  const limit = 256 - (256 % max);
  const byte = new Uint8Array(1);
  do {
    crypto.getRandomValues(byte);
  } while (byte[0] >= limit);
  return byte[0] % max;
}
export function securePassword(length: number): string {
  if (!Number.isInteger(length) || length < 8 || length > 128)
    throw new Error("Password length must be between 8 and 128.");
  const groups = [
    "abcdefghijkmnopqrstuvwxyz",
    "ABCDEFGHJKLMNPQRSTUVWXYZ",
    "23456789",
    "!@#$%&*+-=?",
  ];
  const alphabet = groups.join("");
  const chars = groups.map((group) => group[chooseIndex(group.length)]);
  while (chars.length < length)
    chars.push(alphabet[chooseIndex(alphabet.length)]);
  for (let i = chars.length - 1; i > 0; i--) {
    const j = chooseIndex(i + 1);
    [chars[i], chars[j]] = [chars[j], chars[i]];
  }
  return chars.join("");
}

export async function runTool(
  id: string,
  values: Record<string, string>,
  signal?: AbortSignal,
): Promise<ToolResult> {
  const input = values.input ?? "";
  if (Object.values(values).some((value) => value.length > MAX_INPUT))
    throw new Error("Please keep each input under 100,000 characters.");
  if (
    !["id", "password", "hash", "base64", "diff", "markdown"].includes(id) &&
    !input.trim()
  )
    throw new Error("Add your input first, or try the example.");
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
        body: JSON.stringify({ email: input.trim() }),
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
            label: "Mailbox",
            value: data.reachability?.status ?? "Unknown",
            tone: "warn",
          },
        ],
        notice: data.warning || data.validity?.evidence,
      };
    }
    case "phone": {
      const { parsePhoneNumberFromString } =
        await import("libphonenumber-js/max");
      const phone = parsePhoneNumberFromString(
        input,
        values.country as import("libphonenumber-js").CountryCode,
      );
      if (!phone)
        throw new Error(
          "That number could not be parsed. Include a country code, such as +44.",
        );
      const valid = phone.isValid();
      return {
        text: pretty({
          valid,
          possible: phone.isPossible(),
          country: phone.country ?? "Non-geographic",
          international: phone.formatInternational(),
          national: phone.formatNational(),
          e164: phone.number,
          type: phone.getType() ?? "Unknown",
          extension: phone.ext ?? null,
        }),
        title: valid ? "Valid phone format" : "Invalid phone format",
        facts: [
          {
            label: "Validation",
            value: valid ? "Valid" : "Invalid",
            tone: valid ? "good" : "bad",
          },
          { label: "Country", value: phone.country ?? "Non-geographic" },
          { label: "International", value: phone.formatInternational() },
          { label: "E.164", value: phone.number },
        ],
        notice:
          "Format validation does not prove this number is active or belongs to a person.",
      };
    }
    case "url": {
      const { default: isURL } = await import("validator/lib/isURL");
      if (
        !isURL(input, {
          require_protocol: true,
          protocols: ["http", "https"],
          require_valid_protocol: true,
          disallow_auth: true,
        })
      )
        throw new Error(
          "Enter a valid HTTP or HTTPS URL without embedded credentials.",
        );
      const url = new URL(input);
      return {
        text: pretty({
          protocol: url.protocol,
          hostname: url.hostname,
          port: url.port || (url.protocol === "https:" ? "443" : "80"),
          path: url.pathname,
          query: [...url.searchParams].map(([key, value]) => ({ key, value })),
          fragment: url.hash,
          origin: url.origin,
        }),
        facts: [
          {
            label: "Protocol",
            value: url.protocol.replace(":", ""),
            tone: url.protocol === "https:" ? "good" : "warn",
          },
          { label: "Hostname", value: url.hostname },
          { label: "Path", value: url.pathname },
          {
            label: "Query parameters",
            value: String([...url.searchParams].length),
          },
        ],
      };
    }
    case "ip": {
      const { default: ipaddr } = await import("ipaddr.js");
      let address: ReturnType<typeof ipaddr.parse>;
      let prefix: number | undefined;
      if (input.includes("/"))
        [address, prefix] = ipaddr.parseCIDR(input.trim());
      else address = ipaddr.parse(input.trim());
      return {
        text: pretty({
          version: address.kind(),
          normalized: address.toString(),
          range: address.range(),
          prefix: prefix ?? null,
          subnetMask:
            prefix === undefined
              ? null
              : address.kind() === "ipv4"
                ? ipaddr.IPv4.subnetMaskFromPrefixLength(prefix).toString()
                : ipaddr.IPv6.subnetMaskFromPrefixLength(prefix).toString(),
          network:
            prefix === undefined
              ? null
              : address.kind() === "ipv4"
                ? ipaddr.IPv4.networkAddressFromCIDR(input.trim()).toString()
                : ipaddr.IPv6.networkAddressFromCIDR(input.trim()).toString(),
        }),
        facts: [
          { label: "Version", value: address.kind().toUpperCase() },
          { label: "Range", value: address.range() },
          { label: "Normalized", value: address.toString() },
          {
            label: "Prefix",
            value: prefix === undefined ? "Not supplied" : `/${prefix}`,
          },
        ],
      };
    }
    case "json": {
      const { jsonrepair } = await import("jsonrepair");
      const parsed = JSON.parse(
        values.mode === "repair" ? jsonrepair(input) : input,
      );
      return {
        text: JSON.stringify(
          parsed,
          null,
          values.mode === "minify" ? undefined : 2,
        ),
        filename: "formatted.json",
        mime: "application/json",
        notice:
          values.mode === "repair"
            ? "Review repaired output before using it. Repairs can change how your input is interpreted."
            : undefined,
      };
    }
    case "yaml": {
      const YAML = await import("yaml");
      if (values.mode === "json-yaml")
        return {
          text: YAML.stringify(JSON.parse(input)),
          filename: "converted.yaml",
          mime: "application/yaml",
        };
      const parsed = YAML.parse(input, { maxAliasCount: 50, uniqueKeys: true });
      return {
        text: pretty(parsed),
        filename: "converted.json",
        mime: "application/json",
      };
    }
    case "xml": {
      const { XMLParser, XMLBuilder, XMLValidator } =
        await import("fast-xml-parser");
      if (values.mode === "json-xml")
        return {
          text: new XMLBuilder({ format: true, ignoreAttributes: false }).build(
            JSON.parse(input),
          ),
          filename: "converted.xml",
          mime: "application/xml",
        };
      if (/<!\s*(DOCTYPE|ENTITY)\b/i.test(input))
        throw new Error(
          "DTD and entity declarations are not supported. Remove them before converting.",
        );
      const valid = XMLValidator.validate(input);
      if (valid !== true) throw new Error(`Invalid XML: ${valid.err.msg}`);
      return {
        text: pretty(
          new XMLParser({
            ignoreAttributes: false,
            processEntities: false,
          }).parse(input),
        ),
        filename: "converted.json",
        mime: "application/json",
      };
    }
    case "csv": {
      const { default: Papa } = await import("papaparse");
      if (values.mode === "json-csv") {
        const data = JSON.parse(input);
        if (
          !Array.isArray(data) ||
          !data.length ||
          data.some(
            (row) => !row || typeof row !== "object" || Array.isArray(row),
          )
        )
          throw new Error("Use a non-empty JSON array of objects.");
        return {
          text: Papa.unparse(data, { escapeFormulae: true }),
          filename: "converted.csv",
          mime: "text/csv",
        };
      }
      const parsed = Papa.parse(input, {
        header: true,
        skipEmptyLines: "greedy",
      });
      if (parsed.errors.length)
        throw new Error(`CSV could not be parsed: ${parsed.errors[0].message}`);
      return {
        text: pretty(parsed.data),
        filename: "converted.json",
        mime: "application/json",
      };
    }
    case "jwt": {
      const { jwtDecode } = await import("jwt-decode");
      const token = input.trim();
      if (token.split(".").length !== 3)
        throw new Error("A signed JWT must contain three dot-separated parts.");
      const header = jwtDecode(token, { header: true });
      const payload = jwtDecode<import("jwt-decode").JwtPayload>(token);
      const expiry = payload.exp
        ? new Date(payload.exp * 1000).toISOString()
        : null;
      return {
        text: pretty({ header, payload, expiresAt: expiry }),
        notice:
          "Decoded only — the signature and authenticity have NOT been verified.",
        facts: [
          { label: "Signature", value: "Not verified", tone: "warn" },
          { label: "Expires", value: expiry ?? "Not supplied" },
          {
            label: "Expiry status",
            value: !payload.exp
              ? "Unknown"
              : payload.exp * 1000 < Date.now()
                ? "Expired"
                : "Not expired",
            tone: "warn",
          },
        ],
      };
    }
    case "cron": {
      const { default: cronstrue } = await import("cronstrue");
      return {
        text: cronstrue.toString(input.trim(), {
          throwExceptionOnParseError: true,
          use24HourTimeFormat: true,
          verbose: true,
        }),
        title: "Your schedule, explained",
        notice: "Timezone and actual execution depend on your scheduler.",
      };
    }
    case "qr": {
      if (input.length > 2500)
        throw new Error(
          "For a readable QR code, keep your content under 2,500 characters.",
        );
      const { default: QRCode } = await import("qrcode");
      const svg = await QRCode.toString(input, {
        type: "svg",
        errorCorrectionLevel: (values.correction ||
          "M") as import("qrcode").QRCodeErrorCorrectionLevel,
        margin: 4,
        color: { dark: "#173d2b", light: "#ffffff" },
      });
      return {
        text: svg,
        image: `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`,
        filename: "weme-qr.svg",
        mime: "image/svg+xml",
        title: "Ready to scan",
      };
    }
    case "markdown": {
      const [{ marked }, { default: DOMPurify }] = await Promise.all([
        import("marked"),
        import("dompurify"),
      ]);
      const html = DOMPurify.sanitize(
        await marked.parse(input, { async: true }),
        {
          ALLOWED_TAGS: [
            "h1",
            "h2",
            "h3",
            "h4",
            "h5",
            "h6",
            "p",
            "br",
            "hr",
            "strong",
            "em",
            "del",
            "blockquote",
            "ul",
            "ol",
            "li",
            "pre",
            "code",
            "a",
            "table",
            "thead",
            "tbody",
            "tr",
            "th",
            "td",
          ],
          ALLOWED_ATTR: ["href", "title"],
          ALLOW_DATA_ATTR: false,
        },
      );
      return { text: html, html, filename: "preview.html", mime: "text/html" };
    }
    case "diff": {
      if (input.length + (values.second?.length ?? 0) > 20_000)
        throw new Error(
          "Keep compared texts under 20,000 characters combined.",
        );
      const { diffWordsWithSpace } = await import("diff");
      const diff = diffWordsWithSpace(input, values.second ?? "", {
        timeout: 2000,
      });
      if (!diff)
        throw new Error(
          "These texts are too complex to compare. Try a smaller section.",
        );
      const changed = diff.some((part) => part.added || part.removed);
      return {
        diff,
        text: diff
          .map(
            (part) =>
              `${part.added ? "[+] " : part.removed ? "[-] " : ""}${part.value}`,
          )
          .join(""),
        title: changed ? "Here’s what changed" : "These texts are identical",
      };
    }
    case "id": {
      const { nanoid } = await import("nanoid");
      const count = integer(values.count, 5, 1, 100);
      return {
        text: Array.from({ length: count }, () =>
          values.mode === "nanoid" ? nanoid() : crypto.randomUUID(),
        ).join("\n"),
        filename: "identifiers.txt",
      };
    }
    case "password": {
      const length = integer(values.length, 24, 8, 128);
      const count = integer(values.count, 3, 1, 20);
      return {
        text: Array.from({ length: count }, () => securePassword(length)).join(
          "\n",
        ),
        title: "Fresh, strong passwords",
        notice:
          "Store these in a password manager. They are not saved by WeMe.",
      };
    }
    case "hash": {
      const { sha256, sha512 } = await import("@noble/hashes/sha2.js");
      const bytes = new TextEncoder().encode(input);
      const hash = values.mode === "sha512" ? sha512(bytes) : sha256(bytes);
      return {
        text: Array.from(hash, (byte) =>
          byte.toString(16).padStart(2, "0"),
        ).join(""),
        title:
          values.mode === "sha512"
            ? "SHA-512 fingerprint"
            : "SHA-256 fingerprint",
      };
    }
    case "base64": {
      if (values.mode === "decode") {
        const value = input.replace(/\s/g, "");
        if (!/^[A-Za-z0-9+/]*={0,2}$/.test(value) || value.length % 4 === 1)
          throw new Error("That is not valid Base64 text.");
        const binary = atob(value);
        const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
        return {
          text: new TextDecoder("utf-8", { fatal: true }).decode(bytes),
        };
      }
      const bytes = new TextEncoder().encode(input);
      let binary = "";
      for (const byte of bytes) binary += String.fromCharCode(byte);
      return { text: btoa(binary) };
    }
    case "timestamp": {
      const value = input.trim();
      const numeric = /^-?\d+(\.\d+)?$/.test(value);
      if (
        !numeric &&
        !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d{1,3})?)?(Z|[+-]\d{2}:\d{2})$/.test(
          value,
        )
      )
        throw new Error(
          "Use a Unix number or an ISO date with a timezone, such as 2026-01-01T00:00:00Z.",
        );
      const date = new Date(
        numeric
          ? Number(value) * (values.unit === "milliseconds" ? 1 : 1000)
          : value,
      );
      if (!Number.isFinite(date.getTime()))
        throw new Error("That timestamp is outside the supported date range.");
      if (!numeric) {
        const [year, month, day] = value.slice(0, 10).split("-").map(Number);
        if (
          month < 1 ||
          month > 12 ||
          day < 1 ||
          day > new Date(Date.UTC(year, month, 0)).getUTCDate()
        )
          throw new Error("That date does not exist.");
      }
      return {
        text: pretty({
          unixSeconds: Math.floor(date.getTime() / 1000),
          unixMilliseconds: date.getTime(),
          iso: date.toISOString(),
          utc: date.toUTCString(),
          local: date.toLocaleString(),
          localTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        }),
        facts: [
          { label: "ISO 8601", value: date.toISOString() },
          {
            label: "Unix seconds",
            value: String(Math.floor(date.getTime() / 1000)),
          },
          { label: "Unix milliseconds", value: String(date.getTime()) },
        ],
      };
    }
    case "case": {
      const change = await import("change-case");
      return {
        text: pretty({
          camelCase: change.camelCase(input),
          PascalCase: change.pascalCase(input),
          snake_case: change.snakeCase(input),
          "kebab-case": change.kebabCase(input),
          CONSTANT_CASE: change.constantCase(input),
          "Title Case": change.capitalCase(input),
          "Sentence case": change.sentenceCase(input),
          lowercase: input.toLowerCase(),
          UPPERCASE: input.toUpperCase(),
        }),
      };
    }
    case "url-codec": {
      try {
        return {
          text:
            values.mode === "decode"
              ? decodeURIComponent(input)
              : encodeURIComponent(input),
        };
      } catch {
        throw new Error(
          "Invalid percent encoding or Unicode sequence. Check the input and try again.",
        );
      }
    }
    default:
      throw new Error("This tool is not available.");
  }
}

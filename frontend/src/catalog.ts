export type Category =
  "Validation" | "Conversion" | "Generators" | "Developer" | "Text";
export type Field = {
  key: string;
  label: string;
  type?: "text" | "textarea" | "select" | "number";
  placeholder?: string;
  defaultValue?: string;
  options?: [string, string][];
};
export type Tool = {
  id: string;
  name: string;
  description: string;
  category: Category;
  icon: string;
  color: string;
  action: string;
  fields: Field[];
  example: Record<string, string>;
  source?: string;
  library: string;
  note?: string;
  featured?: boolean;
  server?: boolean;
};
const text = (label = "Input", key = "input", placeholder = ""): Field => ({
  key,
  label,
  type: "textarea",
  placeholder,
});
const select = (
  key: string,
  label: string,
  options: [string, string][],
  defaultValue?: string,
): Field => ({
  key,
  label,
  type: "select",
  options,
  defaultValue: defaultValue ?? options[0][0],
});
const github = (repo: string) => `https://github.com/${repo}`;

export const tools: Tool[] = [
  {
    id: "email",
    name: "Email validator",
    description: "Check email syntax, mail routing, and disposable domains.",
    category: "Validation",
    icon: "MailCheck",
    color: "green",
    action: "Verify email",
    featured: true,
    server: true,
    library: "AfterShip / email-verifier",
    source: github("AfterShip/email-verifier"),
    fields: [
      { key: "input", label: "Email address", placeholder: "you@company.com" },
    ],
    example: { input: "hello@gmail.com" },
    note: "Your address is sent to this server for DNS checks. A valid domain does not prove that a mailbox exists.",
  },
  {
    id: "json",
    name: "JSON formatter",
    description: "Format, minify, and repair your imperfect JSON.",
    category: "Developer",
    icon: "Braces",
    color: "orange",
    action: "Format JSON",
    featured: true,
    library: "jsonrepair",
    source: github("josdejong/jsonrepair"),
    fields: [
      text("JSON input", "input", '{ "hello": "world" }'),
      select("mode", "Operation", [
        ["format", "Pretty print"],
        ["minify", "Minify"],
        ["repair", "Repair & format"],
      ]),
    ],
    example: { input: "{name: 'WeMe', tools: 20,}", mode: "repair" },
  },
  {
    id: "qr",
    name: "QR code generator",
    description: "Turn any link or text into a crisp, downloadable QR code.",
    category: "Generators",
    icon: "QrCode",
    color: "purple",
    action: "Generate QR code",
    featured: true,
    library: "node-qrcode",
    source: github("soldair/node-qrcode"),
    fields: [
      text("Link or text", "input", "https://example.com"),
      select("correction", "Error correction", [
        ["M", "Medium · 15%"],
        ["L", "Low · 7%"],
        ["Q", "Quartile · 25%"],
        ["H", "High · 30%"],
      ]),
    ],
    example: { input: "https://example.com", correction: "M" },
  },
  {
    id: "password",
    name: "Password generator",
    description: "Create strong passwords with secure browser randomness.",
    category: "Generators",
    icon: "KeyRound",
    color: "pink",
    action: "Generate passwords",
    featured: true,
    library: "Web Crypto API",
    fields: [
      {
        key: "length",
        label: "Password length (8–128)",
        type: "number",
        defaultValue: "24",
      },
      {
        key: "count",
        label: "Number of passwords (1–20)",
        type: "number",
        defaultValue: "3",
      },
    ],
    example: { length: "24", count: "3" },
    note: "Generated locally with cryptographic randomness. These passwords are never saved to your history.",
  },
  {
    id: "phone",
    name: "Phone validator",
    description: "Validate and format international phone numbers.",
    category: "Validation",
    icon: "Phone",
    color: "blue",
    action: "Check number",
    library: "libphonenumber-js",
    source: github("catamphetamine/libphonenumber-js"),
    fields: [
      { key: "input", label: "Phone number", placeholder: "+44 20 7946 0018" },
      select("country", "Default country", [
        ["US", "United States"],
        ["IN", "India"],
        ["GB", "United Kingdom"],
        ["CA", "Canada"],
        ["AU", "Australia"],
        ["DE", "Germany"],
        ["FR", "France"],
        ["SG", "Singapore"],
      ]),
    ],
    example: { input: "+44 20 7946 0018", country: "GB" },
  },
  {
    id: "url",
    name: "URL inspector",
    description: "Validate a link and explore its individual components.",
    category: "Validation",
    icon: "Link",
    color: "green",
    action: "Inspect URL",
    library: "validator.js",
    source: github("validatorjs/validator.js"),
    fields: [
      {
        key: "input",
        label: "URL",
        placeholder: "https://example.com/search?q=hello",
      },
    ],
    example: {
      input: "https://example.com:8443/search?q=hello&lang=en#results",
    },
    note: "Inspects the URL locally. No request is made to the destination.",
  },
  {
    id: "ip",
    name: "IP address inspector",
    description: "Understand IPv4, IPv6, CIDR ranges, and address types.",
    category: "Validation",
    icon: "Network",
    color: "purple",
    action: "Inspect address",
    library: "ipaddr.js",
    source: github("whitequark/ipaddr.js"),
    fields: [
      {
        key: "input",
        label: "IP address or CIDR",
        placeholder: "192.168.1.42/24",
      },
    ],
    example: { input: "192.168.1.42/24" },
  },
  {
    id: "yaml",
    name: "JSON ↔ YAML",
    description: "Switch effortlessly between JSON and YAML documents.",
    category: "Conversion",
    icon: "FileJson",
    color: "orange",
    action: "Convert document",
    library: "yaml",
    source: github("eemeli/yaml"),
    fields: [
      select("mode", "Direction", [
        ["yaml-json", "YAML → JSON"],
        ["json-yaml", "JSON → YAML"],
      ]),
      text("Document"),
    ],
    example: {
      input: "name: WeMe\ntools: 20\nprivate: true",
      mode: "yaml-json",
    },
  },
  {
    id: "xml",
    name: "XML ↔ JSON",
    description: "Parse XML or transform JSON into readable XML.",
    category: "Conversion",
    icon: "CodeXml",
    color: "blue",
    action: "Convert document",
    library: "fast-xml-parser",
    source: github("NaturalIntelligence/fast-xml-parser"),
    fields: [
      select("mode", "Direction", [
        ["xml-json", "XML → JSON"],
        ["json-xml", "JSON → XML"],
      ]),
      text("Document"),
    ],
    example: {
      input: "<toolkit><name>WeMe</name><tools>20</tools></toolkit>",
      mode: "xml-json",
    },
    note: "DTD and entity declarations are rejected. No external resources are loaded.",
  },
  {
    id: "csv",
    name: "CSV ↔ JSON",
    description: "Convert spreadsheets and structured data in seconds.",
    category: "Conversion",
    icon: "Table",
    color: "green",
    action: "Convert data",
    library: "Papa Parse",
    source: github("mholt/PapaParse"),
    fields: [
      select("mode", "Direction", [
        ["csv-json", "CSV → JSON"],
        ["json-csv", "JSON → CSV"],
      ]),
      text("Data"),
    ],
    example: {
      input: "name,category\nEmail validator,Validation\nQR code,Generators",
      mode: "csv-json",
    },
    note: "CSV exports escape spreadsheet formula cells. Input must include a header row.",
  },
  {
    id: "jwt",
    name: "JWT decoder",
    description: "Read token headers and payloads without sending them away.",
    category: "Developer",
    icon: "Fingerprint",
    color: "pink",
    action: "Decode token",
    library: "jwt-decode",
    source: github("auth0/jwt-decode"),
    fields: [text("JSON Web Token", "input", "eyJhbGciOiJIUzI1NiJ9…")],
    example: {
      input:
        "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJkZW1vIiwibmFtZSI6IldlTWUifQ.ZGVtby1zaWduYXR1cmU",
    },
    note: "Decoding does not verify a signature, issuer, or authenticity. Tokens are never saved to history.",
  },
  {
    id: "cron",
    name: "Cron explainer",
    description: "Translate schedule expressions into plain English.",
    category: "Developer",
    icon: "CalendarClock",
    color: "purple",
    action: "Explain schedule",
    library: "cRonstrue",
    source: github("bradymholt/cRonstrue"),
    fields: [
      { key: "input", label: "Cron expression", placeholder: "0 9 * * 1-5" },
    ],
    example: { input: "0 9 * * 1-5" },
    note: "Explains cron syntax only. The timezone is set by your scheduler.",
  },
  {
    id: "markdown",
    name: "Markdown preview",
    description: "Write Markdown and preview clean, sanitized HTML.",
    category: "Text",
    icon: "FileText",
    color: "blue",
    action: "Preview Markdown",
    library: "Marked + DOMPurify",
    source: github("markedjs/marked"),
    fields: [text("Markdown", "input", "# Hello, world")],
    example: {
      input:
        "# Small tools. Big possibilities.\n\nMake everyday work **a little easier**.\n\n- Validate\n- Convert\n- Create",
    },
  },
  {
    id: "diff",
    name: "Text compare",
    description: "Spot what changed between two pieces of text.",
    category: "Text",
    icon: "GitCompareArrows",
    color: "orange",
    action: "Compare text",
    library: "jsdiff",
    source: github("kpdecker/jsdiff"),
    fields: [text("Original text", "input"), text("Updated text", "second")],
    example: {
      input: "Small tools. Good ideas.\nMade for everyday work.",
      second: "Small tools. Big possibilities.\nMade for your everyday work.",
    },
  },
  {
    id: "id",
    name: "UUID & Nano ID",
    description: "Generate unique identifiers for your next project.",
    category: "Generators",
    icon: "ScanLine",
    color: "blue",
    action: "Generate IDs",
    library: "Nano ID + Web Crypto",
    source: github("ai/nanoid"),
    fields: [
      select("mode", "ID format", [
        ["uuid", "UUID v4"],
        ["nanoid", "Nano ID · 21 characters"],
      ]),
      {
        key: "count",
        label: "Number of IDs (1–100)",
        type: "number",
        defaultValue: "5",
      },
    ],
    example: { mode: "uuid", count: "5" },
  },
  {
    id: "hash",
    name: "Hash generator",
    description: "Calculate SHA-256 and SHA-512 fingerprints for text.",
    category: "Developer",
    icon: "Hash",
    color: "green",
    action: "Generate hash",
    library: "@noble/hashes",
    source: github("paulmillr/noble-hashes"),
    fields: [
      text("Text to hash"),
      select("mode", "Algorithm", [
        ["sha256", "SHA-256"],
        ["sha512", "SHA-512"],
      ]),
    ],
    example: { input: "Hello, WeMe!", mode: "sha256" },
    note: "A hash is not encryption. Plain SHA hashes are not suitable for storing passwords.",
  },
  {
    id: "base64",
    name: "Base64 encoder",
    description: "Encode or decode text, including Unicode characters.",
    category: "Conversion",
    icon: "Binary",
    color: "purple",
    action: "Convert text",
    library: "Browser encoding APIs",
    fields: [
      select("mode", "Direction", [
        ["encode", "Text → Base64"],
        ["decode", "Base64 → Text"],
      ]),
      text("Input"),
    ],
    example: { input: "Hello, 世界 👋", mode: "encode" },
  },
  {
    id: "timestamp",
    name: "Timestamp converter",
    description: "Make sense of Unix timestamps and ISO dates.",
    category: "Conversion",
    icon: "Clock",
    color: "pink",
    action: "Convert time",
    library: "JavaScript Date / Intl",
    fields: [
      {
        key: "input",
        label: "Timestamp or ISO date",
        placeholder: "2026-01-01T00:00:00Z",
      },
      select("unit", "Numeric timestamp unit", [
        ["seconds", "Seconds"],
        ["milliseconds", "Milliseconds"],
      ]),
    ],
    example: { input: "2026-01-01T00:00:00Z", unit: "seconds" },
    note: "Use an ISO date with a timezone (Z or ±hh:mm) for an unambiguous result.",
  },
  {
    id: "case",
    name: "Text case converter",
    description: "Transform text into camelCase, snake_case, and more.",
    category: "Text",
    icon: "CaseSensitive",
    color: "orange",
    action: "Convert case",
    library: "change-case",
    source: github("blakeembrey/change-case"),
    fields: [text("Text")],
    example: { input: "small tools big possibilities" },
  },
  {
    id: "url-codec",
    name: "URL encoder",
    description: "Encode query values or decode percent-encoded text.",
    category: "Conversion",
    icon: "Link",
    color: "blue",
    action: "Convert URL text",
    library: "Browser URI APIs",
    fields: [
      select("mode", "Direction", [
        ["encode", "Text → URL component"],
        ["decode", "URL component → Text"],
      ]),
      text("Input"),
    ],
    example: { input: "hello world & café", mode: "encode" },
  },
  {
    id: "finder",
    name: "Work email finder",
    description: "Discover email candidates from public company pages.",
    category: "Validation",
    icon: "SearchCheck",
    color: "green",
    action: "Find candidates",
    library: "AfterShip + Go public-page crawler",
    source: github("AfterShip/email-verifier"),
    server: true,
    fields: [
      { key: "input", label: "Person’s full name", placeholder: "Jane Smith" },
      { key: "company", label: "Company name", placeholder: "Acme Labs" },
      {
        key: "domain",
        label: "Company domain (optional)",
        placeholder: "acme.com",
      },
    ],
    example: {
      input: "Jane Example",
      company: "Example",
      domain: "example.com",
    },
    note: "Name, company, and domain are sent to this server. Candidate addresses are suggestions, not proof of mailbox ownership.",
  },
];
export const categories: Category[] = [
  "Validation",
  "Conversion",
  "Generators",
  "Developer",
  "Text",
];
export function initialValues(tool: Tool): Record<string, string> {
  return Object.fromEntries(
    tool.fields.map((field) => [field.key, field.defaultValue ?? ""]),
  );
}

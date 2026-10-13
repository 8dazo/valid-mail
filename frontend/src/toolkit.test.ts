import { execFileSync } from "node:child_process";
import { resolve } from "node:path";
import { describe, expect, it, vi } from "vitest";
import { runTool } from "./toolkit";
import { tools } from "./catalog";
import { readList, saveList } from "./storage";

// Exercise the real Go core through its native CLI in jsdom. Browser tests use
// the actual WebAssembly worker; no duplicate tool implementation is mocked.
vi.mock("./go-tools", () => ({
  runGoTool: async (tool: string, values: Record<string, string>) => {
    const binary = resolve(process.cwd(), "../bin/weme-toolkit");
    const response = JSON.parse(
      execFileSync(binary, {
        input: JSON.stringify({ tool, values }) + "\n",
        encoding: "utf8",
        timeout: 10_000,
      }),
    );
    if (response.error) throw new Error(response.error);
    return response.result;
  },
}));

describe("the complete tool collection", () => {
  it("has 39 unique tools with working local Go examples", async () => {
    expect(tools.length).toBe(39);
    expect(new Set(tools.map((tool) => tool.id)).size).toBe(tools.length);
    for (const tool of tools.filter((tool) => !tool.server)) {
      const result = await runTool(tool.id, tool.example);
      expect(result.text, tool.name).toBeTypeOf("string");
      expect(result.text.length, tool.name).toBeGreaterThan(0);
    }
  }, 30_000);
});
describe("data conversion", () => {
  it("formats and repairs JSON, and rejects malformed JSON in strict mode", async () => {
    expect(
      JSON.parse(
        (await runTool("json", { input: "{foo: 'bar',}", mode: "repair" }))
          .text,
      ),
    ).toEqual({ foo: "bar" });
    expect(
      (await runTool("json", { input: '{ "a": 1 }', mode: "minify" })).text,
    ).toBe('{"a":1}');
    await expect(
      runTool("json", { input: "{", mode: "format" }),
    ).rejects.toThrow();
  });
  it("round-trips YAML and rejects duplicate keys and alias bombs", async () => {
    const yaml = await runTool("yaml", {
      input: '{"name":"WeMe","tools":21}',
      mode: "json-yaml",
    });
    expect(
      JSON.parse(
        (await runTool("yaml", { input: yaml.text, mode: "yaml-json" })).text,
      ),
    ).toEqual({ name: "WeMe", tools: 21 });
    await expect(
      runTool("yaml", { input: "a: 1\na: 2", mode: "yaml-json" }),
    ).rejects.toThrow();
    const bomb =
      "a: &a [x, x, x, x, x, x, x, x, x, x]\nb: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a,*a]\nc: [*b,*b,*b,*b,*b,*b,*b,*b,*b,*b]";
    await expect(
      runTool("yaml", { input: bomb, mode: "yaml-json" }),
    ).rejects.toThrow();
  });
  it("converts XML and blocks document entity declarations", async () => {
    const result = await runTool("xml", {
      input: "<root><item>hello</item></root>",
      mode: "xml-json",
    });
    expect(JSON.parse(result.text)).toEqual({ root: { item: "hello" } });
    expect(
      (await runTool("xml", { input: result.text, mode: "json-xml" })).text,
    ).toContain("<item>hello</item>");
    await expect(
      runTool("xml", { input: "<root>", mode: "xml-json" }),
    ).rejects.toThrow();
    await expect(
      runTool("xml", {
        input:
          '<!DOCTYPE foo [<!ENTITY x SYSTEM "file:///etc/passwd">]><foo>&x;</foo>',
        mode: "xml-json",
      }),
    ).rejects.toThrow("entity");
  });
  it("handles quoted CSV and prevents exported spreadsheet formula injection", async () => {
    const json = await runTool("csv", {
      input: 'name,note\nJane,"hello, world"',
      mode: "csv-json",
    });
    expect(JSON.parse(json.text)).toEqual([
      { name: "Jane", note: "hello, world" },
    ]);
    const csv = await runTool("csv", {
      input: JSON.stringify([{ name: '=HYPERLINK("https://bad.example")' }]),
      mode: "json-csv",
    });
    expect(csv.text).toContain("'=HYPERLINK");
    await expect(
      runTool("csv", { input: "{}", mode: "json-csv" }),
    ).rejects.toThrow("array");
  });
  it("round-trips multilingual Base64 and rejects invalid bytes", async () => {
    const text = "Hello, 世界 👋";
    const encoded = await runTool("base64", { input: text, mode: "encode" });
    expect(
      (await runTool("base64", { input: encoded.text, mode: "decode" })).text,
    ).toBe(text);
    await expect(
      runTool("base64", { input: "%%%bad", mode: "decode" }),
    ).rejects.toThrow();
    await expect(
      runTool("base64", { input: "/w==", mode: "decode" }),
    ).rejects.toThrow();
  });
  it("converts timestamps exactly and rejects ambiguous and nonexistent dates", async () => {
    expect(
      JSON.parse(
        (await runTool("timestamp", { input: "0", unit: "seconds" })).text,
      ).iso,
    ).toBe("1970-01-01T00:00:00.000Z");
    expect(
      JSON.parse(
        (await runTool("timestamp", { input: "1000", unit: "milliseconds" }))
          .text,
      ).unixSeconds,
    ).toBe(1);
    await expect(
      runTool("timestamp", { input: "2026-02-30T00:00:00Z" }),
    ).rejects.toThrow("exist");
    await expect(runTool("timestamp", { input: "01/02/2026" })).rejects.toThrow(
      "timezone",
    );
  });
  it("round-trips URL components and rejects malformed percent encoding", async () => {
    const encoded = await runTool("url-codec", {
      input: "hello world & café",
      mode: "encode",
    });
    expect(encoded.text).toBe("hello%20world%20%26%20caf%C3%A9");
    expect(
      (await runTool("url-codec", { input: encoded.text, mode: "decode" }))
        .text,
    ).toBe("hello world & café");
    await expect(
      runTool("url-codec", { input: "%ZZ", mode: "decode" }),
    ).rejects.toThrow();
  });
});
describe("validation and inspection", () => {
  it("validates international numbers and separates format from ownership", async () => {
    const result = await runTool("phone", {
      input: "+44 20 7946 0018",
      country: "GB",
    });
    expect(JSON.parse(result.text)).toMatchObject({
      valid: true,
      country: "GB",
      e164: "+442079460018",
    });
    expect(result.notice).toContain("does not prove");
  });
  it("parses repeated URL query parameters without fetching a destination", async () => {
    const fetch = vi.spyOn(globalThis, "fetch");
    const result = await runTool("url", {
      input: "https://example.com/?a=1&a=2",
    });
    expect(JSON.parse(result.text).query).toEqual([
      { key: "a", value: "1" },
      { key: "a", value: "2" },
    ]);
    expect(fetch).not.toHaveBeenCalled();
    await expect(
      runTool("url", { input: "javascript:alert(1)" }),
    ).rejects.toThrow();
    await expect(
      runTool("url", { input: "https://user:pass@example.com" }),
    ).rejects.toThrow();
  });
  it("inspects IPv4 and IPv6 network prefixes", async () => {
    expect(
      JSON.parse((await runTool("ip", { input: "192.168.1.42/24" })).text),
    ).toMatchObject({
      version: "ipv4",
      network: "192.168.1.0",
      subnetMask: "255.255.255.0",
      range: "private",
    });
    expect(
      JSON.parse((await runTool("ip", { input: "2001:db8::1/64" })).text),
    ).toMatchObject({ version: "ipv6", network: "2001:db8::" });
    await expect(runTool("ip", { input: "999.0.0.1" })).rejects.toThrow();
  });
  it("decodes JWT without claiming signature verification", async () => {
    const jwt = tools.find((tool) => tool.id === "jwt")!;
    const result = await runTool("jwt", jwt.example);
    expect(JSON.parse(result.text).payload.name).toBe("WeMe");
    expect(result.notice).toContain("NOT been verified");
    await expect(runTool("jwt", { input: "broken" })).rejects.toThrow();
  });
  it("explains a valid cron schedule and rejects malformed syntax", async () => {
    expect((await runTool("cron", { input: "0 9 * * 1-5" })).text).toContain(
      "09:00",
    );
    await expect(runTool("cron", { input: "not cron" })).rejects.toBeDefined();
  });
});
describe("generators and text tools", () => {
  it("generates secure passwords with every character group and enforces limits", async () => {
    const passwords = [
      ...(await runTool("password", { length: "24", count: "20" })).text.split(
        "\n",
      ),
      ...(await runTool("password", { length: "24", count: "10" })).text.split(
        "\n",
      ),
    ];
    expect(new Set(passwords).size).toBe(30);
    for (const password of passwords) {
      expect(password).toHaveLength(24);
      expect(password).toMatch(/[a-z]/);
      expect(password).toMatch(/[A-Z]/);
      expect(password).toMatch(/[2-9]/);
      expect(password).toMatch(/[!@#$%&*+\-=?]/);
    }
    await expect(
      runTool("password", { length: "7", count: "1" }),
    ).rejects.toThrow();
    await expect(
      runTool("password", { length: "24", count: "21" }),
    ).rejects.toThrow();
  });
  it("generates the requested unique UUIDs and Nano IDs", async () => {
    for (const mode of ["uuid", "nanoid"]) {
      const ids = (await runTool("id", { mode, count: "10" })).text.split("\n");
      expect(ids).toHaveLength(10);
      expect(new Set(ids).size).toBe(10);
      expect(ids[0]).toMatch(
        mode === "uuid"
          ? /^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/
          : /^[\w-]{21}$/,
      );
    }
    await expect(runTool("id", { count: "1000" })).rejects.toThrow();
  });
  it("matches known SHA-256 and SHA-512 vectors", async () => {
    expect((await runTool("hash", { input: "abc", mode: "sha256" })).text).toBe(
      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
    );
    expect((await runTool("hash", { input: "abc", mode: "sha512" })).text).toBe(
      "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f",
    );
  });
  it("creates downloadable QR SVGs and rejects oversize content", async () => {
    const result = await runTool("qr", {
      input: "https://example.com",
      correction: "M",
    });
    expect(result.text).toContain("<svg");
    expect(result.image).toMatch(/^data:image\/svg\+xml/);
    expect(result.filename).toBe("weme-qr.svg");
    await expect(runTool("qr", { input: "x".repeat(2501) })).rejects.toThrow();
  });
  it("sanitizes script, event, SVG, and unsafe link attacks from Markdown", async () => {
    const result = await runTool("markdown", {
      input:
        "# Safe\n<script>alert(1)</script><img src=x onerror=alert(1)><svg onload=alert(1)></svg>\n[bad](javascript:alert%281%29)",
    });
    expect(
      new DOMParser()
        .parseFromString(result.html ?? "", "text/html")
        .querySelector("h1")?.textContent,
    ).toBe("Safe");
    expect(result.html).not.toMatch(
      /<script|<img|<svg|onerror|onload|href="javascript:/i,
    );
  });
  it("identifies added and removed words and converts text case", async () => {
    const result = await runTool("diff", {
      input: "hello world",
      second: "hello WeMe",
    });
    expect(
      result.diff?.some((part) => part.removed && part.value.includes("world")),
    ).toBe(true);
    expect(
      result.diff?.some((part) => part.added && part.value.includes("WeMe")),
    ).toBe(true);
    expect(
      JSON.parse((await runTool("case", { input: "hello world" })).text),
    ).toMatchObject({
      camelCase: "helloWorld",
      snake_case: "hello_world",
      "kebab-case": "hello-world",
    });
  });
});
describe("boundaries, API failures, and storage", () => {
  it("rejects oversized inputs, empty required values, and unknown tools", async () => {
    await expect(
      runTool("json", { input: "x".repeat(100_001) }),
    ).rejects.toThrow("100,000");
    await expect(runTool("phone", { input: " " })).rejects.toThrow("input");
    await expect(runTool("nonexistent", { input: "test" })).rejects.toThrow(
      "available",
    );
  });
  it("renders DNS failures as inconclusive and surfaces API errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          success: true,
          verification: { syntax: { valid: true }, has_mx_records: false },
          validity: { status: "unknown" },
          reachability: { status: "unknown" },
          warning: "DNS unavailable",
        }),
      }),
    );
    const result = await runTool("email", { input: "hello@gmail.com" });
    expect(result.title).toBe("Verification inconclusive");
    expect(
      result.facts?.find((fact) => fact.label === "MX records")?.value,
    ).toBe("Unavailable");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        json: async () => ({ error: "Too many checks" }),
      }),
    );
    await expect(
      runTool("email", { input: "hello@gmail.com" }),
    ).rejects.toThrow("Too many");
    vi.unstubAllGlobals();
  });
  it("stores tool IDs only and tolerates unavailable or malformed storage", () => {
    saveList("test", ["json", "qr"]);
    expect(readList("test")).toEqual(["json", "qr"]);
    localStorage.setItem("test", "{broken");
    expect(readList("test")).toEqual([]);
    localStorage.setItem("test", '["json",12,null]');
    expect(readList("test")).toEqual(["json"]);
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("Storage unavailable");
    });
    expect(() => saveList("test", ["json"])).not.toThrow();
  });
});

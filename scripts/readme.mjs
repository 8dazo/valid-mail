import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { tools, categories } from "../frontend/src/catalog.ts";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const read = (file) => fs.readFileSync(path.join(root, file), "utf8");
const manifest = JSON.parse(read("frontend/package.json"));
const go = read("go.mod").match(/^toolchain go(\S+)$/m)?.[1];
const node = read(".node-version").trim();
if (!go) throw new Error("Missing pinned Go toolchain in go.mod.");
const safe = (value) => value.replaceAll("|", "\\|").replaceAll("\n", " ");
const browserTools = tools.filter((tool) => !tool.server).length;
const sections = {
  overview: `**${tools.length} utilities · ${browserTools} run in your browser · ${tools.length - browserTools} use the Go API**\n\nWeMe is a calm place to validate, convert, and create. It brings practical open-source tools into consistent workspaces, with a coastal landing page, self-hosted fonts, and a React + TypeScript interface.`,
  stack: `| Frontend | Backend | Build |\n| --- | --- | --- |\n| React ${manifest.dependencies.react} · TypeScript ${manifest.devDependencies.typescript} | Go ${go} | Node ${node} · Vite ${manifest.devDependencies.vite} |`,
  tools: `| Category | Tools |\n| --- | --- |\n${categories.map((category) => `| ${category} | ${tools.filter((tool) => tool.category === category).map((tool) => safe(tool.name)).join(" · ")} |`).join("\n")}\n\n${browserTools} tools execute locally. Email validation and work-email discovery call the same-origin API.`,
};
const filename = path.join(root, "README.md");
const original = fs.readFileSync(filename, "utf8");
let updated = original;
for (const [key, content] of Object.entries(sections)) {
  const start = `<!-- generated:${key}:start -->`;
  const end = `<!-- generated:${key}:end -->`;
  const from = updated.indexOf(start);
  const to = updated.indexOf(end);
  if (from < 0 || to < from || updated.indexOf(start, from + 1) !== -1 || updated.indexOf(end, to + 1) !== -1) {
    throw new Error(`README needs exactly one ordered marker pair for ${key}.`);
  }
  updated = updated.slice(0, from + start.length) + `\n${content}\n` + updated.slice(to);
}
const mode = process.argv[2] ?? "--check";
if (mode === "--write") {
  if (updated !== original) fs.writeFileSync(filename, updated);
  console.log("README tool catalog, counts, and stack versions updated.");
} else if (mode === "--check") {
  if (updated !== original) {
    console.error("README is out of date. Run npm run docs:update and commit the result.");
    process.exitCode = 1;
  } else console.log("README matches the tool catalog and pinned versions.");
} else throw new Error("Usage: node scripts/readme.mjs --check|--write");

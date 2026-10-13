"""Publish complete notices for the actually shipped npm/Go runtime packages."""
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]


def objects(text):
    decoder = json.JSONDecoder()
    while text.strip():
        text = text.lstrip()
        value, end = decoder.raw_decode(text)
        text = text[end:]
        yield value


def licenses(directory):
    return sorted(path for path in directory.iterdir() if path.is_file() and path.name.lower().startswith(("license", "copying", "ofl")))


def main():
    records = []
    package_root = ROOT / "frontend/node_modules"
    manifest = json.loads((ROOT / "frontend/package.json").read_text())
    queue = list(manifest["dependencies"])
    visited = set()
    while queue:
        name = queue.pop()
        if name in visited:
            continue
        visited.add(name)
        directory = package_root / name
        package = json.loads((directory / "package.json").read_text())
        queue.extend(package.get("dependencies", {}))
        notices = licenses(directory)
        if not notices:
            raise RuntimeError(f"Missing npm license notice for {name}")
        records.append((f"npm: {name}@{package['version']}", package.get("license", "See notice"), f"https://www.npmjs.com/package/{name}", notices))

    data = subprocess.check_output(["go", "list", "-deps", "-json", "./cmd/weme-toolkit", "."], cwd=ROOT, text=True)
    modules = {}
    for package in objects(data):
        module = package.get("Module")
        if module and module.get("Version"):
            modules[module["Path"]] = module
    for name, module in modules.items():
        directory = Path(module["Dir"])
        notices = licenses(directory)
        if not notices:
            raise RuntimeError(f"Missing Go license notice for {name}")
        if any("GNU GENERAL PUBLIC LICENSE" in path.read_text(errors="replace") for path in notices):
            raise RuntimeError(f"Review copyleft licensing before shipping {name}")
        records.append((f"Go: {name}@{module['Version']}", "See distributed notice", f"https://pkg.go.dev/{name}@{module['Version']}", notices))
    goroot = Path(subprocess.check_output(["go", "env", "GOROOT"], text=True).strip())
    records.append(("Go standard library and WebAssembly runtime", "BSD-3-Clause", "https://github.com/golang/go", [goroot / "LICENSE"]))
    output = ["WeMe third-party runtime notices", "", "This file includes original license texts. It does not grant a separate license for WeMe.", ""]
    table = ["# Third-party software in WeMe", "", "Runtime package notices are bundled in [third-party-licenses.txt](../frontend/public/third-party-licenses.txt). Rebuild them after dependency updates with `python3 scripts/update-licenses.py` (Go and installed frontend dependencies required).", "", "| Package/version | License | Reference |", "| --- | --- | --- |"]
    for label, license, link, files in sorted(records):
        output += ["=" * 80, label, link, ""]
        for notice in files:
            output += [notice.name, "", notice.read_text(errors="replace"), ""]
        table.append(f"| {label} | {license} | {link} |")
    text = "\n".join(line.rstrip() for line in "\n".join(output).splitlines()).rstrip() + "\n"
    (ROOT / "frontend/public/third-party-licenses.txt").write_text(text)
    table += ["", "Instrument Serif and DM Sans use the SIL Open Font License. Their original notices are included above. The WeMe SVG mark and generated coastal artwork are application assets.", "", "Competitor implementations are not bundled. See [research and selection notes](competitor-research.md) for the source-based comparison and license decisions.", ""]
    (ROOT / "docs/third-party.md").write_text("\n".join(table))
    print(f"Updated {len(records)} runtime package/standard-library notices.")


if __name__ == "__main__":
    main()

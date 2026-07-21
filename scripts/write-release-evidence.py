#!/usr/bin/env python3
"""Bundle successful CI E2E and DAST evidence for a GitHub release."""
import argparse
import json
import shutil
import sys
from datetime import datetime, timezone
from pathlib import Path


def load_reports(root, name):
    reports = []
    for path in sorted(Path(root).glob(f"*/{name}")):
        try:
            reports.append((path.parent.name, json.loads(path.read_text(encoding="utf-8-sig"))))
        except (OSError, json.JSONDecodeError) as error:
            raise RuntimeError(f"cannot read {path}: {error}") from error
    return reports


def validate_e2e(reports):
    if not reports:
        raise RuntimeError("no E2E evidence reports were provided")
    result = []
    for name, report in reports:
        summary = report.get("summary", {})
        passed = int(summary.get("pass", 0))
        failed = int(summary.get("fail", 0))
        skipped = int(summary.get("skip", 0))
        if report.get("status") != "passed" or not passed or failed or skipped:
            raise RuntimeError(f"E2E evidence {name} is not a complete success: {summary}")
        result.append({"job": name, "passed": passed, "failed": failed, "skipped": skipped, "suite": report.get("suite", "")})
    return result


def validate_dast(reports):
    if not reports:
        raise RuntimeError("no DAST evidence reports were provided")
    result = []
    for name, report in reports:
        counts = report.get("counts", {})
        high = int(counts.get("High", 0))
        critical = int(counts.get("Critical", 0))
        if report.get("status") != "passed" or high or critical or report.get("blocking_alerts"):
            raise RuntimeError(f"DAST evidence {name} contains blocking findings: {counts}")
        result.append({"job": name, "threshold": report.get("threshold", "High"), "counts": counts})
    return result


def copy_source_evidence(output, source_root, filename, prefix):
    destination = output / "source-evidence" / prefix
    destination.mkdir(parents=True, exist_ok=True)
    for source in sorted(Path(source_root).glob(f"*/{filename}")):
        shutil.copy2(source, destination / f"{source.parent.name}-{filename}")
        markdown = source.with_suffix(".md")
        if markdown.exists():
            shutil.copy2(markdown, destination / f"{source.parent.name}-{markdown.name}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--e2e-root", required=True)
    parser.add_argument("--dast-root", required=True)
    parser.add_argument("--output-dir", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--commit", required=True)
    args = parser.parse_args()

    e2e_reports = load_reports(args.e2e_root, "e2e-evidence.json")
    # Negative DAST probes use the same protected-service E2E harness. Include
    # their result in the release evidence alongside the regular E2E stage.
    e2e_reports.extend(load_reports(args.dast_root, "e2e-evidence.json"))
    e2e = validate_e2e(e2e_reports)
    dast = validate_dast(load_reports(args.dast_root, "dast-evidence.json"))
    output = Path(args.output_dir)
    output.mkdir(parents=True, exist_ok=True)
    copy_source_evidence(output, args.e2e_root, "e2e-evidence.json", "e2e")
    copy_source_evidence(output, args.dast_root, "e2e-evidence.json", "dast-e2e")
    copy_source_evidence(output, args.dast_root, "dast-evidence.json", "dast")
    shutil.make_archive(str(output / "release-evidence-source"), "zip", output / "source-evidence")

    report = {
        "schema_version": 1,
        "version": args.version,
        "commit": args.commit,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "status": "passed",
        "e2e": e2e,
        "dast": dast,
        "assertions": [
            "All attached E2E suites passed without failures or skipped tests.",
            "DAST found no High or Critical alerts in the disposable E2E WAF runtime.",
            "The source evidence archive contains redacted, machine-readable reports.",
        ],
    }
    (output / "release-evidence.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    e2e_table = "\n".join(
        f"| `{item['job']}` | {item['passed']} | {item['failed']} | {item['skipped']} |"
        for item in e2e
    )
    dast_table = "\n".join(
        f"| `{item['job']}` | {item['counts'].get('High', 0)} | {item['counts'].get('Critical', 0)} | passed |"
        for item in dast
    )
    (output / "release-evidence-summary.md").write_text(f"""### WAF verification summary

| E2E suite | Passed | Failed | Skipped |
| --- | ---: | ---: | ---: |
{e2e_table}

| DAST suite | High | Critical | Status |
| --- | ---: | ---: | --- |
{dast_table}

All E2E suites passed without failures or skipped tests. DAST found no blocking High or Critical vulnerabilities. Full machine-readable evidence is attached to this release.
""", encoding="utf-8")
    e2e_rows = "\n".join(f"- `{item['job']}`: пройдено {item['passed']}; ошибок {item['failed']}; пропусков {item['skipped']}." for item in e2e)
    dast_rows = "\n".join(f"- `{item['job']}`: High {item['counts'].get('High', 0)}, Critical {item['counts'].get('Critical', 0)}." for item in dast)
    (output / "release-evidence.md").write_text(f"""# Доказательства проверок релиза {args.version}

**Коммит:** `{args.commit}`
**Статус:** **пройдено**

### E2E-проверки

{e2e_rows}

### DAST

{dast_rows}

DAST запускался только против одноразового E2E-контура WAF. Во всех приложенных отчётах отсутствуют блокирующие находки уровня High и Critical.

### Исходные доказательства

`release-evidence-source.zip` содержит переименованные JSON и Markdown отчёты E2E/DAST без секретов и тел запросов.
""", encoding="utf-8")


if __name__ == "__main__":
    try:
        main()
    except RuntimeError as error:
        print(f"release evidence validation failed: {error}", file=sys.stderr)
        sys.exit(1)

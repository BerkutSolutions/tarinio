#!/usr/bin/env python3
"""Render a safe bilingual DAST summary and fail only on High/Critical alerts."""
import argparse
import json
import sys
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

RISK = {0: "Informational", 1: "Low", 2: "Medium", 3: "High", 4: "Critical"}


def alerts(document):
    for site in document.get("site", []):
        yield from site.get("alerts", [])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--output-dir", required=True)
    parser.add_argument("--mode", required=True)
    parser.add_argument("--max-risk", type=int, default=3)
    parser.add_argument("--policy")
    args = parser.parse_args()
    source = Path(args.input)
    data = json.loads(source.read_text(encoding="utf-8-sig", errors="replace"))
    found = list(alerts(data))
    policy = {}
    if args.policy:
        policy = json.loads(Path(args.policy).read_text(encoding="utf-8-sig"))
    accepted = set(policy.get("accepted_alerts", []))
    counts = Counter(int(item.get("riskcode", 0)) for item in found)
    blocking = [item for item in found if int(item.get("riskcode", 0)) >= args.max_risk and item.get("alert") not in accepted]
    status = "passed" if not blocking else "failed"
    report = {
        "schema_version": 1,
        "mode": args.mode,
        "target_scope": "disposable E2E WAF runtime only",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "status": status,
        "threshold": RISK[args.max_risk],
        "policy": {"accepted_alerts": sorted(accepted), "review_policy": policy.get("review_policy", "")},
        "counts": {RISK[key]: counts[key] for key in sorted(RISK)},
        "blocking_alerts": [{"name": item.get("alert", "unknown"), "risk": RISK.get(int(item.get("riskcode", 0)), "Unknown")} for item in blocking],
    }
    out = Path(args.output_dir)
    out.mkdir(parents=True, exist_ok=True)
    (out / "dast-evidence.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    summary = "\n".join(f"- {risk}: {count}" for risk, count in report["counts"].items())
    blocked = "\n".join(f"- {item['name']} ({item['risk']})" for item in report["blocking_alerts"]) or "- None / Нет"
    (out / "dast-evidence.md").write_text(f"""# DAST evidence report / Отчёт-доказательство DAST

**Mode / Режим:** `{args.mode}`<br>
**Scope / Контур:** disposable E2E WAF runtime only / только одноразовый E2E runtime WAF<br>
**Status / Статус:** **{status}**<br>
**Blocking threshold / Порог блокировки:** {report['threshold']}+

## Alert counts / Количество предупреждений

{summary}

## Blocking alerts / Блокирующие предупреждения

{blocked}

The raw ZAP reports are attached separately; no production endpoint was scanned.<br>
Исходные отчёты ZAP приложены отдельно; production endpoint не сканировался.
""", encoding="utf-8")
    if blocking:
        print(f"DAST blocked: {len(blocking)} alert(s) at {report['threshold']} or above", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Create portable, bilingual evidence reports from Go E2E JSON output."""
import argparse
import json
import os
import re
import subprocess
from datetime import datetime, timezone
from pathlib import Path


def run_git(*args):
    try:
        return subprocess.check_output(["git", *args], text=True).strip()
    except Exception:
        return "unknown"


def outcomes(path):
    final = {}
    if not path.exists():
        return final
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            event = json.loads(raw)
        except json.JSONDecodeError:
            continue
        name, action = event.get("Test", ""), event.get("Action", "")
        if name.startswith("TestE2E") and action in {"pass", "fail", "skip"}:
            final[name] = action
    return final


def extract_runtime_evidence(path):
    raw = path.read_text(encoding="utf-8", errors="replace") if path.exists() else ""
    return {
        "revision_ids": sorted(set(re.findall(r"\brev-[0-9]+\b", raw))),
        "http_statuses": sorted(set(re.findall(r"\bstatus[=: ]+(\d{3})\b", raw, re.I))),
        "blocking_reasons": sorted(set(re.findall(r"\bsecurity_reason[=: ]+[\"']?([a-z0-9_.-]+)", raw, re.I))),
    }


def runtime_config_checksum():
    try:
        out = subprocess.check_output(["docker", "exec", "waf-e2e-runtime", "sh", "-lc", "find /etc/waf/nginx -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum"], text=True, stderr=subprocess.DEVNULL)
        return out.split()[0]
    except Exception:
        return "unavailable"


def runtime_adaptive_evidence():
    commands = {
        "adaptive_entries": ["docker", "exec", "waf-e2e-runtime", "sh", "-lc", "cat /etc/waf/l4guard-adaptive/adaptive.json 2>/dev/null || true"],
        "iptables_rules": ["docker", "exec", "waf-e2e-runtime", "iptables", "-S", "WAF-RUNTIME-L4"],
    }
    result = {}
    for name, command in commands.items():
        try:
            value = subprocess.check_output(command, text=True, stderr=subprocess.DEVNULL).strip()
            result[name] = value[:12000] if value else "unavailable"
        except Exception:
            result[name] = "unavailable"
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--log", required=True)
    parser.add_argument("--output-dir", required=True)
    parser.add_argument("--suite", required=True)
    args = parser.parse_args()
    log_path = Path(args.log)
    result = outcomes(log_path)
    counts = {key: sum(value == key for value in result.values()) for key in ("pass", "fail", "skip")}
    report = {
        "schema_version": 1,
        "suite": args.suite,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "commit": os.getenv("CI_COMMIT_SHA") or run_git("rev-parse", "HEAD"),
        "pipeline_url": os.getenv("CI_PIPELINE_URL", "local"),
        "status": "passed" if counts["fail"] == 0 else "failed",
        "summary": counts,
        "tests": [{"name": name, "status": status} for name, status in sorted(result.items())],
        "runtime_evidence": {
            **extract_runtime_evidence(log_path),
            "runtime_config_checksum": runtime_config_checksum(),
            **runtime_adaptive_evidence(),
        },
        "security_invariants": [
            "Authentication rejects disabled users and rotated credentials / Аутентификация отклоняет отключённых пользователей и сменённые учётные данные.",
            "Authentication sessions and credentials are isolated between sites / Сессии и учётные данные изолированы между сайтами.",
            "Configured protection modes take effect after compile and apply / Настроенные режимы защиты вступают в силу после compile/apply.",
            "Security-relevant requests are persisted for dashboard evidence / Значимые для безопасности запросы сохраняются как доказательства для дашборда.",
        ],
    }
    out = Path(args.output_dir)
    out.mkdir(parents=True, exist_ok=True)
    (out / "e2e-evidence.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    tests = "\n".join(f"- `{item['name']}` — **{item['status']}**" for item in report["tests"]) or "- No matching E2E test events / Нет подходящих событий E2E."
    markdown = f"""# E2E evidence report / Отчёт-доказательство E2E

**Suite / Набор:** `{args.suite}`<br>
**Status / Статус:** **{report['status']}**<br>
**Commit / Коммит:** `{report['commit']}`<br>
**Passed / Пройдено:** {counts['pass']} · **Failed / Ошибки:** {counts['fail']} · **Skipped / Пропущено:** {counts['skip']}

## Runtime evidence / Доказательства runtime

**Revision IDs:** {', '.join(report['runtime_evidence']['revision_ids']) or 'unavailable'}<br>
**HTTP statuses:** {', '.join(report['runtime_evidence']['http_statuses']) or 'unavailable'}<br>
**Blocking reasons:** {', '.join(report['runtime_evidence']['blocking_reasons']) or 'unavailable'}<br>
**Runtime-config checksum:** `{report['runtime_evidence']['runtime_config_checksum']}`

L4/L7 adaptive entries and `WAF-RUNTIME-L4` iptables rules are preserved in the JSON companion when available.

## Security invariants / Инварианты безопасности

""" + "\n".join(f"- {item}" for item in report["security_invariants"]) + f"""

## Test evidence / Результаты проверок

{tests}

The JSON companion is machine-readable and contains no credentials or request bodies.<br>
JSON-файл машиночитаем и не содержит учётных данных или тел запросов.
"""
    (out / "e2e-evidence.md").write_text(markdown, encoding="utf-8")


if __name__ == "__main__":
    main()

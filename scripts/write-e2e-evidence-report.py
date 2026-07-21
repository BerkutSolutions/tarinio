#!/usr/bin/env python3
"""Create portable, bilingual evidence reports from Go E2E JSON output."""
import argparse
import json
import os
import re
import subprocess
import urllib.request
from http.cookiejar import CookieJar
from datetime import datetime, timezone
from pathlib import Path


def run_git(*args):
    try:
        return subprocess.check_output(["git", *args], text=True).strip()
    except Exception:
        return "unknown"


def test_evidence(path):
    final, observations = {}, {}
    if not path.exists():
        return final
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            event = json.loads(raw)
        except json.JSONDecodeError:
            continue
        name, action = event.get("Test", ""), event.get("Action", "")
        if not (name.startswith("TestE2E") or name.startswith("TestFreshOnboarding")):
            continue
        if action in {"pass", "fail", "skip"}:
            final[name] = action
        if action == "output":
            line = event.get("Output", "").strip()
            lowered = line.lower()
            if (line and len(line) <= 500 and
                    not any(secret in lowered for secret in ("password", "authorization", "cookie", "token", "secret")) and
                    ("status=" in lowered or "http/" in lowered or "revision" in lowered or "checksum" in lowered)):
                observations.setdefault(name, []).append(line)
    return final, observations


def extract_runtime_evidence(path):
    raw = path.read_text(encoding="utf-8", errors="replace") if path.exists() else ""
    return {
        "revision_ids": sorted(set(re.findall(r"\brev-[0-9]+\b", raw))),
        "http_statuses": sorted(set(re.findall(r"\bstatus[=: ]+(\d{3})\b", raw, re.I))),
        "blocking_reasons": sorted(set(re.findall(r"\bsecurity_reason[=: ]+[\"']?([a-z0-9_.-]+)", raw, re.I))),
    }


def test_report(final, observations):
    result = []
    for name, status in sorted(final.items()):
        lines = observations.get(name, [])[-8:]
        result.append({
            "name": name,
            "status": status,
            "steps": ["started", f"completed: {status}"],
            "observations": lines,
            "http_statuses": sorted(set(re.findall(r"(?:status=|HTTP/[0-9.]+\\s+)(\\d{3})", "\\n".join(lines), re.I))),
        })
    return result


def runtime_config_checksum():
    runtime = os.getenv("WAF_E2E_RUNTIME_CONTAINER", "waf-e2e-runtime")
    try:
        out = subprocess.check_output(["docker", "exec", runtime, "sh", "-lc", "find /etc/waf/nginx -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum"], text=True, stderr=subprocess.DEVNULL)
        return out.split()[0]
    except Exception:
        return "unavailable"


def runtime_adaptive_evidence():
    runtime = os.getenv("WAF_E2E_RUNTIME_CONTAINER", "waf-e2e-runtime")
    commands = {
        "adaptive_entries": ["docker", "exec", runtime, "sh", "-lc", "cat /etc/waf/l4guard-adaptive/adaptive.json 2>/dev/null || true"],
        "iptables_rules": ["docker", "exec", runtime, "iptables", "-S", "WAF-RUNTIME-L4"],
    }
    result = {}
    for name, command in commands.items():
        try:
            value = subprocess.check_output(command, text=True, stderr=subprocess.DEVNULL).strip()
            result[name] = value[:12000] if value else "unavailable"
        except Exception:
            result[name] = "unavailable"
    return result


def request_security_evidence():
    base_url = os.getenv("WAF_E2E_BASE_URL", "").rstrip("/")
    username = os.getenv("WAF_E2E_USERNAME", "")
    password = os.getenv("WAF_E2E_PASSWORD", "")
    if not base_url or not username or not password:
        return []
    try:
        jar = CookieJar()
        opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
        login = urllib.request.Request(
            base_url + "/api/auth/login",
            data=json.dumps({"username": username, "password": password}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with opener.open(login, timeout=5):
            pass
        with opener.open(base_url + "/api/requests?limit=100", timeout=5) as response:
            payload = json.load(response)
    except Exception:
        return []
    rows = payload.get("requests", []) if isinstance(payload, dict) else payload if isinstance(payload, list) else []
    result = []
    for row in rows:
        if not isinstance(row, dict) or not row.get("security_reason"):
            continue
        result.append({key: row.get(key) for key in ("id", "site_id", "uri", "method", "status", "security_reason", "timestamp")})
    return result[:100]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--log", required=True)
    parser.add_argument("--runtime-log")
    parser.add_argument("--output-dir", required=True)
    parser.add_argument("--suite", required=True)
    args = parser.parse_args()
    log_path = Path(args.log)
    runtime_log_path = Path(args.runtime_log) if args.runtime_log else log_path
    result, observations = test_evidence(log_path)
    counts = {key: sum(value == key for value in result.values()) for key in ("pass", "fail", "skip")}
    request_evidence = request_security_evidence()
    report = {
        "schema_version": 1,
        "suite": args.suite,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "commit": os.getenv("CI_COMMIT_SHA") or run_git("rev-parse", "HEAD"),
        "pipeline_url": os.getenv("CI_PIPELINE_URL", "local"),
        "status": "passed" if counts["fail"] == 0 else "failed",
        "summary": counts,
        "tests": test_report(result, observations),
        "runtime_evidence": {
            **extract_runtime_evidence(runtime_log_path),
            "runtime_config_checksum": runtime_config_checksum(),
            **runtime_adaptive_evidence(),
        },
        "request_security_evidence": request_evidence,
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
    tests = "\n".join(
        f"- `{item['name']}` — **{item['status']}**; steps: {', '.join(item['steps'])}; "
        f"HTTP: {', '.join(item['http_statuses']) or 'n/a'}"
        for item in report["tests"]
    ) or "- No matching E2E test events / Нет подходящих событий E2E."
    report["runtime_evidence"]["blocking_reasons"] = sorted(set(
        report["runtime_evidence"]["blocking_reasons"] + [str(item["security_reason"]) for item in request_evidence]
    ))
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

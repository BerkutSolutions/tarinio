#!/usr/bin/env python3
"""Bundle verified CI evidence into a GitHub release."""
import argparse
import json
import shutil
import sys
from datetime import datetime, timezone
from pathlib import Path


SUITE_DESCRIPTIONS = {
    "smoke": "Быстрая обязательная проверка входа, health-check, AntiBot, Basic Auth, ModSecurity и телеметрии.",
    "security-invariants": "Инварианты безопасности: изоляция сайтов, RBAC, TOTP step-up, trusted proxy, recovery и защита панели.",
    "management-rate-limit": "Глобальный L7 rate limit не блокирует контур управления.",
    "full": "Полный контур: шаблоны страниц, compiler-to-runtime, error pages, Geo, L4/L7, TLS, recovery и несколько сайтов.",
    "baseline": "Базовая DAST-проверка поверхности одноразового WAF-стенда.",
    "negative": "Негативные security-пробы: блокировки атак, защита API, заголовки, cookies и безопасный fuzzing.",
}


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
        passed, failed, skipped = (int(summary.get(key, 0)) for key in ("pass", "fail", "skip"))
        if report.get("status") != "passed" or not passed or failed or skipped:
            raise RuntimeError(f"E2E evidence {name} is not a complete success: {summary}")
        result.append({"job": name, "description": SUITE_DESCRIPTIONS.get(name, "Функциональная E2E-проверка WAF-стенда."), "passed": passed, "failed": failed, "skipped": skipped})
    return result


def validate_dast(reports):
    if not reports:
        raise RuntimeError("no DAST evidence reports were provided")
    result = []
    for name, report in reports:
        counts = report.get("counts", {})
        high, critical = int(counts.get("High", 0)), int(counts.get("Critical", 0))
        if report.get("status") != "passed" or high or critical or report.get("blocking_alerts"):
            raise RuntimeError(f"DAST evidence {name} contains blocking findings: {counts}")
        result.append({"job": name, "description": SUITE_DESCRIPTIONS.get(name, "DAST-проверка WAF-стенда."), "high": high, "critical": critical})
    return result


def copy_source_evidence(output, source_root, filename, prefix):
    destination = output / "source-evidence" / prefix
    destination.mkdir(parents=True, exist_ok=True)
    for source in sorted(Path(source_root).glob(f"*/{filename}")):
        shutil.copy2(source, destination / f"{source.parent.name}-{filename}")
        markdown = source.with_suffix(".md")
        if markdown.exists():
            shutil.copy2(markdown, destination / f"{source.parent.name}-{markdown.name}")


def load_stability(root):
    reports = []
    for path in sorted(Path(root).glob("*/e2e-stability.json")):
        try:
            report = json.loads(path.read_text(encoding="utf-8-sig"))
        except (OSError, json.JSONDecodeError) as error:
            raise RuntimeError(f"cannot read stability report {path}: {error}") from error
        reports.append({
            "job": path.parent.name,
            "test_attempts": int(report.get("test_attempts", 0)),
            "build_retries": int(report.get("build_retries", 0)),
            "flaky": bool(report.get("flaky", False)),
        })
    return reports


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--e2e-root", required=True)
    parser.add_argument("--dast-root", required=True)
    parser.add_argument("--output-dir", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--commit", required=True)
    args = parser.parse_args()

    e2e = validate_e2e(load_reports(args.e2e_root, "e2e-evidence.json") + load_reports(args.dast_root, "e2e-evidence.json"))
    stability = load_stability(args.e2e_root) + load_stability(args.dast_root)
    dast = validate_dast(load_reports(args.dast_root, "dast-evidence.json"))
    output = Path(args.output_dir)
    output.mkdir(parents=True, exist_ok=True)
    copy_source_evidence(output, args.e2e_root, "e2e-evidence.json", "e2e")
    copy_source_evidence(output, args.e2e_root, "e2e-stability.json", "e2e-stability")
    copy_source_evidence(output, args.dast_root, "e2e-evidence.json", "dast-e2e")
    copy_source_evidence(output, args.dast_root, "e2e-stability.json", "dast-e2e-stability")
    copy_source_evidence(output, args.dast_root, "dast-evidence.json", "dast")
    shutil.make_archive(str(output / "release-evidence-source"), "zip", output / "source-evidence")

    report = {"schema_version": 2, "version": args.version, "commit": args.commit, "generated_at": datetime.now(timezone.utc).isoformat(), "status": "passed", "e2e": e2e, "e2e_stability": stability, "dast": dast}
    (output / "release-evidence.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    e2e_rows = "\n".join(f"| `{x['job']}` | {x['description']} | {x['passed']} | {x['failed']} | {x['skipped']} |" for x in e2e)
    dast_rows = "\n".join(f"| `{x['job']}` | {x['description']} | {x['high']} | {x['critical']} | пройдено |" for x in dast)
    stability_rows = "\n".join(f"| `{item['job']}` | {item['test_attempts']} | {item['build_retries']} | {'да' if item['flaky'] else 'нет'} |" for item in stability)
    stability_section = "" if not stability else f"""\n| Стабильность E2E | Попыток теста | Повторов сборки | Инфраструктурная нестабильность |\n| --- | ---: | ---: | --- |\n{stability_rows}\n"""
    summary = f"""### Результаты проверки WAF\n\nE2E-наборы разворачивают одноразовый стенд с control-plane, runtime, базой данных, Vault и тестовыми upstream-сервисами. Они подтверждают цепочку: настройка через API -> compile/apply revision -> активная runtime-конфигурация -> фактический HTTP-результат.\n\n| E2E-набор | Что подтверждает | Пройдено | Ошибок | Пропусков |\n| --- | --- | ---: | ---: | ---: |\n{e2e_rows}\n\n| DAST-набор | Что подтверждает | Высокий | Критический | Статус |\n| --- | --- | ---: | ---: | --- |\n{dast_rows}\n{stability_section}\nВсе E2E-наборы пройдены без ошибок и пропусков. DAST не выявил блокирующих уязвимостей уровней «Высокий» и «Критический». Полные машиночитаемые доказательства приложены к релизу.\n"""
    (output / "release-evidence-summary.md").write_text(summary, encoding="utf-8")
    details = f"""# Доказательства проверок релиза {args.version}\n\n**Коммит:** `{args.commit}`  \n**Статус:** **пройдено**\n\n{summary}\n### Исходные доказательства\n\n`release-evidence-source.zip` содержит JSON и Markdown-отчёты E2E/DAST без тел запросов и секретов.\n"""
    (output / "release-evidence.md").write_text(details, encoding="utf-8")


if __name__ == "__main__":
    try:
        main()
    except RuntimeError as error:
        print(f"release evidence validation failed: {error}", file=sys.stderr)
        sys.exit(1)

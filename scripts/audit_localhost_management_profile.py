#!/usr/bin/env python3
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path('/var/lib/waf')
ACTIVE = ROOT / 'active' / 'current.json'
NGINX_EASY = Path('/etc/waf/nginx/easy/localhost.conf')
NGINX_SITE = Path('/etc/waf/nginx/sites/localhost.conf')


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding='utf-8', errors='replace')
    except Exception:
        return ''


def load_active_revision() -> str:
    try:
        data = json.loads(read_text(ACTIVE))
        return str(data.get('revision_id', '')).strip()
    except Exception:
        return ''


def candidate_path(rev: str, rel: str) -> Path:
    return ROOT / 'candidates' / rev / rel


def has_blocking_localhost_profile(text: str) -> dict:
    return {
        'modsecurity_on': 'modsecurity on;' in text,
        'modsecurity_rules_file': '/etc/waf/modsecurity/easy/localhost.conf' in text,
        'captcha_effective': 'set $waf_antibot_effective_challenge "captcha";' in text,
        'stage1_redirect': '/challenge/stage1/verify' in text,
        'guard403': 'if ($waf_antibot_guard ~ "^0:0:") { return 403; }' in text,
    }


def main() -> int:
    rev = load_active_revision()
    report = {
        'active_revision': rev,
        'signals': {},
        'findings': [],
        'recommendations': [],
    }

    live_easy = read_text(NGINX_EASY)
    live_site = read_text(NGINX_SITE)
    candidate_easy = read_text(candidate_path(rev, 'nginx/easy/localhost.conf')) if rev else ''
    candidate_site = read_text(candidate_path(rev, 'nginx/sites/localhost.conf')) if rev else ''

    live = has_blocking_localhost_profile(live_easy)
    candidate = has_blocking_localhost_profile(candidate_easy)
    report['signals']['live_easy_localhost'] = live
    report['signals']['candidate_easy_localhost'] = candidate
    report['signals']['site_has_login_locations'] = 'location = /login {' in live_site or 'location = /login {' in candidate_site
    report['signals']['site_has_api_proxy'] = 'location ^~ /api/' in live_site or 'location ^~ /api/' in candidate_site

    blocking = live['modsecurity_on'] and live['modsecurity_rules_file'] and (live['captcha_effective'] or live['stage1_redirect'] or live['guard403'])
    if blocking:
        report['findings'].append({
            'code': 'localhost_management_profile_blocking',
            'severity': 'high',
            'message': 'Active localhost easy profile still contains blocking ModSecurity/antibot directives that can break /dashboard and POST /api/app/ping.',
        })
        report['recommendations'].append('Verify the localhost easy-site profile source of truth and rewrite it to transparent + antibot=no + modsecurity disabled for management localhost.')
        report['recommendations'].append('After rewriting profile, run compile/apply and re-check GET https://localhost/dashboard and POST https://localhost/api/app/ping.')

    drift = live != candidate and rev
    if drift:
        report['findings'].append({
            'code': 'localhost_profile_live_candidate_drift',
            'severity': 'medium',
            'message': 'Active /etc/waf nginx localhost easy profile differs from candidate revision localhost easy profile.',
        })
        report['recommendations'].append('Inspect apply/reload path and runtime sync because candidate and live localhost profile content diverged.')

    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 1 if report['findings'] else 0


if __name__ == '__main__':
    raise SystemExit(main())

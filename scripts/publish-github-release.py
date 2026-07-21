#!/usr/bin/env python3
"""Create or update a GitHub release and replace its attached assets."""

import argparse
import json
import os
import sys
from pathlib import Path
from urllib.error import HTTPError
from urllib.parse import urlencode
from urllib.request import Request, urlopen


API_ROOT = "https://api.github.com"


def request(token, method, url, payload=None, content_type="application/json"):
    headers = {
        "Accept": "application/vnd.github+json",
        "Authorization": f"Bearer {token}",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    if payload is not None:
        headers["Content-Type"] = content_type
    req = Request(url, data=payload, headers=headers, method=method)
    with urlopen(req, timeout=60) as response:
        content = response.read()
        if not content:
            return response.status, None
        return response.status, json.loads(content.decode("utf-8"))


def release_for_tag(token, repo, tag):
    try:
        _, release = request(token, "GET", f"{API_ROOT}/repos/{repo}/releases/tags/{tag}")
        return release
    except HTTPError as error:
        if error.code != 404:
            raise
        return None


def upsert_release(token, repo, tag, title, notes):
    payload = json.dumps({"tag_name": tag, "name": title, "body": notes}).encode("utf-8")
    release = release_for_tag(token, repo, tag)
    if release is None:
        _, release = request(token, "POST", f"{API_ROOT}/repos/{repo}/releases", payload)
        print(f"[release] Created GitHub Release {tag}")
        return release
    _, release = request(token, "PATCH", f"{API_ROOT}/repos/{repo}/releases/{release['id']}", payload)
    print(f"[release] Updated GitHub Release {tag}")
    return release


def replace_assets(token, release, assets):
    expected = {asset.name for asset in assets}
    for existing in release.get("assets", []):
        if existing.get("name") in expected:
            request(token, "DELETE", existing["url"])
    upload_base = release["upload_url"].split("{", 1)[0]
    for asset in assets:
        query = urlencode({"name": asset.name, "label": asset.name})
        request(token, "POST", f"{upload_base}?{query}", asset.read_bytes(), "application/octet-stream")
        print(f"[release] Uploaded {asset.name}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--title", required=True)
    parser.add_argument("--notes", required=True)
    parser.add_argument("--asset", action="append", default=[])
    args = parser.parse_args()

    token = os.environ.get("GITHUB_TOKEN", "").strip()
    if not token:
        raise RuntimeError("GITHUB_TOKEN is required")
    notes = Path(args.notes).read_text(encoding="utf-8")
    assets = [Path(value) for value in args.asset]
    missing = [str(asset) for asset in assets if not asset.is_file()]
    if missing:
        raise RuntimeError(f"release asset is missing: {', '.join(missing)}")

    release = upsert_release(token, args.repo, args.tag, args.title, notes)
    replace_assets(token, release, assets)


if __name__ == "__main__":
    try:
        main()
    except (HTTPError, OSError, RuntimeError) as error:
        print(f"GitHub release publication failed: {error}", file=sys.stderr)
        raise SystemExit(1)

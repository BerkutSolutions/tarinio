#!/usr/bin/env python3
"""Extract one top-level release section from CHANGELOG.md."""

import argparse
from pathlib import Path


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--changelog", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    header = f"## [{args.version}]"
    lines = Path(args.changelog).read_text(encoding="utf-8-sig").splitlines(keepends=True)
    start = next((index for index, line in enumerate(lines) if line.rstrip("\r\n").startswith(header)), None)
    if start is None:
        raise RuntimeError(f"release section {header!r} was not found in {args.changelog}")

    end = next(
        (index for index in range(start + 1, len(lines)) if lines[index].rstrip("\r\n").startswith("## [")),
        len(lines),
    )
    section = "".join(lines[start:end]).strip() + "\n"
    if section == header + "\n":
        raise RuntimeError(f"release section {header!r} is empty")
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(section, encoding="utf-8")


if __name__ == "__main__":
    try:
        main()
    except RuntimeError as error:
        raise SystemExit(f"release notes extraction failed: {error}")

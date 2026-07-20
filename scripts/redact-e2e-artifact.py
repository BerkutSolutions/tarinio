#!/usr/bin/env python3
"""Redact credentials from a text E2E artifact without removing diagnostics."""
import argparse
import re
from pathlib import Path


REDACTIONS = (
    (re.compile(r'(?im)(authorization\s*[:=]\s*)(basic|bearer|apikey)\s+[^\s"\']+'), r'\1[REDACTED]'),
    (re.compile(r'(?im)(cookie\s*[:=]\s*)[^\r\n]+'), r'\1[REDACTED]'),
    (re.compile(r'(?i)("(?:password|token|secret|api[_-]?key)"\s*:\s*")[^"]*(")'), r'\1[REDACTED]\2'),
    (re.compile(r'(?i)\b(password|token|secret|api[_-]?key)=([^\s&\r\n]+)'), r'\1=[REDACTED]'),
)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("path")
    args = parser.parse_args()
    path = Path(args.path)
    if not path.exists():
        return
    text = path.read_text(encoding="utf-8", errors="replace")
    for pattern, replacement in REDACTIONS:
        text = pattern.sub(replacement, text)
    path.write_text(text, encoding="utf-8")


if __name__ == "__main__":
    main()

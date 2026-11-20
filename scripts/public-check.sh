#!/usr/bin/env bash
set -uo pipefail
cd "$(dirname "$0")/.." || exit 2
found=0
if git grep -nIE '/home/[a-z]|/Users/[A-Za-z]' -- .; then found=1; fi
if [ -n "${HOARD_PRIVATE_PATTERNS:-}" ] &&
	git grep -nIiE "$HOARD_PRIVATE_PATTERNS" -- .; then found=1; fi
if [ "$found" -eq 1 ]; then echo "public-check: private identifiers found" >&2; exit 1; fi
echo "public-check: clean"

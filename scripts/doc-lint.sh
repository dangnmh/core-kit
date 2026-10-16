#!/usr/bin/env bash
set -euo pipefail

echo "=== Running BMAD HADS Documentation Lint Gate ==="

ERRORS=0

# 1. Enforce line count caps (<= 120 lines) on Layer 0 and Layer 1 READMEs
FILES_TO_CHECK=(
  "AGENTS.md"
  "docs/pkg/README.md"
  "docs/py/README.md"
  "docs/ts/README.md"
)

for file in "${FILES_TO_CHECK[@]}"; do
  if [[ -f "$file" ]]; then
    LINES=$(wc -l < "$file")
    if [[ "$LINES" -gt 120 ]]; then
      echo "❌ ERROR: $file exceeds 120 lines (currently $LINES lines). Violation of CONST-08."
      ERRORS=$((ERRORS + 1))
    else
      echo "✅ PASS: $file ($LINES lines <= 120)"
    fi
  else
    echo "⚠️ WARNING: $file does not exist yet."
    ERRORS=$((ERRORS + 1))
  fi
done

# 2. Check for broken markdown links across all docs (excluding code blocks)
echo "Checking markdown links..."
LINK_ERRORS=$(python3 -c '
import glob, re, os, sys

errors = 0
for f in glob.glob("docs/**/*.md", recursive=True):
    d = os.path.dirname(f)
    try:
        with open(f, "r", encoding="utf-8", errors="ignore") as fp:
            lines = fp.readlines()
    except Exception as e:
        continue

    in_code_block = False
    for line_num, line in enumerate(lines, 1):
        stripped = line.strip()
        if stripped.startswith("```"):
            in_code_block = not in_code_block
            continue
        if in_code_block:
            continue

        for m in re.finditer(r"\[[^\]]+\]\(([^)]+\.md(?:#[^)]*)?)\)", line):
            link = m.group(1)
            if link.startswith("http://") or link.startswith("https://") or link.startswith("mailto:"):
                continue
            clean = link.split("#")[0]
            if not clean:
                continue
            if clean.startswith("file://"):
                target = clean[7:]
            elif clean.startswith("/"):
                target = clean
            else:
                target = os.path.normpath(os.path.join(d, clean))
                if not os.path.exists(target) and os.path.exists(clean):
                    target = clean
            if not os.path.exists(target):
                print(f"❌ BROKEN LINK in {f}:{line_num}: target \"{clean}\" does not exist", file=sys.stderr)
                errors += 1

sys.exit(errors)
' 2>&1 || true)

if [[ -n "$LINK_ERRORS" ]]; then
  echo "$LINK_ERRORS"
  LINK_COUNT=$(echo "$LINK_ERRORS" | wc -l)
  ERRORS=$((ERRORS + LINK_COUNT))
fi

if [[ "$ERRORS" -gt 0 ]]; then
  echo "=== Doc-Lint Failed with $ERRORS error(s) ==="
  exit 1
fi

echo "=== All Documentation Checks Passed Cleanly ==="
exit 0

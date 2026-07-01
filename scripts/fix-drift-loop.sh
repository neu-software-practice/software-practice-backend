#!/bin/bash
# ================================================================
# API Drift Fix Loop
#
# 1. Extract frontend Zod fields
# 2. Extract backend Go fields
# 3. Compare → drift report
# 4. If drift found: report for manual/agent fix
# 5. Verify: go test + golangci-lint
# 6. Git commit
# 7. Re-scan until dry (2 consecutive clean rounds)
# ================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(dirname "$SCRIPT_DIR")"
FRONTEND_DIR="$BACKEND_DIR/../neuhis-agent-front"

ROUND=0
DRY=0
MAX_ROUNDS=5

echo "=== API Drift Fix Loop ==="
echo "Backend:  $BACKEND_DIR"
echo "Frontend: $FRONTEND_DIR"
echo ""

while [ $DRY -lt 2 ] && [ $ROUND -lt $MAX_ROUNDS ]; do
  ROUND=$((ROUND + 1))
  echo "=========================================="
  echo "=== Round $ROUND ==="
  echo "=========================================="

  # ── Step 1: Extract ──
  echo "[1/6] Extracting frontend Zod fields..."
  (cd "$FRONTEND_DIR" && node scripts/extract-zod-fields.mjs 2>&1 | grep "\[DONE\]") || {
    echo "ERROR: Frontend extraction failed"
    exit 1
  }

  echo "[2/6] Extracting backend Go fields..."
  (cd "$BACKEND_DIR" && node scripts/extract-go-fields.mjs 2>&1 | grep "\[DONE\]") || {
    echo "ERROR: Backend extraction failed"
    exit 1
  }

  # ── Step 2: Compare ──
  echo "[3/6] Comparing fields..."
  (cd "$BACKEND_DIR" && node scripts/compare-fields.mjs 2>&1) || true

  DRIFT_COUNT=$(node -e "
    try {
      const r = JSON.parse(require('fs').readFileSync('$BACKEND_DIR/drift-report-fields.json', 'utf-8'));
      console.log(r.totalDriftItems || 0);
    } catch(e) { console.log(0); }
  ")

  if [ "$DRIFT_COUNT" -eq 0 ]; then
    DRY=$((DRY + 1))
    echo ""
    echo "✅ No drift detected. Dry round $DRY/2"
    echo ""
    continue
  fi

  DRY=0
  echo ""
  echo "⚠️  $DRIFT_COUNT drift items found."
  echo ""

  # ── Step 3: Report ──
  echo "[4/6] Drift report written to drift-report-fields.json"
  echo "       Review and fix drift items manually or via agent."
  echo ""
  echo "       Breaking out of loop for manual fix."
  echo "       After fixing, re-run: bash scripts/fix-drift-loop.sh"
  break

  # NOTE: The automated fix (Step 3b) is done by an agent outside this script.
  # The agent reads drift-report-fields.json, modifies Go source files,
  # then continues with steps 4-6 below.

  # ── Step 4: Verify ──
  echo "[5/6] Running tests..."
  (cd "$BACKEND_DIR" && go test -count=1 -short ./... -race) || {
    echo "ERROR: Tests failed. Fix issues before continuing."
    exit 1
  }

  echo "[5/6] Running linter..."
  (cd "$BACKEND_DIR" && golangci-lint run ./...) || {
    echo "ERROR: Lint failed. Fix issues before continuing."
    exit 1
  }

  # ── Step 5: Commit ──
  echo "[6/6] Committing fixes..."
  (cd "$BACKEND_DIR" && git add -A && git commit -m "fix: 修复 API 字段级漂移 (round $ROUND)

  Co-Authored-By: Claude <noreply@anthropic.com>") || {
    echo "WARN: Nothing to commit or commit failed."
  }
done

echo ""
echo "=========================================="
echo "=== Loop Complete ==="
echo "=== Rounds: $ROUND, Dry: $DRY ==="
echo "=========================================="

if [ $DRY -ge 2 ]; then
  echo "✅ SUCCESS: API is fully aligned at field level!"
  exit 0
elif [ $ROUND -ge $MAX_ROUNDS ]; then
  echo "⚠️  Max rounds reached. Some drift may remain."
  exit 1
else
  echo "⏸️  Paused for manual fix. See drift-report-fields.json"
  exit 2
fi

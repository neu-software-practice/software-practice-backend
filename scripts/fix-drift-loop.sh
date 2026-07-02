#!/bin/bash
# ================================================================
# API Drift Fix Loop
#
# 1. Extract frontend Zod fields
# 2. Extract backend Go fields
# 3. Compare: field-level drift + request drift
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
  echo "[1/7] Extracting frontend Zod fields..."
  (cd "$FRONTEND_DIR" && node scripts/extract-zod-fields.mjs 2>&1 | grep "\[DONE\]") || {
    echo "ERROR: Frontend extraction failed"
    exit 1
  }

  echo "[2/7] Extracting backend Go fields..."
  (cd "$BACKEND_DIR" && node scripts/extract-go-fields.mjs 2>&1 | grep "\[DONE\]") || {
    echo "ERROR: Backend extraction failed"
    exit 1
  }

  # ── Step 2: Compare ──
  echo "[3/7] Comparing response fields (field-level drift)..."
  (cd "$BACKEND_DIR" && node scripts/compare-fields.mjs 2>&1) || true

  echo "[4/7] Comparing request bodies & query params (request drift)..."
  (cd "$BACKEND_DIR" && node scripts/compare-request.mjs 2>&1) || true

  # ── Step 3: Aggregate drift counts ──
  FIELD_DRIFT=$(node -e "
    try {
      const r = JSON.parse(require('fs').readFileSync('$BACKEND_DIR/drift-report-fields.json', 'utf-8'));
      console.log(r.totalDriftItems || 0);
    } catch(e) { console.log(0); }
  ")
  REQUEST_DRIFT=$(node -e "
    try {
      const r = JSON.parse(require('fs').readFileSync('$BACKEND_DIR/drift-report-request.json', 'utf-8'));
      console.log(r.totalDriftItems || 0);
    } catch(e) { console.log(0); }
  ")
  TOTAL_DRIFT=$((FIELD_DRIFT + REQUEST_DRIFT))

  if [ "$TOTAL_DRIFT" -eq 0 ]; then
    DRY=$((DRY + 1))
    echo ""
    echo "✅ No drift detected (fields: $FIELD_DRIFT, requests: $REQUEST_DRIFT). Dry round $DRY/2"
    echo ""
    continue
  fi

  DRY=0
  echo ""
  echo "⚠️  Drift found: $FIELD_DRIFT field-level + $REQUEST_DRIFT request-level = $TOTAL_DRIFT total"
  echo ""

  # ── Step 4: Report ──
  echo "[5/7] Drift reports:"
  echo "       - drift-report-fields.json  ($FIELD_DRIFT items)"
  echo "       - drift-report-request.json ($REQUEST_DRIFT items)"
  echo "       Review and fix drift items manually or via agent."
  echo ""
  echo "       Breaking out of loop for manual fix."
  echo "       After fixing, re-run: bash scripts/fix-drift-loop.sh"
  break

  # NOTE: The automated fix (Step 5) is done by an agent outside this script.
  # The agent reads drift reports, modifies Go/TS source files,
  # then continues with steps 6-7 below.

  # ── Step 6: Verify ──
  echo "[6/7] Running tests..."
  (cd "$BACKEND_DIR" && go test -count=1 -short ./... -race) || {
    echo "ERROR: Tests failed. Fix issues before continuing."
    exit 1
  }

  echo "[6/7] Running linter..."
  (cd "$BACKEND_DIR" && golangci-lint run ./...) || {
    echo "ERROR: Lint failed. Fix issues before continuing."
    exit 1
  }

  # ── Step 7: Commit ──
  echo "[7/7] Committing fixes..."
  (cd "$BACKEND_DIR" && git add -A && git commit -m "fix: 修复 API 漂移 (round $ROUND)

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
  echo "✅ SUCCESS: API is fully aligned!"
  exit 0
elif [ $ROUND -ge $MAX_ROUNDS ]; then
  echo "⚠️  Max rounds reached. Some drift may remain."
  exit 1
else
  echo "⏸️  Paused for manual fix. See drift-report-fields.json and drift-report-request.json"
  exit 2
fi

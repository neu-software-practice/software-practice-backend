#!/bin/bash
set -euo pipefail

COVER_PROFILE="${COVER_PROFILE:-/tmp/precommit-coverage.out}"
SERVICE_COVERAGE_THRESHOLD="${SERVICE_COVERAGE_THRESHOLD:-90}"
TOTAL_COVERAGE_THRESHOLD="${TOTAL_COVERAGE_THRESHOLD:-75}"
MODULE_PATH="$(go list -m)"

mapfile -t TEST_PACKAGES < <(go list ./... | grep -v /cmd/ | grep -v /tests/testutil | grep -v /internal/service/medagent)

go test -race -timeout=5m -coverprofile="$COVER_PROFILE" "${TEST_PACKAGES[@]}"

FAIL=0

coverage_for_prefix() {
  local prefix="$1"

  awk -v prefix="$prefix" '
    BEGIN { covered = 0; total = 0 }
    /^mode:/ { next }
    index($1, prefix) == 1 {
      statements = $2 + 0
      count = $3 + 0
      total += statements
      if (count > 0) {
        covered += statements
      }
    }
    END {
      if (total == 0) {
        printf "0.0"
      } else {
        printf "%.1f", covered * 100 / total
      }
    }
  ' "$COVER_PROFILE"
}

echo "=== Service Layer Coverage (target: >=${SERVICE_COVERAGE_THRESHOLD}%) ==="
for pkg in internal/service/patient internal/service/visit internal/service/workbench; do
  COV_NUM="$(coverage_for_prefix "$MODULE_PATH/$pkg")"
  echo "  ./$pkg/...: ${COV_NUM}%"
  if awk -v cov="$COV_NUM" -v threshold="$SERVICE_COVERAGE_THRESHOLD" 'BEGIN{exit(!(cov < threshold))}'; then
    echo "  -> FAIL: below ${SERVICE_COVERAGE_THRESHOLD}%"
    FAIL=1
  fi
done

echo "=== Total Coverage (target: >=${TOTAL_COVERAGE_THRESHOLD}%) ==="
TOTAL=$(go tool cover -func="$COVER_PROFILE" | grep total | awk '{print $3}' | sed 's/%//')
echo "Total: ${TOTAL}%"
if awk -v total="$TOTAL" -v threshold="$TOTAL_COVERAGE_THRESHOLD" 'BEGIN{exit(!(total < threshold))}'; then
  echo "Total coverage ${TOTAL}% below ${TOTAL_COVERAGE_THRESHOLD}% threshold"
  FAIL=1
fi

if [ $FAIL -eq 1 ]; then
  echo "Coverage check FAILED"
  exit 1
fi
echo "Coverage check PASSED"

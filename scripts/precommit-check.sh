#!/bin/bash
set -euo pipefail

go test -race -coverprofile=/tmp/precommit-coverage.out $(go list ./... | grep -v /cmd/server | grep -v /tests/testutil | grep -v /internal/service/medagent)

FAIL=0

echo "=== Service Layer Coverage (target: >=90%) ==="
for pkg in ./internal/service/patient/... ./internal/service/visit/... ./internal/service/workbench/...; do
  COV=$(go tool cover -func=/tmp/precommit-coverage.out 2>/dev/null | grep "$pkg" | tail -1 | awk '{print $NF}')
  if [ -z "$COV" ]; then COV="0.0%"; fi
  COV_NUM=$(echo "$COV" | sed 's/%//')
  echo "  $pkg: $COV"
  if [ "$(echo "$COV_NUM < 90" | bc -l)" -eq 1 ]; then
    echo "  -> FAIL: below 90%"
    FAIL=1
  fi
done

echo "=== Total Coverage (target: >=75%) ==="
TOTAL=$(go tool cover -func=/tmp/precommit-coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Total: ${TOTAL}%"
if [ "$(echo "$TOTAL < 75" | bc -l)" -eq 1 ]; then
  echo "Total coverage ${TOTAL}% below 75% threshold"
  FAIL=1
fi

if [ $FAIL -eq 1 ]; then
  echo "Coverage check FAILED"
  exit 1
fi
echo "Coverage check PASSED"

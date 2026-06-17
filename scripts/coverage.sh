#!/usr/bin/env bash
# Fail if total coverage in the given profile is below the threshold.
# Usage: scripts/coverage.sh cover.out 80
#
# Excluded from the gate: process entrypoints (cmd/*) and the live-MySQL adapters
# (internal/migrate, internal/pkg/database). These need a real database and are
# validated by the CI `make migrate` / `make seed` steps instead of unit tests.
set -euo pipefail

profile="${1:-cover.out}"
threshold="${2:-80}"

filtered="$(mktemp)"
trap 'rm -f "${filtered}"' EXIT
grep -vE '/(cmd/|internal/migrate|internal/pkg/database|internal/swagger)' "${profile}" > "${filtered}"

total="$(go tool cover -func="${filtered}" | awk '/^total:/ {gsub(/%/,"",$3); print $3}')"
echo "total coverage (gated packages): ${total}% (threshold: ${threshold}%)"

awk -v have="${total}" -v want="${threshold}" 'BEGIN { exit (have + 0 < want + 0) ? 1 : 0 }' || {
    echo "coverage ${total}% is below threshold ${threshold}%" >&2
    exit 1
}

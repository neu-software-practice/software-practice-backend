#!/usr/bin/env bash
# =============================================================================
# HIS Backend Smoke Test Runner
# =============================================================================
# Builds the full Docker stack, waits for backend readiness, runs the Newman
# Postman collection, and reports results.
#
# Usage:
#   ./test/smoke/run_smoke.sh              # Full run with teardown
#   ./test/smoke/run_smoke.sh --no-teardown # Leave stack running for debugging
#
# Dependencies: docker, curl, newman (or npx)
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
COLLECTION="$SCRIPT_DIR/his_smoke.postman_collection.json"
RESULTS_DIR="$SCRIPT_DIR/results"
TEARDOWN=true

# --- Parse arguments ---
for arg in "$@"; do
    case "$arg" in
        --no-teardown) TEARDOWN=false ;;
        --help|-h)
            echo "Usage: $0 [--no-teardown]"
            echo "  --no-teardown  Leave the Docker stack running after tests"
            exit 0
            ;;
    esac
done

# --- Colored output ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[smoke]${NC} $*"; }
warn() { echo -e "${YELLOW}[smoke]${NC} $*"; }
err()  { echo -e "${RED}[smoke]${NC} $*" >&2; }

# --- Step 0: Ensure prerequisites ---
log "Checking prerequisites..."

if ! command -v docker >/dev/null 2>&1; then
    err "docker is required but not found"
    exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
    err "curl is required but not found"
    exit 1
fi

# Detect Newman: prefer npx (comes with Node.js), fall back to global
NEWMAN_CMD=""
if command -v npx >/dev/null 2>&1; then
    NEWMAN_CMD="npx newman"
elif command -v newman >/dev/null 2>&1; then
    NEWMAN_CMD="newman"
else
    err "Newman is not installed. Install it with: npm install -g newman"
    exit 1
fi
log "Using: $NEWMAN_CMD"

# --- Step 1: Build and start the stack ---
log "Building and starting Docker stack..."
cd "$PROJECT_ROOT"
docker compose up -d --build 2>&1 | sed 's/^/  /'
log "Docker stack is starting up."

# --- Step 2: Wait for backend health ---
log "Waiting for backend health check (timeout: 300s)..."
MAX_RETRIES=60
RETRY=0
BACKEND_URL="${SMOKE_BASE_URL:-http://localhost:8080}"

while [ $RETRY -lt $MAX_RETRIES ]; do
    if curl -sf "${BACKEND_URL}/api/health" >/dev/null 2>&1; then
        log "Backend is ready! (attempt $((RETRY+1)))"
        break
    fi
    RETRY=$((RETRY + 1))
    if [ $((RETRY % 6)) -eq 0 ]; then
        warn "Still waiting... (${RETRY}s / ${MAX_RETRIES}s)"
    fi
    sleep 5
done

if [ $RETRY -ge $MAX_RETRIES ]; then
    err "Backend failed to become healthy within timeout"
    err "--- Backend logs (last 50 lines) ---"
    docker compose logs backend --tail=50 2>&1 | sed 's/^/  /'
    if [ "$TEARDOWN" = true ]; then
        log "Tearing down failed stack..."
        docker compose down -v 2>&1 | sed 's/^/  /'
    fi
    exit 1
fi

# Extra grace period for seed data to fully propagate
sleep 2

# --- Step 3: Verify the health endpoint returns valid JSON ---
log "Verifying health endpoint response..."
HEALTH_RESP=$(curl -s "${BACKEND_URL}/api/health")
if echo "$HEALTH_RESP" | grep -q '"success"'; then
    log "Health check response OK: $HEALTH_RESP"
else
    warn "Health check response unexpected: $HEALTH_RESP"
fi

# --- Step 4: Run Newman ---
log "Running Newman smoke tests..."
mkdir -p "$RESULTS_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

set +e  # Capture Newman exit code without aborting
$NEWMAN_CMD run "$COLLECTION" \
    --env-var "base_url=${BACKEND_URL}" \
    --iteration-count 1 \
    --delay-request 100 \
    --timeout-request 10000 \
    --timeout-script 5000 \
    --reporters cli,junit \
    --reporter-junit-export "$RESULTS_DIR/smoke-results-${TIMESTAMP}.xml" \
    --color on 2>&1 | tee "$RESULTS_DIR/smoke-output-${TIMESTAMP}.txt"

NEWMAN_EXIT=${PIPESTATUS[0]}
set -e

# --- Step 5: Report results ---
echo ""
if [ $NEWMAN_EXIT -eq 0 ]; then
    log "✅ ALL SMOKE TESTS PASSED"
else
    err "❌ SOME SMOKE TESTS FAILED (Newman exit code: $NEWMAN_EXIT)"
    warn "Results saved to:"
    warn "  Log:  $RESULTS_DIR/smoke-output-${TIMESTAMP}.txt"
    warn "  JUnit: $RESULTS_DIR/smoke-results-${TIMESTAMP}.xml"
fi

# --- Step 6: Teardown ---
if [ "$TEARDOWN" = true ]; then
    echo ""
    log "Tearing down Docker stack..."
    cd "$PROJECT_ROOT"
    docker compose down -v 2>&1 | sed 's/^/  /'
    log "Teardown complete."
else
    echo ""
    warn "Teardown skipped (--no-teardown). Stack is still running."
    warn "Stop with: docker compose down -v"
fi

exit $NEWMAN_EXIT

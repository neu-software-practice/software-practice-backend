#!/bin/bash
# ============================================================================
# NEUHIS Agent Admin Newman Black-Box Test Runner
# ============================================================================
# 用法:
#   ./tests/newman/run-admin-smoke.sh [baseUrl]
#
# 示例:
#   ./tests/newman/run-admin-smoke.sh http://localhost:8080
#   ./tests/newman/run-admin-smoke.sh                  # defaults to http://localhost:8080
#
# 环境变量:
#   BASE_URL  - 覆盖 baseUrl 参数
#   BAIL       - 设置为 1 启用 --bail (遇错即停)
# ============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

BASE_URL="${1:-${BASE_URL:-http://localhost:8080}}"
COLLECTION="$SCRIPT_DIR/admin.postman_collection.json"
ENVIRONMENT="$SCRIPT_DIR/admin.postman_environment.json"
REPORT_DIR="$SCRIPT_DIR/reports"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "============================================"
echo " NEUHIS Agent Admin Black-Box Tests"
echo "============================================"
echo " Base URL:   $BASE_URL"
echo " Collection: $COLLECTION"
echo " Environment: $ENVIRONMENT"
echo " Reports:    $REPORT_DIR"
echo "============================================"
echo ""

# Create report directory
mkdir -p "$REPORT_DIR"

# Wait for server to be ready
echo "⏳ Waiting for server at $BASE_URL ..."
MAX_RETRIES=30
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -sf "$BASE_URL/api/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Server is ready${NC}"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ "$RETRY_COUNT" -eq "$MAX_RETRIES" ]; then
        echo -e "${RED}❌ Server not ready after $MAX_RETRIES attempts${NC}"
        exit 1
    fi
    printf "."
    sleep 1
done

# Show server health response
echo ""
echo "📋 Health check response:"
curl -s "$BASE_URL/api/health" | python3 -m json.tool 2>/dev/null || echo "(raw response)"
echo ""

# Build newman arguments
NEWMAN_ARGS=(
    "run" "$COLLECTION"
    "-e" "$ENVIRONMENT"
    "--env-var" "baseUrl=$BASE_URL"
    "--reporters" "cli,junit"
    "--reporter-junit-export" "$REPORT_DIR/admin-junit-report.xml"
    "--timeout-request" "60000"
    "--timeout-script" "15000"
    "--delay-request" "200"
)

# Add htmlextra reporter if available
if newman --help 2>/dev/null | grep -q "htmlextra"; then
    NEWMAN_ARGS+=("--reporters" "cli,junit,htmlextra")
    NEWMAN_ARGS+=("--reporter-htmlextra-export" "$REPORT_DIR/admin-html-report.html")
fi

# Bail on first failure if BAIL is set
if [ "${BAIL:-0}" = "1" ]; then
    NEWMAN_ARGS+=("--bail")
fi

# Run tests
echo ""
echo "🧪 Running admin black-box tests..."
echo ""

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
SUMMARY_FILE="$REPORT_DIR/admin-summary_$TIMESTAMP.txt"

set +e
newman "${NEWMAN_ARGS[@]}" 2>&1 | tee "$SUMMARY_FILE"
NEWMAN_EXIT_CODE=${PIPESTATUS[0]}
set -e

echo ""
echo "============================================"
if [ "$NEWMAN_EXIT_CODE" -eq 0 ]; then
    echo -e "${GREEN}✅ All admin black-box tests passed!${NC}"
else
    echo -e "${RED}❌ Some admin tests failed (exit code: $NEWMAN_EXIT_CODE)${NC}"
fi
echo "============================================"
echo " Reports:"
echo "   JUnit: $REPORT_DIR/admin-junit-report.xml"
if [ -f "$REPORT_DIR/admin-html-report.html" ]; then
    echo "   HTML:  $REPORT_DIR/admin-html-report.html"
fi
echo "   Log:   $SUMMARY_FILE"
echo "============================================"

exit $NEWMAN_EXIT_CODE

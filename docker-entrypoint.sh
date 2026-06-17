#!/bin/sh
# Apply migrations (and optionally seed) before starting the API server, so a
# fresh `docker compose up` yields a ready-to-use system (SPEC §9.6).
set -e

# The MySQL healthcheck (mysqladmin ping) may pass before the server is truly
# ready to accept TCP connections, causing migrate to fail with "connection
# refused". Retry with backoff to bridge the race window.
echo "[entrypoint] waiting for database..."
MAX_RETRIES=30
RETRY=0
while [ $RETRY -lt $MAX_RETRIES ]; do
    if /app/migrate -dir up 2>/dev/null; then
        echo "[entrypoint] migrations applied (attempt $((RETRY+1)))"
        break
    fi
    RETRY=$((RETRY + 1))
    if [ $RETRY -ge $MAX_RETRIES ]; then
        echo "[entrypoint] ERROR: database not reachable after ${MAX_RETRIES} attempts"
        exit 1
    fi
    sleep 2
done

if [ "${SEED_ON_START:-true}" = "true" ]; then
    echo "[entrypoint] seeding base data..."
    /app/seed
fi

echo "[entrypoint] starting server..."
exec /app/server

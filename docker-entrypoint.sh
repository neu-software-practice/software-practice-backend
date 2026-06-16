#!/bin/sh
# Apply migrations (and optionally seed) before starting the API server, so a
# fresh `docker compose up` yields a ready-to-use system (SPEC §9.6).
set -e

echo "[entrypoint] applying migrations..."
/app/migrate -dir up

if [ "${SEED_ON_START:-true}" = "true" ]; then
    echo "[entrypoint] seeding base data..."
    /app/seed
fi

echo "[entrypoint] starting server..."
exec /app/server

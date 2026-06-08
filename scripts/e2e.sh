#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "Run 'docker compose up -d' before executing this script."

echo "Waiting for API to become healthy at http://localhost:8000/health..."
until curl -sSf http://localhost:8000/health >/dev/null 2>&1; do
  printf '.'
  sleep 1
done
echo "\nAPI is healthy."

echo "Registering test user..."
REG=$(curl -s -X POST http://localhost:8000/register -H 'Content-Type: application/json' -d '{"email":"test@example.com","password":"password"}')
ACCESS_TOKEN=$(echo "$REG" | python3 -c 'import sys,json;print(json.load(sys.stdin)["Data"]["access_token"])')
echo "Access token acquired."

echo "Creating job that posts to webhook..."
CREATE=$(curl -s -X POST http://localhost:8000/jobs -H "Content-Type: application/json" -H "Authorization: Bearer $ACCESS_TOKEN" -d '{"type":"email","name":"test-job","webhook_url":"http://webhook:9000/echo"}')
JOB_ID=$(echo "$CREATE" | python3 -c 'import sys,json;print(json.load(sys.stdin)["Data"]["job_id"])')
echo "Job created with id: $JOB_ID"

echo "Waiting for job to complete (timeout 60s)..."
for i in $(seq 1 60); do
  STATE=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" http://localhost:8000/jobs/$JOB_ID | python3 -c 'import sys,json;print(json.load(sys.stdin)["Data"]["state"])')
  echo "[$i] job state: $STATE"
  if [[ "$STATE" == "completed" || "$STATE" == "failed" ]]; then
    break
  fi
  sleep 1
done

echo "Showing recent webhook container logs (last 200 lines):"
docker compose logs --no-log-prefix --tail=200 webhook || true

echo "E2E script finished."

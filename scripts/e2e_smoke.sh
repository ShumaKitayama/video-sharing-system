#!/usr/bin/env bash
#
# End-to-end smoke test against the running Docker stack.
#
# This is a black-box check: it talks to the real api container over HTTP exactly
# like a browser would, and verifies the side effects (a DB row and an on-disk
# video file) by peeking into the db and api containers.
#
# Prerequisites:
#   docker compose up -d --build
#
# Run:
#   bash scripts/e2e_smoke.sh
#
# Optional environment overrides:
#   BASE_URL   (default: http://localhost:8080)
#   COMPOSE    (default: "docker compose")

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
COMPOSE="${COMPOSE:-docker compose}"
API="${BASE_URL}/api/v1"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SEED_SQL="${SCRIPT_DIR}/seed_test_users.sql"

PASSWORD="${E2E_PASSWORD:-e2e-pass-123}"
WORK_DIR="$(mktemp -d)"
COOKIE_JAR="${WORK_DIR}/cookies.txt"
CLIP="${WORK_DIR}/clip.mp4"

pass_count=0
fail_count=0

cleanup() {
  rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

ok() {
  echo "  PASS: $1"
  pass_count=$((pass_count + 1))
}

fail() {
  echo "  FAIL: $1"
  fail_count=$((fail_count + 1))
}

section() {
  echo ""
  echo "== $1 =="
}

# assert_status <expected> <actual> <message>
assert_status() {
  if [ "$2" = "$1" ]; then
    ok "$3 (HTTP $2)"
  else
    fail "$3 (expected HTTP $1, got $2)"
  fi
}

# Extract the first "id" value from a JSON blob without requiring jq.
extract_id() {
  grep -o '"id":"[^"]*"' | head -n1 | sed 's/.*:"//; s/"$//'
}

# --------------------------------------------------------------------------
section "1. Health and readiness"

health_code="$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/health")"
assert_status 200 "${health_code}" "GET /health"

ready_code="$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/ready")"
assert_status 200 "${ready_code}" "GET /ready"

# --------------------------------------------------------------------------
section "2. Seed dedicated test users"

if ${COMPOSE} exec -T db psql -U postgres -d videoshare -v ON_ERROR_STOP=1 < "${SEED_SQL}" >/dev/null; then
  ok "Seeded teacher_test / student_test"
else
  fail "Seeding test users failed"
  echo "Aborting: cannot continue without test users."
  exit 1
fi

# --------------------------------------------------------------------------
section "3. Login as student_test"

login_code="$(curl -s -o "${WORK_DIR}/login.json" -w '%{http_code}' \
  -c "${COOKIE_JAR}" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"student_test\",\"password\":\"${PASSWORD}\"}" \
  "${API}/auth/login")"
assert_status 200 "${login_code}" "POST /auth/login"

# --------------------------------------------------------------------------
section "4. Upload a video (multipart)"

# Build a minimal but valid MP4: bytes 4..8 must spell "ftyp" for the sniffer.
printf '\x00\x00\x00\x18ftypisom\x00\x00\x02\x00isomiso2' > "${CLIP}"
# Pad to 4096 bytes so Range requests are meaningful.
dd if=/dev/zero bs=1 count=4072 >> "${CLIP}" 2>/dev/null
clip_size="$(wc -c < "${CLIP}" | tr -d ' ')"

upload_code="$(curl -s -o "${WORK_DIR}/upload.json" -w '%{http_code}' \
  -b "${COOKIE_JAR}" \
  -F 'title=E2E スモーク動画' \
  -F 'description=自動E2Eテスト' \
  -F "file=@${CLIP};type=video/mp4" \
  "${API}/videos")"
assert_status 201 "${upload_code}" "POST /videos"

VIDEO_ID="$(extract_id < "${WORK_DIR}/upload.json")"
if [ -n "${VIDEO_ID}" ]; then
  ok "Got video id ${VIDEO_ID}"
else
  fail "Could not parse video id from upload response"
  cat "${WORK_DIR}/upload.json"
  exit 1
fi

# --------------------------------------------------------------------------
section "5. Fetch detail and stream with Range"

get_code="$(curl -s -o /dev/null -w '%{http_code}' "${API}/videos/${VIDEO_ID}")"
assert_status 200 "${get_code}" "GET /videos/:id"

# Full stream.
full_code="$(curl -s -o /dev/null -w '%{http_code}' "${API}/videos/${VIDEO_ID}/stream")"
assert_status 200 "${full_code}" "GET /videos/:id/stream (full)"

# Range stream: expect 206 + a Content-Range header.
range_headers="$(curl -s -D - -o /dev/null -H 'Range: bytes=0-99' "${API}/videos/${VIDEO_ID}/stream")"
range_code="$(printf '%s' "${range_headers}" | head -n1 | awk '{print $2}')"
assert_status 206 "${range_code}" "GET /videos/:id/stream (Range)"

if printf '%s' "${range_headers}" | grep -qi '^Content-Range:'; then
  ok "Range response includes Content-Range header"
else
  fail "Range response missing Content-Range header"
fi

# --------------------------------------------------------------------------
section "6. Comment and like"

comment_code="$(curl -s -o /dev/null -w '%{http_code}' \
  -b "${COOKIE_JAR}" \
  -H 'Content-Type: application/json' \
  -d '{"body":"E2Eからのコメント"}' \
  "${API}/videos/${VIDEO_ID}/comments")"
assert_status 201 "${comment_code}" "POST /videos/:id/comments"

like_code="$(curl -s -o /dev/null -w '%{http_code}' \
  -b "${COOKIE_JAR}" \
  -X PUT \
  "${API}/videos/${VIDEO_ID}/like")"
assert_status 200 "${like_code}" "PUT /videos/:id/like"

# --------------------------------------------------------------------------
section "7. Permission check (anonymous upload rejected)"

anon_code="$(curl -s -o /dev/null -w '%{http_code}' \
  -F 'title=匿名' \
  -F "file=@${CLIP};type=video/mp4" \
  "${API}/videos")"
assert_status 401 "${anon_code}" "POST /videos without auth"

# --------------------------------------------------------------------------
section "8. Verify side effects (DB row + disk file)"

db_count="$(${COMPOSE} exec -T db psql -U postgres -d videoshare -tA \
  -c "SELECT COUNT(*) FROM videos WHERE public_id = '${VIDEO_ID}' AND deleted_at IS NULL;" | tr -d '[:space:]')"
if [ "${db_count}" = "1" ]; then
  ok "Video row exists in PostgreSQL"
else
  fail "Expected 1 video row, found '${db_count}'"
fi

STORAGE_KEY="$(${COMPOSE} exec -T db psql -U postgres -d videoshare -tA \
  -c "SELECT storage_key FROM videos WHERE public_id = '${VIDEO_ID}';" | tr -d '[:space:]')"

db_size="$(${COMPOSE} exec -T db psql -U postgres -d videoshare -tA \
  -c "SELECT file_size_bytes FROM videos WHERE public_id = '${VIDEO_ID}';" | tr -d '[:space:]')"
if [ "${db_size}" = "${clip_size}" ]; then
  ok "Stored file_size_bytes matches uploaded size (${clip_size})"
else
  fail "file_size_bytes mismatch (db=${db_size}, uploaded=${clip_size})"
fi

if [ -n "${STORAGE_KEY}" ] && ${COMPOSE} exec -T api ls "/app/uploads/${STORAGE_KEY}" >/dev/null 2>&1; then
  ok "Video file exists on disk at /app/uploads/${STORAGE_KEY}"
else
  fail "Video file not found on disk (storage_key='${STORAGE_KEY}')"
fi

# --------------------------------------------------------------------------
section "Summary"
echo "  Passed: ${pass_count}"
echo "  Failed: ${fail_count}"

if [ "${fail_count}" -ne 0 ]; then
  exit 1
fi
echo ""
echo "E2E smoke test PASSED."

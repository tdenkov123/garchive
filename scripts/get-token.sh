#!/usr/bin/env bash
set -euo pipefail

OWNER_ID="${1:-user-1}"
SECRET="${JWT_HMAC_SECRET:-$(openssl rand -hex 32)}"
ISSUER="${JWT_ISSUER:-garchive}"
AUDIENCE="${JWT_AUDIENCE:-garchive-api}"

payload=$(printf '{"sub":"%s","owner_id":"%s","iss":"%s","aud":"%s","exp":%s}' \
  "$OWNER_ID" "$OWNER_ID" "$ISSUER" "$AUDIENCE" "$(($(date +%s)+3600))")

header=$(printf '{"alg":"HS256","typ":"JWT"}' | openssl base64 -A | tr '+/' '-_' | tr -d '=')
body=$(printf '%s' "$payload" | openssl base64 -A | tr '+/' '-_' | tr -d '=')
sig=$(printf '%s.%s' "$header" "$body" | openssl dgst -sha256 -hmac "$SECRET" -binary | openssl base64 -A | tr '+/' '-_' | tr -d '=')

echo "${header}.${body}.${sig}"

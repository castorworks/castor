#!/usr/bin/env bash
# Usage: scripts/build-images.sh <load|push|tar> <image-prefix> <tag> [archive]
# Example: scripts/build-images.sh push registry.example.com/castor v1.0.0
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:?load|push|tar}"
PREFIX="${2:?image-prefix}"
TAG="${3:?tag}"
case "$MODE" in load|push|tar) ;; *) exit 2 ;; esac
ACTION=load
[[ "$MODE" == push ]] && ACTION=push
for APP in api web; do
  docker buildx build --platform "${PLATFORM:-linux/amd64}" \
    --"$ACTION" -f "$ROOT/apps/$APP/Dockerfile" \
    -t "${PREFIX%/}/castor-$APP:$TAG" "$ROOT/apps/$APP"
done
if [[ "$MODE" == tar ]]; then
  docker save "${PREFIX%/}/castor-api:$TAG" "${PREFIX%/}/castor-web:$TAG" | gzip > "${4:?archive path}"
fi

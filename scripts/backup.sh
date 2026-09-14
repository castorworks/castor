#!/usr/bin/env bash
# Cold snapshots for Compose-managed volumes only. External/K8s data uses provider backups.
# Usage: scripts/backup.sh <create|list|restore> <full|dev> [snapshot]
# Create/restore briefly stop this Compose project's services; previously running services restart.
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIR="$ROOT/deploy/compose"
BACKUPS="${BACKUP_ROOT:-$ROOT/deploy/backups}"
ACTION="${1:?create|list|restore}"
if [[ "$ACTION" == list ]]; then
  [[ ! -d "$BACKUPS" ]] || find "$BACKUPS" -mindepth 1 -maxdepth 1 -type d -name 'snapshot-*' -print
  exit 0
fi
STACK="${2:?full|dev}"
case "$STACK" in
  full) FILES=(-f "$DIR/compose.yaml" -f "$DIR/compose.infra.yaml") ;;
  dev) FILES=(-f "$DIR/compose.infra.yaml" -f "$DIR/compose.dev.yaml") ;;
  *) exit 2 ;;
esac
DC=(docker compose --project-directory "$DIR" --env-file "$DIR/.env" "${FILES[@]}")
SERVICES=(postgres redis minio)
CONTAINERS=()
for SERVICE in "${SERVICES[@]}"; do
  ID="$("${DC[@]}" ps -aq "$SERVICE")"
  [[ -n "$ID" ]] || exit 1
  CONTAINERS+=("$ID")
done
umask 077
case "$ACTION" in
  create)
    SNAPSHOT="$BACKUPS/snapshot-$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUPS"
    mkdir "$SNAPSHOT"
    ;;
  restore)
    NAME="${3:?snapshot name}"
    [[ "$NAME" =~ ^snapshot-[0-9]{8}_[0-9]{6}$ ]] || exit 2
    SNAPSHOT="$BACKUPS/$NAME"
    [[ -f "$SNAPSHOT/complete" ]] || exit 1
    # Explicit operator acknowledgement; never infer restore consent.
    [[ "${RESTORE_CONFIRM:-}" == "$NAME" ]] || { printf 'RESTORE_CONFIRM=%s\n' "$NAME" >&2; exit 2; }
    ;;
  *) exit 2 ;;
esac
# Pull helper before interrupting services.
docker pull alpine:3.21 >/dev/null
for INDEX in 0 1 2; do
  IMAGE_ID="$(docker inspect --format '{{.Image}}' "${CONTAINERS[$INDEX]}")"
  if [[ "$ACTION" == create ]]; then
    printf '%s\n' "$IMAGE_ID" > "$SNAPSHOT/${SERVICES[$INDEX]}.image"
  else
    [[ "$IMAGE_ID" == "$(cat "$SNAPSHOT/${SERVICES[$INDEX]}.image")" ]] || exit 1
    [[ -s "$SNAPSHOT/${SERVICES[$INDEX]}.tar" ]] || exit 1
    tar -tf "$SNAPSHOT/${SERVICES[$INDEX]}.tar" >/dev/null
  fi
done
RUNNING=()
while IFS= read -r SERVICE; do
  [[ -z "$SERVICE" ]] || RUNNING+=("$SERVICE")
done < <("${DC[@]}" ps --status running --services)
INFRA_RUNNING=()
APP_RUNNING=()
for SERVICE in "${RUNNING[@]}"; do
  case "$SERVICE" in
    postgres|redis|minio) INFRA_RUNNING+=("$SERVICE") ;;
    castor-api|castor-web|gateway) APP_RUNNING+=("$SERVICE") ;;
  esac
done
resume() {
  local status=$?
  # Failed restores remain stopped so partially restored data is never served.
  if [[ "$ACTION" == restore && "$status" != 0 ]]; then return; fi
  if (( ${#INFRA_RUNNING[@]} )); then "${DC[@]}" start --wait --wait-timeout 180 "${INFRA_RUNNING[@]}"; fi
  if (( ${#APP_RUNNING[@]} )); then "${DC[@]}" start --wait --wait-timeout 180 "${APP_RUNNING[@]}"; fi
}
trap resume EXIT
# Drain application requests before stopping the data services.
if (( ${#APP_RUNNING[@]} )); then "${DC[@]}" stop "${APP_RUNNING[@]}"; fi
if (( ${#INFRA_RUNNING[@]} )); then "${DC[@]}" stop "${INFRA_RUNNING[@]}"; fi
for INDEX in 0 1 2; do
  SERVICE="${SERVICES[$INDEX]}"
  DATA=/data
  [[ "$SERVICE" != postgres ]] || DATA=/var/lib/postgresql/data
  if [[ "$ACTION" == create ]]; then
    docker run --rm --volumes-from "${CONTAINERS[$INDEX]}:ro" -v "$SNAPSHOT:/backup" \
      alpine:3.21 tar -cf "/backup/$SERVICE.tar" -C "$DATA" .
  else
    docker run --rm --volumes-from "${CONTAINERS[$INDEX]}" -v "$SNAPSHOT:/backup:ro" \
      alpine:3.21 sh -ec 'find "$1" -mindepth 1 -maxdepth 1 -exec rm -rf {} \; ; tar -xf "$2" -C "$1"' sh "$DATA" "/backup/$SERVICE.tar"
  fi
done
if [[ "$ACTION" == create ]]; then
  tar -cf "$SNAPSHOT/config.tar" -C "$DIR" .env config compose.yaml compose.infra.yaml compose.dev.yaml
  touch "$SNAPSHOT/complete"
  printf '%s\n' "$SNAPSHOT"
fi
# Configuration is archived for inspection, not overwritten during data restore.

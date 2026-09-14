#!/usr/bin/env bash
# Local:  scripts/deploy-compose.sh <up|pull|config|down|rollback> <full|external|dev> [env-file]
# Remote: scripts/deploy-compose.sh remote <registry|tar> <host> <directory> <full|external> [archive]
# Remote mode uses deploy/compose/.env and config/api.config.toml; builds are separate.
# Remote arguments are validated below and intentionally expanded on the client.
# shellcheck disable=SC2029
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIR="$ROOT/deploy/compose"
ACTION="${1:?up|pull|config|down|rollback|remote}"
if [[ "$ACTION" == remote ]]; then
  MODE="${2:?registry|tar}"
  HOST="${3:?SSH host}"
  DEST="${4:?remote directory}"
  STACK="${5:?full|external}"
  case "$MODE" in registry|tar) ;; *) exit 2 ;; esac
  case "$STACK" in full|external) ;; *) exit 2 ;; esac
  # Restrict the destination to an absolute path safe for both scp and SSH.
  [[ "$DEST" =~ ^/[a-zA-Z0-9_./-]+$ && "$HOST" != -* ]] || exit 2
  "$0" config "$STACK" >/dev/null
  ssh "$HOST" "umask 077; mkdir -p '$DEST/deploy/compose/config' '$DEST/scripts'"
  tar -C "$ROOT" -cf - deploy/compose/compose.yaml deploy/compose/compose.infra.yaml \
    deploy/compose/compose.dev.yaml deploy/compose/config/nginx.conf \
    deploy/compose/config/redis.conf deploy/compose/config/api.config.toml \
    deploy/compose/.env scripts/deploy-compose.sh |
    ssh "$HOST" "umask 077; tar -xf - -C '$DEST'"
  # The API runs as UID 10001. The parent directory stays private to the deploy user.
  ssh "$HOST" "chmod 600 '$DEST/deploy/compose/.env'; chmod 644 '$DEST/deploy/compose/config/api.config.toml'"
  if [[ "$MODE" == tar ]]; then
    gzip -dc "${6:?archive path}" | ssh "$HOST" 'docker load'
  else
    ssh "$HOST" "bash '$DEST/scripts/deploy-compose.sh' pull '$STACK'"
  fi
  ssh "$HOST" "bash '$DEST/scripts/deploy-compose.sh' up '$STACK'"
  exit
fi
STACK="${2:?full|external|dev}"
ENV_FILE="${3:-$DIR/.env}"
FILES=()
case "$STACK" in
  full) FILES=(-f "$DIR/compose.yaml" -f "$DIR/compose.infra.yaml") ;;
  external) FILES=(-f "$DIR/compose.yaml") ;;
  dev) FILES=(-f "$DIR/compose.infra.yaml" -f "$DIR/compose.dev.yaml") ;;
  *) exit 2 ;;
esac
DC=(docker compose --project-directory "$DIR" --env-file "$ENV_FILE" "${FILES[@]}")
case "$ACTION" in
  config) "${DC[@]}" --profile tools config --quiet ;;
  pull) "${DC[@]}" --profile tools pull ;;
  down) "${DC[@]}" --profile tools down ;;
  rollback)
    [[ "$STACK" != dev ]] || exit 2
    # Operator has selected database-compatible images/config; never run old migrations.
    "${DC[@]}" up -d --force-recreate --wait --wait-timeout 180 castor-api castor-web gateway
    ;;
  up)
    "${DC[@]}" --profile tools config --quiet
    if [[ "$STACK" != external ]]; then
      "${DC[@]}" up -d --wait --wait-timeout 180 postgres redis minio
      "${DC[@]}" run --rm --no-deps minio-init
    fi
    if [[ "$STACK" != dev ]]; then
      # Always rerun for the requested image/config, even if an earlier run succeeded.
      "${DC[@]}" run --rm --no-deps init-db
      "${DC[@]}" up -d --force-recreate --wait --wait-timeout 180 castor-api castor-web gateway
    fi
    ;;
  *) exit 2 ;;
esac

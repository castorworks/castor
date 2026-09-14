#!/usr/bin/env bash
# Render: scripts/deploy-k8s.sh render <staging|production>
# Deploy: scripts/deploy-k8s.sh apply <staging|production> <kube-context>
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ACTION="${1:?render|apply|rollback}"
ENVIRONMENT="${2:?staging|production}"
case "$ENVIRONMENT" in staging|production) ;; *) exit 2 ;; esac
OVERLAY="$ROOT/deploy/k8s/overlays/$ENVIRONMENT"
if [[ "$ACTION" == render ]]; then
  exec kubectl kustomize "$OVERLAY"
fi
[[ "$ACTION" == apply || "$ACTION" == rollback ]] || exit 2
CONTEXT="${3:?explicit kube-context required}"
K=(kubectl --context "$CONTEXT")
NAMESPACE="castor-$ENVIRONMENT"
umask 077
MANIFEST="$(mktemp)"
LOCKED=false
cleanup() {
  rm -f "$MANIFEST"
  if [[ "$LOCKED" == true ]]; then "${K[@]}" -n "$NAMESPACE" delete configmap castor-release-lock --ignore-not-found; fi
}
trap cleanup EXIT
kubectl kustomize "$OVERLAY" > "$MANIFEST"
if grep -q 'REPLACE_WITH_RELEASE' "$MANIFEST"; then exit 2; fi
for FILE in secrets.env bootstrap.env; do
  while IFS='=' read -r KEY VALUE || [[ -n "$KEY" ]]; do
    [[ -z "$KEY" || "$KEY" == \#* ]] && continue
    [[ -n "$VALUE" ]] || { printf '%s\n' "$KEY" >&2; exit 2; }
    if [[ "$KEY" == CASTOR_RSA_SECRET ]]; then [[ ${#VALUE} == 32 ]] || exit 2; fi
    if [[ "$KEY" == CASTOR_DEFAULT_ADMIN_PASSWORD ]]; then [[ ${#VALUE} -ge 8 ]] || exit 2; fi
  done < "$OVERLAY/$FILE"
done
# Render once so config and images cannot drift between phases.
"${K[@]}" apply -f "$MANIFEST" -l castor.io/phase=namespace
"${K[@]}" -n "$NAMESPACE" create configmap castor-release-lock --from-literal=created-at="$(date -u +%FT%TZ)"
LOCKED=true
"${K[@]}" apply -f "$MANIFEST" -l castor.io/phase=configuration
if [[ "$ACTION" == apply ]]; then
  JOB="castor-init-$(date +%s)-${RANDOM}"
  "${K[@]}" -n "$NAMESPACE" create job "$JOB" --from=cronjob/castor-init-db
  if ! "${K[@]}" -n "$NAMESPACE" wait --for=condition=complete --timeout=700s "job/$JOB"; then
    "${K[@]}" -n "$NAMESPACE" logs "job/$JOB" --all-containers=true || true
    exit 1
  fi
fi
"${K[@]}" apply -f "$MANIFEST" -l castor.io/phase=application
"${K[@]}" -n "$NAMESPACE" rollout status deployment/castor-api --timeout=300s
"${K[@]}" -n "$NAMESPACE" rollout status deployment/castor-web --timeout=300s

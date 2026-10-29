#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${KUBE_OSPREY_E2E_PORT:-38080}"
if [ -n "${KUBE_OSPREY_E2E_DB_PATH:-}" ]; then
  DB_PATH="${KUBE_OSPREY_E2E_DB_PATH}"
else
  DB_PATH="$(mktemp "${TMPDIR:-/tmp}/kube-osprey-e2e.XXXXXX")"
fi

cd "${ROOT_DIR}"

rm -f "${DB_PATH}"

make build

export DB_TYPE=sqlite
export DB_DSN="${DB_PATH}"
export DISABLE_VERSION_CHECK=true
export JWT_SECRET="${JWT_SECRET:-kube-osprey-e2e-jwt-secret}"
export KUBE_OSPREY_ENCRYPT_KEY="${KUBE_OSPREY_ENCRYPT_KEY:-kube-osprey-e2e-encryption-key-must-be-32-chars}"
export PORT
export HOME="${KUBE_OSPREY_E2E_HOME:-$(mktemp -d "${TMPDIR:-/tmp}/kube-osprey-e2e-home.XXXXXX")}"
unset KUBE_OSPREY_USERNAME
unset KUBE_OSPREY_PASSWORD
unset KUBECONFIG

exec ./kube-osprey -v 3

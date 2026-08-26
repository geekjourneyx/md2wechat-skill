#!/usr/bin/env bash
set -euo pipefail

MD2WECHAT_CLI_COMMIT="${MD2WECHAT_CLI_COMMIT:-$(git rev-parse HEAD)}"
output="${LAYOUT_CONFORMANCE_OUTPUT:-/tmp/md2wechat-layout-conformance.jsonl}"
mode="${MD2WECHAT_LAYOUT_CONFORMANCE_MODE:-release}"
pinned_field_contract_sha="052346a43deb83d211471bb7b423318f6f6ff6c1"

if [[ "$mode" == "release" ]]; then
  : "${MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA:?release conformance requires MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA}"
  : "${MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT:?release conformance requires MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT=passed}"
  [[ "$MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT" == "passed" ]] \
    || { echo "release conformance requires MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT=passed" >&2; exit 2; }
  [[ "$MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA" == "$pinned_field_contract_sha" ]] \
    || { echo "release conformance requires MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA=$pinned_field_contract_sha" >&2; exit 2; }
fi

MD2WECHAT_E2E=1 \
MD2WECHAT_CLI_COMMIT="$MD2WECHAT_CLI_COMMIT" \
MD2WECHAT_BASE_URL="${MD2WECHAT_BASE_URL:-}" \
MD2WECHAT_API_BUILD_ID="${MD2WECHAT_API_BUILD_ID:-}" \
MD2WECHAT_LAYOUT_CONFORMANCE_MODE="$mode" \
MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA="${MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA:-}" \
MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT="${MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT:-}" \
GOCACHE="${GOCACHE:-/tmp/md2wechat-go-build}" \
go test -timeout=6m -json ./cmd/md2wechat -run '^(TestE2ELayoutConformance|TestE2ECompactLayoutBoundaryAndThemeProbes)$' -count=1 | tee "$output"

printf 'layout conformance normalized target evidence: conformance_target_normalized (emitted by Go test)\n'
printf 'layout conformance mode: %s\n' "$mode"
if [[ "$mode" == "release" ]]; then
  printf 'layout conformance upstream field-contract evidence: sha=%s result=%s\n' "$MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA" "$MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT"
fi
printf 'layout conformance report: %s\n' "$output"

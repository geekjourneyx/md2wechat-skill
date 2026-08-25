#!/usr/bin/env bash
set -euo pipefail

MD2WECHAT_CLI_COMMIT="${MD2WECHAT_CLI_COMMIT:-$(git rev-parse HEAD)}"
output="${LAYOUT_CONFORMANCE_OUTPUT:-/tmp/md2wechat-layout-conformance.jsonl}"
target="${MD2WECHAT_BASE_URL:-https://www.md2wechat.cn/api/convert}"

MD2WECHAT_E2E=1 \
MD2WECHAT_CLI_COMMIT="$MD2WECHAT_CLI_COMMIT" \
MD2WECHAT_BASE_URL="${MD2WECHAT_BASE_URL:-}" \
MD2WECHAT_API_BUILD_ID="${MD2WECHAT_API_BUILD_ID:-}" \
GOCACHE="${GOCACHE:-/tmp/md2wechat-go-build}" \
go test -timeout=6m -json ./cmd/md2wechat -run '^(TestE2ELayoutConformance|TestE2ECompactLayoutBoundaryAndThemeProbes)$' -count=1 | tee "$output"

printf 'layout conformance target: %s\n' "$target"
printf 'layout conformance report: %s\n' "$output"

#!/usr/bin/env bash
set -euo pipefail

MD2WECHAT_CLI_COMMIT="${MD2WECHAT_CLI_COMMIT:-$(git rev-parse HEAD)}"
output="${LAYOUT_CONFORMANCE_OUTPUT:-/tmp/md2wechat-layout-conformance.jsonl}"

MD2WECHAT_E2E=1 \
MD2WECHAT_CLI_COMMIT="$MD2WECHAT_CLI_COMMIT" \
GOCACHE="${GOCACHE:-/tmp/md2wechat-go-build}" \
go test -json ./cmd/md2wechat -run 'TestE2E(Layout|Compatibility)Conformance' -count=1 | tee "$output"

printf 'layout conformance target: https://md2wechat.app\n'
printf 'layout conformance report: %s\n' "$output"

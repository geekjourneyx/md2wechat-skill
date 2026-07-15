#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

fail() {
  echo "release-check: $*" >&2
  exit 1
}

require_file() {
  local path="$1"
  [[ -f "$path" ]] || fail "missing required file: $path"
}

require_file "VERSION"
require_file "CHANGELOG.md"
require_file "README.md"
require_file "package.json"
require_file "docs/INSTALL.md"
require_file "docs/DISCOVERY.md"
require_file "docs/FAQ.md"
require_file "docs/QUICKSTART.md"
require_file "docs/USAGE.md"
require_file "scripts/install.sh"
require_file "scripts/install.js"
require_file "scripts/install.ps1"
require_file "scripts/install-openclaw.ps1"
require_file "scripts/run.js"
require_file "scripts/quality-gates.sh"
require_file "scripts/run-golangci-lint.sh"
require_file "scripts/generate-homebrew-formula.sh"
require_file "skills/md2wechat/SKILL.md"
require_file "platforms/openclaw/md2wechat/SKILL.md"
require_file ".github/workflows/release.yml"
require_file ".github/workflows/ci.yml"
require_file "AGENTS.md"
require_file "docs/OPENCLAW.md"
require_file ".claude-plugin/marketplace.json"

[[ ! -e "scripts/install-openclaw.sh" ]] \
  || fail "scripts/install-openclaw.sh was removed in v3.2.0 and must not be restored"
if rg -q 'install-openclaw\.sh' --glob '!CHANGELOG.md' --glob '!scripts/release-check.sh' .; then
  fail "current release, documentation, and repository contracts must not reference the removed install-openclaw.sh"
fi

version="$(tr -d '[:space:]' < VERSION)"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "VERSION must be SemVer, got: $version"

changelog_version="$(sed -n 's/^## \[\([0-9][^]]*\)\].*/\1/p' CHANGELOG.md | head -n1)"
[[ -n "$changelog_version" ]] || fail "failed to read top CHANGELOG version"
[[ "$changelog_version" == "$version" ]] || fail "VERSION ($version) does not match CHANGELOG top version ($changelog_version)"

marketplace_version="$(sed -n 's/.*"version": "\([0-9][^"]*\)".*/\1/p' .claude-plugin/marketplace.json | head -n1)"
[[ -n "$marketplace_version" ]] || fail "failed to read marketplace plugin version"
[[ "$marketplace_version" == "$version" ]] || fail "VERSION ($version) does not match marketplace version ($marketplace_version)"

package_version="$(node -p "require('./package.json').version")"
[[ -n "$package_version" ]] || fail "failed to read package.json version"
[[ "$package_version" == "$version" ]] || fail "VERSION ($version) does not match package.json version ($package_version)"

package_name="$(node -p "require('./package.json').name")"
[[ "$package_name" == "@geekjourneyx/md2wechat" ]] || fail "package.json name must stay @geekjourneyx/md2wechat"

if ! node -e "const pkg=require('./package.json'); if ('os' in pkg || 'cpu' in pkg) process.exit(1)"; then
  fail "package.json must not claim a global os/cpu matrix; scripts/install.js is the source of truth for supported npm targets"
fi

grep -q 'MD2WECHAT_VERSION' scripts/install.sh || fail "scripts/install.sh must support MD2WECHAT_VERSION"
grep -q 'MD2WECHAT_RELEASE_BASE_URL' scripts/install.sh || fail "scripts/install.sh must support MD2WECHAT_RELEASE_BASE_URL"
grep -q 'MD2WECHAT_INSTALL_DIR' scripts/install.sh || fail "scripts/install.sh must support MD2WECHAT_INSTALL_DIR"
grep -q 'MD2WECHAT_RELEASE_BASE_URL' scripts/install.js || fail "scripts/install.js must support MD2WECHAT_RELEASE_BASE_URL"
grep -q 'checksums.txt' scripts/install.js || fail "scripts/install.js must verify checksums.txt"
grep -q 'npm install -g @geekjourneyx/md2wechat' scripts/run.js || fail "scripts/run.js must point users at the npm install command"
grep -q 'MD2WECHAT_VERSION' scripts/install.ps1 || fail "scripts/install.ps1 must support MD2WECHAT_VERSION"
grep -q 'MD2WECHAT_RELEASE_BASE_URL' scripts/install.ps1 || fail "scripts/install.ps1 must support MD2WECHAT_RELEASE_BASE_URL"
grep -q 'MD2WECHAT_INSTALL_DIR' scripts/install.ps1 || fail "scripts/install.ps1 must support MD2WECHAT_INSTALL_DIR"
grep -q 'MD2WECHAT_NONINTERACTIVE' scripts/install.ps1 || fail "scripts/install.ps1 must support MD2WECHAT_NONINTERACTIVE"
grep -q 'MD2WECHAT_NO_PATH_UPDATE' scripts/install.ps1 || fail "scripts/install.ps1 must support MD2WECHAT_NO_PATH_UPDATE"
grep -q 'MD2WECHAT_RELEASE_BASE_URL' scripts/install-openclaw.ps1 || fail "scripts/install-openclaw.ps1 must support MD2WECHAT_RELEASE_BASE_URL"
grep -q 'MD2WECHAT_VERSION' scripts/install-openclaw.ps1 || fail "scripts/install-openclaw.ps1 must support MD2WECHAT_VERSION"
grep -q 'MD2WECHAT_INSTALL_DIR' scripts/install-openclaw.ps1 || fail "scripts/install-openclaw.ps1 must support MD2WECHAT_INSTALL_DIR"
grep -q 'MD2WECHAT_OPENCLAW_INSTALL_DIR' scripts/install-openclaw.ps1 || fail "scripts/install-openclaw.ps1 must support MD2WECHAT_OPENCLAW_INSTALL_DIR"
grep -q 'MD2WECHAT_NONINTERACTIVE' scripts/install-openclaw.ps1 || fail "scripts/install-openclaw.ps1 must support MD2WECHAT_NONINTERACTIVE"
grep -q 'MD2WECHAT_NO_PATH_UPDATE' scripts/install-openclaw.ps1 || fail "scripts/install-openclaw.ps1 must support MD2WECHAT_NO_PATH_UPDATE"
grep -q 'checksums.txt' scripts/install.sh || fail "scripts/install.sh must verify checksums.txt"
grep -q 'checksums.txt' scripts/install.ps1 || fail "scripts/install.ps1 must verify checksums.txt"
grep -q 'checksums.txt' scripts/install-openclaw.ps1 || fail "scripts/install-openclaw.ps1 must verify checksums.txt"
grep -q 'md2wechat-openclaw-skill.tar.gz' scripts/install-openclaw.ps1 || fail "scripts/install-openclaw.ps1 must install the OpenClaw release archive"
grep -q 'md2wechat-windows-amd64.exe' scripts/install-openclaw.ps1 || fail "scripts/install-openclaw.ps1 must install a versioned md2wechat CLI"
grep -q 'upload-artifact' .github/workflows/release.yml || fail "release workflow must upload a release artifact"
grep -q 'download-artifact' .github/workflows/release.yml || fail "release workflow must download a release artifact"
grep -q 'npm pack' .github/workflows/release.yml || fail "release workflow must smoke the npm package"
grep -q 'npm install -g' .github/workflows/release.yml || fail "release workflow must smoke npm global install"
grep -q 'npm publish --access public' .github/workflows/release.yml || fail "release workflow must publish the npm package"
grep -q 'NPM_TOKEN' .github/workflows/release.yml || fail "release workflow must require NPM_TOKEN for npm publish"
grep -q 'install.sh' .github/workflows/release.yml || fail "release workflow must publish install.sh"
grep -q 'install.ps1' .github/workflows/release.yml || fail "release workflow must publish install.ps1"
grep -q 'install-openclaw.ps1' .github/workflows/release.yml || fail "release workflow must publish install-openclaw.ps1"
grep -q 'md2wechat-openclaw-skill.tar.gz' .github/workflows/release.yml || fail "release workflow must publish the OpenClaw skill archive"
grep -q 'md2wechat_Darwin_arm64.tar.gz' .github/workflows/release.yml || fail "release workflow must publish the macOS arm64 Homebrew archive"
grep -q 'md2wechat_Darwin_x86_64.tar.gz' .github/workflows/release.yml || fail "release workflow must publish the macOS x86_64 Homebrew archive"
grep -q 'md2wechat_Linux_arm64.tar.gz' .github/workflows/release.yml || fail "release workflow must publish the Linux arm64 Homebrew archive"
grep -q 'md2wechat_Linux_x86_64.tar.gz' .github/workflows/release.yml || fail "release workflow must publish the Linux x86_64 Homebrew archive"
grep -q 'platforms/openclaw/md2wechat' .github/workflows/release.yml || fail "release workflow must package the OpenClaw-specific skill"
grep -q 'version --json' .github/workflows/release.yml || fail "release workflow must smoke version --json"
grep -q 'MD2WECHAT_RELEASE_BASE_URL' .github/workflows/release.yml || fail "release workflow must smoke the installers from the same bundle"
grep -q 'HOMEBREW_TAP_GITHUB_TOKEN' .github/workflows/release.yml || fail "release workflow must update the tap repository with HOMEBREW_TAP_GITHUB_TOKEN"
grep -q 'generate-homebrew-formula.sh' .github/workflows/release.yml || fail "release workflow must generate the Homebrew formula via scripts/generate-homebrew-formula.sh"
if ! node - <<'NODE'
const fs = require('fs');

const path = 'platforms/openclaw/md2wechat/SKILL.md';
const source = fs.readFileSync(path, 'utf8');
const frontmatterMatch = source.match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/);

function reject(message) {
  console.error(`release-check: ${message}`);
  process.exit(1);
}

if (!frontmatterMatch) {
  reject('OpenClaw skill must begin with complete YAML frontmatter');
}

const frontmatter = frontmatterMatch[1];
if (/WECHAT_(APPID|SECRET)/.test(frontmatter)) {
  reject('OpenClaw skill frontmatter must not globally require WeChat publish credentials');
}
if (/@latest\b/.test(frontmatter)) {
  reject('OpenClaw skill frontmatter must not contain @latest');
}

const metadataMatch = /^metadata:\s*/m.exec(frontmatter);
if (!metadataMatch) {
  reject('OpenClaw skill frontmatter must declare metadata');
}

const metadataStart = metadataMatch.index + metadataMatch[0].length;
if (frontmatter[metadataStart] !== '{') {
  reject('OpenClaw skill metadata must be a JSON object');
}

let depth = 0;
let inString = false;
let escaped = false;
let metadataEnd = -1;
for (let i = metadataStart; i < frontmatter.length; i += 1) {
  const char = frontmatter[i];
  if (inString) {
    if (escaped) {
      escaped = false;
    } else if (char === '\\') {
      escaped = true;
    } else if (char === '"') {
      inString = false;
    }
    continue;
  }
  if (char === '"') {
    inString = true;
  } else if (char === '{') {
    depth += 1;
  } else if (char === '}') {
    depth -= 1;
    if (depth === 0) {
      metadataEnd = i + 1;
      break;
    }
  }
}

if (metadataEnd === -1) {
  reject('OpenClaw skill metadata JSON is incomplete');
}

let metadata;
try {
  metadata = JSON.parse(frontmatter.slice(metadataStart, metadataEnd));
} catch (error) {
  reject(`OpenClaw skill metadata JSON is invalid: ${error.message}`);
}

const expected = {
  openclaw: {
    emoji: '📝',
    requires: { bins: ['md2wechat'] },
    install: [{
      id: 'node',
      kind: 'node',
      package: '@geekjourneyx/md2wechat',
      bins: ['md2wechat'],
      label: 'Install md2wechat (npm)',
    }],
  },
};
if (JSON.stringify(metadata) !== JSON.stringify(expected)) {
  reject('OpenClaw skill metadata must contain only the exact metadata.openclaw npm install contract');
}
NODE
then
  fail "OpenClaw skill frontmatter contract failed"
fi

for release_doc in docs/INSTALL.md docs/OPENCLAW.md; do
  grep -Fq "releases/download/v${version}" "$release_doc" \
    || fail "$release_doc must point install instructions at v${version} release assets"
  ! grep -Fq 'releases/latest/download' "$release_doc" \
    || fail "$release_doc must not contain releases/latest/download"
  stale_release_url="$(grep -Eo 'releases/download/v[0-9]+\.[0-9]+\.[0-9]+' "$release_doc" \
    | grep -Fvx "releases/download/v${version}" \
    | head -n1 || true)"
  [[ -z "$stale_release_url" ]] \
    || fail "$release_doc contains a stale versioned release URL: $stale_release_url"
done
grep -q 'npm install -g @geekjourneyx/md2wechat' README.md || fail "README must document the npm install path"
grep -q 'npm install -g @geekjourneyx/md2wechat' docs/INSTALL.md || fail "docs/INSTALL.md must document the npm install path"
grep -q 'npx cnpm sync @geekjourneyx/md2wechat' docs/INSTALL.md || fail "docs/INSTALL.md must document the post-publish npmmirror sync step"
grep -q 'npx cnpm sync @geekjourneyx/md2wechat' docs/FAQ.md || fail "docs/FAQ.md must document the npmmirror sync recovery path"
! grep -q 'brew install geekjourneyx/tap/md2wechat' README.md || fail "README must keep Homebrew install details in docs/INSTALL.md, not the primary quickstart"
grep -q 'brew install geekjourneyx/tap/md2wechat' docs/INSTALL.md || fail "docs/INSTALL.md must document the tap install path"
! grep -q 'raw.githubusercontent.com/geekjourneyx/md2wechat-skill/main/scripts/install.sh' README.md || fail "README must not point the md2wechat installer at main"
! grep -q 'raw.githubusercontent.com/geekjourneyx/md2wechat-skill/main/scripts/install.ps1' README.md || fail "README must not point the md2wechat installer at main"
! grep -q 'raw.githubusercontent.com/geekjourneyx/md2wechat-skill/main/scripts/install.sh' docs/INSTALL.md || fail "docs/INSTALL.md must not point the md2wechat installer at main"
! grep -q 'raw.githubusercontent.com/geekjourneyx/md2wechat-skill/main/scripts/install.ps1' docs/INSTALL.md || fail "docs/INSTALL.md must not point the md2wechat installer at main"
grep -q 'md2wechat inspect article.md' README.md || fail "README must document inspect as part of the confirm-first flow"
grep -q 'md2wechat preview article.md' README.md || fail "README must document preview as part of the confirm-first flow"
grep -q 'md2wechat inspect article.md' docs/QUICKSTART.md || fail "docs/QUICKSTART.md must document inspect"
grep -q 'md2wechat preview article.md' docs/QUICKSTART.md || fail "docs/QUICKSTART.md must document preview"
grep -q 'md2wechat inspect article.md --json' docs/DISCOVERY.md || fail "docs/DISCOVERY.md must document inspect --json"
grep -q 'md2wechat preview article.md --json' docs/DISCOVERY.md || fail "docs/DISCOVERY.md must document preview --json"
grep -q 'md2wechat inspect article.md' docs/FAQ.md || fail "docs/FAQ.md must document inspect"
grep -q 'md2wechat preview article.md' docs/FAQ.md || fail "docs/FAQ.md must document preview"
grep -q 'md2wechat inspect article.md' docs/USAGE.md || fail "docs/USAGE.md must document inspect"
grep -q 'md2wechat preview article.md' docs/USAGE.md || fail "docs/USAGE.md must document preview"
grep -q 'md2wechat skills list --json' README.md || fail "README must document embedded skills discovery"
grep -q 'md2wechat skills read md2wechat --json' README.md || fail "README must document reading the embedded md2wechat SOP"
grep -q 'md2wechat skills list --json' docs/DISCOVERY.md || fail "docs/DISCOVERY.md must document embedded skills discovery"
grep -q 'md2wechat skills read md2wechat --json' docs/DISCOVERY.md || fail "docs/DISCOVERY.md must document reading the embedded md2wechat SOP"
grep -q 'md2wechat skills read md2wechat --json' docs/AGENT-GUIDE.md || fail "docs/AGENT-GUIDE.md must document reading the embedded md2wechat SOP"
grep -q 'md2wechat skills read md2wechat --json' docs/FAQ.md || fail "docs/FAQ.md must document reading the embedded md2wechat SOP"
grep -q 'md2wechat skills read md2wechat --json' docs/INSTALL.md || fail "docs/INSTALL.md must document embedded SOP verification after install"
grep -q 'md2wechat skills read md2wechat --json' docs/QUICKSTART.md || fail "docs/QUICKSTART.md must document embedded SOP verification"
grep -q 'md2wechat skills read md2wechat --json' docs/USAGE.md || fail "docs/USAGE.md must document embedded SOP verification for Agent setup"
grep -q 'md2wechat skills read md2wechat --json' docs/SMOKE.md || fail "docs/SMOKE.md must include embedded SOP verification in smoke context"
grep -q 'md2wechat skills read md2wechat --json' docs/OBSIDIAN.md || fail "docs/OBSIDIAN.md must document embedded SOP verification"
grep -q 'md2wechat skills read md2wechat --json' docs/OPENCLAW.md || fail "docs/OPENCLAW.md must document embedded SOP verification"
grep -q 'skills read md2wechat --json' scripts/install-openclaw.ps1 || fail "install-openclaw.ps1 must print embedded SOP verification"
grep -q '`inspect` is the source-of-truth command' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must describe inspect as the source-of-truth command"
grep -q '`preview` writes only byte-identical final API HTML from a successful converter result' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must describe exact API preview behavior"
grep -q 'this invocation does not create or overwrite preview HTML' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must describe action-required and failed preview no-write behavior"
grep -q 'PREVIEW_ACTION_REQUIRED' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must describe PREVIEW_ACTION_REQUIRED"
grep -q 'PREVIEW_FAILED' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must describe PREVIEW_FAILED"
grep -q 'PREVIEW_ACTION_REQUIRED.*empty `data.output_file`' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must describe action-required no-output behavior"
grep -q 'Any pre-existing explicit output path is stale' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must reject stale explicit preview output"
grep -q 'WeChat upload, article draft creation, and `create_image_post` require WeChat credentials' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must require credentials for every explicit WeChat side effect"
grep -q 'free of any global WeChat publishing credential requirement' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must distinguish publishing credentials from API-mode credentials"
grep -q 'API-mode preview and conversion require a valid `MD2WECHAT_API_KEY`' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must retain API-mode credential requirements"
grep -q '`create_image_post` effects' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must include create_image_post in named-account pre-effect validation"
grep -q 'md2wechat skills read md2wechat --json' skills/md2wechat/SKILL.md || fail "skills/md2wechat/SKILL.md must route stale-SOP checks through skills read"
grep -q '`inspect` is the source-of-truth command' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must describe inspect as the source-of-truth command"
grep -q '`preview` writes only byte-identical final API HTML from a successful converter result' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must describe exact API preview behavior"
grep -q 'this invocation does not create or overwrite preview HTML' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must describe action-required and failed preview no-write behavior"
grep -q 'PREVIEW_ACTION_REQUIRED' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must describe PREVIEW_ACTION_REQUIRED"
grep -q 'PREVIEW_FAILED' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must describe PREVIEW_FAILED"
grep -q 'PREVIEW_ACTION_REQUIRED.*empty `data.output_file`' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must describe action-required no-output behavior"
grep -q 'Any pre-existing explicit output path is stale' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must reject stale explicit preview output"
grep -q 'WeChat upload, article draft creation, and `create_image_post` require WeChat credentials' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must require credentials for every explicit WeChat side effect"
grep -q 'free of any global WeChat publishing credential requirement' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must distinguish publishing credentials from API-mode credentials"
grep -q 'API-mode preview and conversion require a valid `MD2WECHAT_API_KEY`' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must retain API-mode credential requirements"
grep -q '`create_image_post` effects' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must include create_image_post in named-account pre-effect validation"
grep -q 'md2wechat skills read md2wechat --json' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must route stale-SOP checks through skills read"
GOCACHE="${GOCACHE:-/tmp/md2wechat-go-build}" go test ./cmd/md2wechat -run '^TestLayoutDocumentation' -count=1 >/dev/null || fail "layout documentation contracts must pass"
grep -q -- '--body-file' docs/LAYOUT.md || fail "LAYOUT must document complex body input"
grep -q 'layout list --lifecycle compatibility --json' skills/md2wechat/SKILL.md || fail "embedded skill must isolate compatibility layouts"
grep -q 'layout list --lifecycle compatibility --json' platforms/openclaw/md2wechat/SKILL.md || fail "OpenClaw skill must isolate compatibility layouts"
grep -q 'bash scripts/quality-gates.sh' .github/workflows/quality-gates.yml || fail ".github/workflows/quality-gates.yml must use the shared scripts/quality-gates.sh gate"
grep -q 'VERSION' Makefile || fail "Makefile must reference VERSION"
grep -Fq 'github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v$(VERSION)' Makefile \
  || fail "make help must show a VERSION-pinned Go install command"
! grep -Fq 'github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@latest' Makefile \
  || fail "make help must not show @latest for Go install"
grep -Fq 'required for explicit WeChat publishing effects' cmd/md2wechat/main.go \
  || fail "root help must scope WeChat credentials to explicit publishing effects"
! grep -Eq 'WECHAT_(APPID|SECRET).*\(required\)$' cmd/md2wechat/main.go \
  || fail "root help must not describe WeChat credentials as globally required"
grep -q 'artifact smoke' AGENTS.md || fail "AGENTS.md must mention artifact smoke"

echo "release-check: OK (version $version)"

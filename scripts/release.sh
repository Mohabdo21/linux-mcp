#!/usr/bin/env bash
#
# Release script for linux-mcp.
#
# Cuts a versioned release from a clean checkout of the default branch:
#   tests -> build -> sign (Sigstore keyless) -> update server.json ->
#   commit -> tag -> push -> GitHub release -> publish to MCP registry.
#
# Usage:
#   scripts/release.sh [--force] [VERSION]

set -euo pipefail

EXPECTED_REPO="Mohabdo21/linux-mcp"
BINARIES=("bin/linux-mcp" "bin/linux-mcp_static")
VERSION=""
FORCE=false
CHANGELOG_FILE="$(mktemp "${TMPDIR:-/tmp}/linux-mcp-release.XXXXXX")"

trap 'rm -f -- "$CHANGELOG_FILE"' EXIT

# logging

if [[ -t 1 ]]; then
	RED=$'\033[0;31m'
	GREEN=$'\033[0;32m'
	YELLOW=$'\033[1;33m'
	NC=$'\033[0m'
else
	RED=''
	GREEN=''
	YELLOW=''
	NC=''
fi

info() { printf '%s[INFO]%s %s\n' "$GREEN" "$NC" "$*"; }
warn() { printf '%s[WARN]%s %s\n' "$YELLOW" "$NC" "$*"; }
die() {
	printf '%s[ERROR]%s %s\n' "$RED" "$NC" "$*" >&2
	exit 1
}

# utilities

usage() {
	cat <<EOF
Usage: $(basename "$0") [--force] [VERSION]

Cut a versioned release of linux-mcp.

Options:
  VERSION    Release version, e.g. v1.2.3 (default: latest tag + patch bump)
  --force    Re-publish an existing tag: replace the tag/release and re-upload
  -h, --help Show this help

Run from a clean checkout of the default branch (main).
EOF
}

require_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "'$1' not found in PATH. Install it: $2"
}

confirm() {
	local input
	read -r -p "$1 [y/N] " input || return 1
	[[ $input =~ ^[yY]$ ]]
}

latest_tag() {
	git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

version_ok() {
	[[ $1 =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

remote_is_canonical() {
	local origin canonical
	origin=$(git remote get-url origin)
	canonical=$(sed -E 's#\.git$##' <<<"$origin" |
		sed -E -n 's#.*[:/]([^/]*)/([^/]*)$#\1/\2#p')
	[[ $canonical == "$EXPECTED_REPO" ]]
}

has_push_permission() {
	local user perm
	user=$(gh api user --jq .login 2>/dev/null) || return 1
	perm=$(gh api "repos/$EXPECTED_REPO/collaborators/$user" --jq .permissions.push 2>/dev/null) || return 1
	[[ $perm == true ]]
}

# steps

parse_args() {
	while (($#)); do
		case $1 in
		--force) FORCE=true ;;
		-h | --help)
			usage
			exit 0
			;;
		--)
			shift
			if (($# > 1)); then die "too many arguments: $*"; fi
			VERSION=${1:-}
			;;
		-*) die "unknown option: $1 (run '$(basename "$0") --help')" ;;
		*)
			[[ -z $VERSION ]] || die "too many arguments: $1"
			VERSION=$1
			;;
		esac
		shift
	done
}

preflight() {
	[[ -z $(git status --porcelain) ]] ||
		die "working tree is not clean. Commit or stash changes first."
	local branch
	branch=$(git rev-parse --abbrev-ref HEAD)
	[[ $branch == main ]] || die "releases must be cut from 'main' (currently on '$branch')"
	remote_is_canonical || die "origin must be $EXPECTED_REPO (refusing to release to a fork/mirror)"
	require_cmd gh "https://cli.github.com/"
	require_cmd cosign "https://docs.sigstore.dev/cosign/installation/"
	require_cmd jq "https://jqlang.github.io/jq/"
	require_cmd openssl "https://www.openssl.org/"
	require_cmd mcp-publisher "https://github.com/modelcontextprotocol/registry"
	gh auth status >/dev/null 2>&1 || die "gh is not authenticated. Run 'gh auth login'."
	has_push_permission || die "current gh user has no push rights on $EXPECTED_REPO"
	[[ -n ${MCP_GITHUB_TOKEN:-} ]] ||
		die "MCP_GITHUB_TOKEN is not set. Create a PAT at https://github.com/settings/tokens/new (repo + read:user) and export it."
	[[ -f server.json ]] || die "server.json not found at the repository root"
	git fetch --tags origin
	info "pre-flight checks passed"
}

resolve_version() {
	local latest
	latest=$(latest_tag)

	if [[ -z $VERSION ]]; then
		local base=${latest#v}
		base=${base%%-*}
		local major minor patch
		IFS='.' read -r major minor patch <<<"$base"
		major=${major:-0}
		minor=${minor:-0}
		patch=${patch:-0}
		VERSION="v$major.$minor.$((patch + 1))"
		info "no version given: releasing $VERSION (latest tag: $latest)"
		local input
		read -r -p "Press Enter to use $VERSION, or type a different version: " input || die "aborted"
		VERSION=${input:-$VERSION}
	fi

	version_ok "$VERSION" || die "version must match 'vX.Y.Z', got: $VERSION"

	if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null 2>&1; then
		local on_remote=""
		git ls-remote --tags origin "$VERSION" 2>/dev/null | grep -q . && on_remote=" and on origin"
		warn "tag $VERSION already exists$on_remote"
		if confirm "force re-release this tag?"; then
			FORCE=true
		else
			die "aborted. Use a different version or pass --force."
		fi
	fi

	if [[ $(printf '%s\n' "$latest" "$VERSION" | sort -V | tail -n 1) != "$VERSION" ]]; then
		warn "$VERSION is older than the latest tag ($latest)"
		confirm "continue anyway?" || die "aborted"
	fi

	info "releasing: $VERSION"
}

generate_changelog() {
	local prev body
	prev=$(git describe --tags --abbrev=0 2>/dev/null || true)
	if [[ -z $prev ]]; then
		body=$(git log --format="format:- %s (%h)" --reverse)
		info "first release: including all commits"
	else
		body=$(git log --format="format:- %s (%h)" "$prev"..HEAD)
		info "changelog: commits since $prev"
	fi
	if [[ -z $body ]]; then
		confirm "no new commits. Release anyway?" || die "aborted"
	fi
	printf '# Release %s\n\n%s\n' "$VERSION" "$body" >"$CHANGELOG_FILE"
}

build() {
	info "running tests"
	make test
	info "building binaries"
	make build build-static VERSION="$VERSION"
	local f
	for f in "${BINARIES[@]}"; do
		[[ -f $f ]] || die "build artifact missing: $f"
	done
}

sign() {
	info "signing binaries with cosign (keyless)"
	info "a browser window will open for GitHub authentication"
	local b
	for b in "${BINARIES[@]}"; do
		cosign sign-blob --yes --bundle "${b}.sigstore.json" "$b"
	done
	info "binaries signed"
}

update_server_json() {
	local version_no_v=${VERSION#v}
	local sha
	sha=$(openssl dgst -sha256 "${BINARIES[0]}" | awk '{print $2}')
	local url="https://github.com/$EXPECTED_REPO/releases/download/$VERSION/linux-mcp"
	jq --arg v "$version_no_v" --arg sha "$sha" --arg url "$url" \
		'.version = $v
         | .packages[0].version = $v
         | .packages[0].fileSha256 = $sha
         | .packages[0].identifier = $url' \
		server.json >server.json.tmp
	mv server.json.tmp server.json
	info "updated server.json (version $version_no_v, sha ${sha:0:12})"
}

commit_and_tag() {
	git add server.json
	if git diff --cached --quiet; then
		info "server.json unchanged; skipping commit"
	else
		git commit -m "chore: update server.json for $VERSION"
	fi
	git push origin HEAD

	local tag_opts=(-a)
	local push_opts=()
	[[ $FORCE == true ]] && {
		tag_opts=(-fa)
		push_opts=(--force)
	}
	git tag "${tag_opts[@]}" "$VERSION" -m "$VERSION"
	git push origin "$VERSION" "${push_opts[@]}"
}

create_github_release() {
	if [[ $FORCE == true ]] && gh release view "$VERSION" --json id >/dev/null 2>&1; then
		warn "release $VERSION already exists on GitHub; deleting it"
		gh release delete "$VERSION" --yes
	fi
	local artifacts=("${BINARIES[@]}")
	local b
	for b in "${BINARIES[@]}"; do
		artifacts+=("${b}.sigstore.json")
	done
	gh release create "$VERSION" "${artifacts[@]}" \
		--title "$VERSION" \
		--notes-file "$CHANGELOG_FILE"
	info "release created: https://github.com/$EXPECTED_REPO/releases/tag/$VERSION"
}

publish_to_registry() {
	info "publishing to the MCP registry"
	mcp-publisher login github --token "$MCP_GITHUB_TOKEN"
	mcp-publisher publish
	info "published to the MCP registry"
}

main() {
	parse_args "$@"
	preflight
	resolve_version
	generate_changelog
	build
	sign
	update_server_json
	commit_and_tag
	create_github_release
	publish_to_registry
}

main "$@"

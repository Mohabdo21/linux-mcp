#!/usr/bin/env bash
#
# Release script for linux-mcp.
#
# Cuts a versioned release from a clean checkout of the default branch:
#   checks -> immutable tag -> build -> verify -> sign -> generated manifest ->
#   push -> GitHub release -> verify downloaded bytes -> publish to MCP registry.
#
# Usage:
#   scripts/release.sh [VERSION] | --resume-publish VERSION

set -euo pipefail

EXPECTED_REPO="Mohabdo21/linux-mcp"
BINARIES=()
VERSION=""
RESUME_PUBLISH=false
WORK_DIR=""
ARTIFACT_DIR=""
SOURCE_COMMIT=""
RELEASE_TAGS=""

cleanup() {
	if [[ -n "$WORK_DIR" ]]; then
		if [[ -f "$ARTIFACT_DIR/server.json" ]]; then
			retain_artifacts || return
		fi
		if [[ -d "$WORK_DIR/source" ]]; then
			git worktree remove --force "$WORK_DIR/source" || return
		fi
		rm -rf -- "$WORK_DIR"
	fi
}
trap cleanup EXIT

# logging

if [[ -t 1 ]]; then
	RED=$'\033[0;31m'
	GREEN=$'\033[0;32m'
	NC=$'\033[0m'
else
	RED=''
	GREEN=''
	NC=''
fi

info() { printf '%s[INFO]%s %s\n' "$GREEN" "$NC" "$*"; }
die() {
	printf '%s[ERROR]%s %s\n' "$RED" "$NC" "$*" >&2
	exit 1
}

# utilities

usage() {
	cat <<EOF
Usage: $(basename "$0") [VERSION]
       $(basename "$0") --resume-publish VERSION

Cut a versioned release of linux-mcp.

Options:
  VERSION    Release version, e.g. v1.2.3 (default: latest tag + patch bump)
  --resume-publish VERSION
             Download and verify existing release assets, then publish its manifest
  -h, --help Show this help

Run from a clean checkout of the default branch (main).
Normal releases require HEAD to equal fetched origin/main. Existing tags are
never moved. After a release upload, retry only with --resume-publish VERSION
(or the manual registry workflow). Partial uploads must be repaired explicitly;
this script never replaces release assets or deletes a release.
EOF
}

require_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "'$1' not found in PATH. Install it: $2"
}

version_ok() {
	[[ "$1" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]
}

remote_is_canonical() {
	local origin
	origin="$(git remote get-url origin)"
	case "$origin" in
	"https://github.com/$EXPECTED_REPO" | "https://github.com/$EXPECTED_REPO.git" | \
		"git@github.com:$EXPECTED_REPO" | "git@github.com:$EXPECTED_REPO.git" | \
		"ssh://git@github.com/$EXPECTED_REPO" | "ssh://git@github.com/$EXPECTED_REPO.git") return 0 ;;
	*) return 1 ;;
	esac
}

has_push_permission() {
	[[ "$(gh api "repos/$EXPECTED_REPO" --jq .permissions.push 2>/dev/null)" == true ]]
}

# steps

parse_args() {
	local version_seen=false
	while (($#)); do
		case "$1" in
		--resume-publish) RESUME_PUBLISH=true ;;
		-h | --help)
			usage
			exit 0
			;;
		--)
			shift
			if (($# > 1)); then die "too many arguments: $*"; fi
			[[ "$version_seen" == false ]] || die "too many arguments"
			VERSION="${1:-}"
			break
			;;
		-*) die "unknown option: $1 (run '$(basename "$0") --help')" ;;
		*)
			[[ "$version_seen" == false ]] || die "too many arguments: $1"
			VERSION="$1"
			version_seen=true
			;;
		esac
		shift
	done
}

preflight() {
	[[ -z "$(git rev-parse --show-prefix)" ]] || die "run from the repository root"
	local branch status
	status="$(git status --porcelain)"
	[[ -z "$status" ]] ||
		die "working tree is not clean. Commit or stash changes first."
	branch="$(git rev-parse --abbrev-ref HEAD)"
	if [[ "$RESUME_PUBLISH" == false ]]; then
		[[ "$branch" == main ]] || die "releases must be cut from 'main' (currently on '$branch')"
	fi
	remote_is_canonical || die "origin must be $EXPECTED_REPO (refusing to release to a fork/mirror)"
	require_cmd gh "https://cli.github.com/"
	require_cmd cosign "https://docs.sigstore.dev/cosign/installation/"
	require_cmd jq "https://jqlang.github.io/jq/"
	require_cmd openssl "https://www.openssl.org/"
	require_cmd go "https://go.dev/dl/"
	require_cmd timeout "GNU coreutils"
	require_cmd mcp-publisher "https://github.com/modelcontextprotocol/registry"
	gh auth status >/dev/null 2>&1 || die "gh is not authenticated. Run 'gh auth login'."
	has_push_permission || die "current gh user has no push rights on $EXPECTED_REPO"
	[[ -n "${MCP_GITHUB_TOKEN:-}" ]] ||
		die "MCP_GITHUB_TOKEN is not set. Create a PAT at https://github.com/settings/tokens/new (repo + read:user) and export it."
	[[ -f server.json ]] || die "server.json not found at the repository root"
	git fetch --tags origin refs/heads/main:refs/remotes/origin/main
	SOURCE_COMMIT="$(git rev-parse HEAD)"
	if [[ "$RESUME_PUBLISH" == false ]]; then
		[[ "$(git rev-parse refs/remotes/origin/main)" == "$SOURCE_COMMIT" ]] ||
			die "HEAD must equal fetched origin/main. Push or synchronize main before releasing."
	fi
	RELEASE_TAGS="$(gh api "repos/$EXPECTED_REPO/releases?per_page=100" --paginate --jq '.[].tag_name')"
	[[ "$(go env GOHOSTOS GOHOSTARCH GOOS GOARCH)" == $'linux\namd64\nlinux\namd64' ]] || die "release verification requires a Linux amd64 Go host and target"
	info "pre-flight checks passed"
}

resolve_version() {
	local latest
	latest="$(git tag --list | awk '/^v[0-9]+\.[0-9]+\.[0-9]+$/' | sort -V | tail -n 1)"
	latest="${latest:-v0.0.0}"

	if [[ -z "$VERSION" ]]; then
		[[ "$RESUME_PUBLISH" == false ]] || die "--resume-publish requires an explicit VERSION"
		local base="${latest#v}"
		base="${base%%-*}"
		local major minor patch
		IFS='.' read -r major minor patch <<<"$base"
		major="${major:-0}"
		minor="${minor:-0}"
		patch="${patch:-0}"
		VERSION="v$major.$minor.$((patch + 1))"
		info "no version given: releasing $VERSION (latest tag: $latest)"
		local input
		read -r -p "Press Enter to use $VERSION, or type a different version: " input || die "aborted"
		VERSION="${input:-$VERSION}"
	fi

	version_ok "$VERSION" || die "version must match 'vX.Y.Z', got: $VERSION"

	if [[ "$RESUME_PUBLISH" == true ]]; then
		grep -Fxq "$VERSION" <<<"$RELEASE_TAGS" || die "release $VERSION does not exist; nothing to resume"
		SOURCE_COMMIT="$(git rev-parse "refs/tags/$VERSION^{}")"
		return
	fi
	if grep -Fxq "$VERSION" <<<"$RELEASE_TAGS"; then
		die "release $VERSION already exists. Use --resume-publish $VERSION; never rebuild published assets."
	fi
	if git show-ref --verify --quiet "refs/tags/$VERSION"; then
		[[ "$(git rev-parse "refs/tags/$VERSION^{}")" == "$SOURCE_COMMIT" ]] ||
			die "tag $VERSION identifies another commit. Choose a new version; tags are never moved."
	fi

	if [[ "$(printf '%s\n' "$latest" "$VERSION" | sort -V | tail -n 1)" != "$VERSION" ]]; then
		die "$VERSION is older than the latest tag ($latest); choose a newer version"
	fi

	info "releasing: $VERSION"
}

build() {
	info "running tests"
	make test
	local status
	status="$(git status --porcelain)"
	[[ -z "$status" && "$(git rev-parse HEAD)" == "$SOURCE_COMMIT" ]] ||
		die "checks changed the source. Review and commit changes, push main, then retry. No source changes were committed."
	if ! git show-ref --verify --quiet "refs/tags/$VERSION"; then
		git tag -a "$VERSION" "$SOURCE_COMMIT" -m "$VERSION"
	fi
	info "building binaries"
	git worktree add --detach "$WORK_DIR/source" "refs/tags/$VERSION"
	make -C "$WORK_DIR/source" build build-static VERSION="$VERSION"
	status="$(git -C "$WORK_DIR/source" status --porcelain)"
	[[ -z "$status" ]] || die "build changed tagged source; refusing to sign"
	mkdir -p "$ARTIFACT_DIR"
	local name
	for name in linux-mcp linux-mcp_static; do
		[[ -f "$WORK_DIR/source/bin/$name" ]] || die "build artifact missing: $name"
		cp -- "$WORK_DIR/source/bin/$name" "$ARTIFACT_DIR/$name"
	done
	verify_binaries "$ARTIFACT_DIR"
}

verify_binaries() {
	local dir="$1" binary metadata response server_pid input_fd output_fd
	for binary in "$dir/linux-mcp" "$dir/linux-mcp_static"; do
		metadata="$(go version -m "$binary")"
		[[ "$(awk '$1 == "mod" && $2 == "github.com/Mohabdo21/linux-mcp" {print $3}' <<<"$metadata")" == "$VERSION" ]] ||
			die "$binary: Go module version is not $VERSION (requires Go with tagged module build metadata)"
		[[ "$(awk '$1 == "build" && /^.*vcs.revision=/ {sub(/.*vcs.revision=/, ""); print}' <<<"$metadata")" == "$SOURCE_COMMIT" ]] ||
			die "$binary: vcs.revision does not identify $SOURCE_COMMIT"
		grep -Eq '^[[:space:]]*build[[:space:]]+vcs.modified=false$' <<<"$metadata" || die "$binary: missing clean VCS provenance"
		# Keep stdin open until initialize responds; immediate EOF cancels SDK handlers.
		coproc RELEASE_SERVER { env LINUX_MCP_CONFIG='' timeout --kill-after=2s 10s "$binary"; }
		server_pid="$RELEASE_SERVER_PID" input_fd="${RELEASE_SERVER[1]}" output_fd="${RELEASE_SERVER[0]}"
		response=''
		if printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"release-check","version":"1"}}}' >&"$input_fd"; then
			IFS= read -r -t 10 response <&"$output_fd" || response=''
		fi
		exec {input_fd}>&- {output_fd}<&-
		wait "$server_pid" || die "$binary: initialize failed or timed out"
		jq -e --arg v "$VERSION" 'select(.id == 1) | .result.serverInfo.version == $v' <<<"$response" >/dev/null ||
			die "$binary: application Version does not match $VERSION"
	done
}

sign() {
	info "signing binaries with cosign (keyless)"
	info "a browser window will open for GitHub authentication"
	local b
	for b in "${BINARIES[@]}"; do
		cosign sign-blob --yes --bundle "${b}.sigstore.json" "$b"
		bash scripts/verify-signature.sh "$b"
	done
	info "binaries signed"
}

update_server_json() {
	local version_no_v="${VERSION#v}"
	local sha
	sha="$(openssl dgst -sha256 "${BINARIES[0]}" | awk '{print $2}')"
	local url="https://github.com/$EXPECTED_REPO/releases/download/$VERSION/linux-mcp"
	jq --arg v "$version_no_v" --arg sha "$sha" --arg url "$url" \
		'.version = $v
         | .packages[0].version = $v
         | .packages[0].fileSha256 = $sha
         | .packages[0].identifier = $url' \
		server.json >"$ARTIFACT_DIR/server.json"
	validate_manifest "$ARTIFACT_DIR"
	info "generated release manifest (version $version_no_v, sha ${sha:0:12})"
}

retain_artifacts() {
	mkdir -p bin
	local name
	for name in linux-mcp linux-mcp_static linux-mcp.sigstore.json linux-mcp_static.sigstore.json server.json; do
		cp -- "$ARTIFACT_DIR/$name" "bin/$name" || return
	done
}

validate_manifest() {
	local dir="$1" sha url
	sha="$(openssl dgst -sha256 "$dir/linux-mcp" | awk '{print $2}')"
	url="https://github.com/$EXPECTED_REPO/releases/download/$VERSION/linux-mcp"
	jq -e --arg v "${VERSION#v}" --arg sha "$sha" --arg url "$url" \
		'.version == $v and (.packages | length) == 1
		 and .packages[0].version == $v and .packages[0].identifier == $url
		 and .packages[0].fileSha256 == $sha' "$dir/server.json" >/dev/null ||
		die "generated manifest does not match $VERSION and the primary binary"
	mcp-publisher validate "$dir/server.json"
}

create_github_release() {
	local status
	status="$(git status --porcelain)"
	[[ -z "$status" && "$(git rev-parse HEAD)" == "$SOURCE_COMMIT" ]] || die "source changed during release; refusing to push"
	[[ "$(git rev-parse "refs/tags/$VERSION^{}")" == "$SOURCE_COMMIT" ]] || die "tag changed during release; refusing to push"
	git push --atomic origin HEAD:refs/heads/main "refs/tags/$VERSION"
	local artifacts=("${BINARIES[@]}" "$ARTIFACT_DIR/server.json")
	local b
	for b in "${BINARIES[@]}"; do
		artifacts+=("${b}.sigstore.json")
	done
	gh release create "$VERSION" "${artifacts[@]}" \
		--repo "$EXPECTED_REPO" --verify-tag \
		--title "$VERSION" \
		--generate-notes
	info "release created: https://github.com/$EXPECTED_REPO/releases/tag/$VERSION"
}

download_release() {
	local dir="$WORK_DIR/download" name
	mkdir -p "$dir"
	gh release download "$VERSION" --repo "$EXPECTED_REPO" --dir "$dir" \
		--pattern linux-mcp --pattern linux-mcp_static --pattern '*.sigstore.json' --pattern server.json
	for name in linux-mcp linux-mcp_static linux-mcp.sigstore.json linux-mcp_static.sigstore.json server.json; do
		[[ -f "$dir/$name" ]] || die "release asset missing: $name; repair the partial upload explicitly, without rebuilding published binaries"
		if [[ "$RESUME_PUBLISH" == false ]]; then
			cmp -- "$ARTIFACT_DIR/$name" "$dir/$name" || die "downloaded $name differs from the uploaded bytes"
		fi
	done
	validate_manifest "$dir"
	chmod +x "$dir/linux-mcp" "$dir/linux-mcp_static"
	if [[ "$RESUME_PUBLISH" == true ]]; then
		local binary
		for binary in "$dir/linux-mcp" "$dir/linux-mcp_static"; do
			bash scripts/verify-signature.sh "$binary"
		done
		verify_binaries "$dir"
	fi
	ARTIFACT_DIR="$dir"
	retain_artifacts
}

publish_to_registry() {
	info "publishing to the MCP registry"
	mcp-publisher login github --token "$MCP_GITHUB_TOKEN"
	mcp-publisher publish "$ARTIFACT_DIR/server.json"
	info "published to the MCP registry"
}

main() {
	parse_args "$@"
	preflight
	resolve_version
	WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/linux-mcp-release.XXXXXX")"
	ARTIFACT_DIR="$WORK_DIR/artifacts"
	BINARIES=("$ARTIFACT_DIR/linux-mcp" "$ARTIFACT_DIR/linux-mcp_static")
	if [[ "$RESUME_PUBLISH" == false ]]; then
		build
		sign
		update_server_json
		retain_artifacts
		create_github_release
	fi
	download_release
	publish_to_registry
}

main "$@"

#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"
FORCE=false

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() {
	echo -e "${RED}[ERROR]${NC} $*" >&2
	exit 1
}

preflight_checks() {
	if ! git diff --quiet HEAD; then
		error "Working tree has uncommitted changes. Commit or stash first."
	fi
	branch=$(git rev-parse --abbrev-ref HEAD)
	if [ "$branch" != "main" ]; then
		error "Releases must be cut from 'main', currently on '$branch'"
	fi
	if ! command -v gh &>/dev/null; then
		error "'gh' CLI not found. Install it: https://cli.github.com/"
	fi
	if [ -z "${MCP_GITHUB_TOKEN:-}" ]; then
		error "MCP_GITHUB_TOKEN is not set. Create a PAT at https://github.com/settings/tokens/new (repo + read:user) and set it as an env var."
	fi
	git fetch --tags origin
	info "Pre-flight checks passed"
}

push_branch() {
	git push origin "$(git rev-parse --abbrev-ref HEAD)"
	info "Branch pushed to origin"
}

resolve_version() {
	local latest_tag
	latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

	if [ -z "$VERSION" ]; then
		local base="${latest_tag#v}"
		local major minor patch
		IFS='.' read -r major minor patch <<<"$base"
		: "${patch:=0}"
		VERSION="v$major.$minor.$((patch + 1))"
		info "No version specified. Suggesting: $VERSION (latest was $latest_tag)"
		read -r -p "Press Enter to use $VERSION, or type a different version: " input
		if [ -n "$input" ]; then
			VERSION="$input"
		fi
	fi

	if ! echo "$VERSION" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
		error "Version must match 'vX.Y.Z' format, got: $VERSION"
	fi

	if git rev-parse -q --verify "refs/tags/$VERSION" &>/dev/null; then
		warn "Tag $VERSION already exists locally."
		if git ls-remote --tags origin "$VERSION" 2>/dev/null | grep -q .; then
			warn "Tag $VERSION also exists on remote."
		fi
		read -r -p "Re-release (force replace)? [y/N] " confirm
		if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
			error "Aborted. Use a different version or pass --force."
		fi
		FORCE=true
	fi

	if [ "$(echo -e "$latest_tag\n$VERSION" | sort -V | tail -1)" != "$VERSION" ]; then
		warn "$VERSION is lower than the latest tag ($latest_tag)"
		read -r -p "Continue anyway? [y/N] " confirm
		if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
			error "Aborted."
		fi
	fi

	info "Releasing version: $VERSION"
}

generate_changelog() {
	local prev_tag
	prev_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

	if [ -z "$prev_tag" ]; then
		CHANGELOG=$(git log --oneline --format="- %s (%h)" --reverse 2>/dev/null || echo "No commits found.")
		info "First release. Including all commits."
	else
		CHANGELOG=$(git log --oneline --format="- %s (%h)" "$prev_tag"..HEAD 2>/dev/null || echo "No new commits since $prev_tag.")
	fi

	if [ -z "$CHANGELOG" ] || [ "$CHANGELOG" = "No new commits since $prev_tag." ]; then
		warn "$CHANGELOG"
		read -r -p "No new commits. Continue anyway? [y/N] " confirm
		if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
			error "Aborted."
		fi
	fi

	CHANGELOG_FILE=$(mktemp)
	{
		echo "# Release $VERSION"
		echo ""
		echo "$CHANGELOG"
		echo ""
	} >"$CHANGELOG_FILE"
}

build_binaries() {
	info "Running tests..."
	make test
	info "Building binaries..."
	make build build-static VERSION="$VERSION"
}

update_server_json() {
	local version_no_v="${VERSION#v}"
	local sha url
	sha=$(openssl dgst -sha256 bin/linux-mcp | awk '{print $2}')
	url="https://github.com/Mohabdo21/linux-mcp/releases/download/$VERSION/linux-mcp"
	if [ -f server.json ]; then
		jq --arg v "$version_no_v" --arg sha "$sha" --arg url "$url" \
			'.version = $v | .packages[0].version = $v | .packages[0].fileSha256 = $sha | .packages[0].identifier = $url' \
			server.json >server.tmp && mv server.tmp server.json
		info "Updated server.json to version $version_no_v"
	fi
}

commit_server_json() {
	git add server.json
	git commit -m "chore: update server.json for $VERSION"
	info "Committed server.json update"
}

tag_and_push() {
	local tag_opts="-a"
	local push_opts=""
	if [ "$FORCE" = true ]; then
		tag_opts="-fa"
		push_opts="--force"
	fi
	git push origin HEAD
	git tag $tag_opts "$VERSION" -m "$VERSION"
	git push origin "$VERSION" $push_opts
}

create_release() {
	if [ "$FORCE" = true ]; then
		if gh release view "$VERSION" --json id &>/dev/null 2>&1; then
			warn "Release $VERSION already exists on GitHub. Deleting..."
			gh release delete "$VERSION" --yes
		fi
	fi
	gh release create "$VERSION" \
		bin/linux-mcp \
		bin/linux-mcp_static \
		--title "$VERSION" \
		--notes-file "$CHANGELOG_FILE"
	rm -f "$CHANGELOG_FILE"
	info "Release $VERSION created successfully!"
	local repo
	repo=$(git remote get-url origin 2>/dev/null | sed 's|.*github.com[:/]||; s|\.git$||')
	info "URL: https://github.com/$repo/releases/tag/$VERSION"
}

publish_to_registry() {
	if ! command -v mcp-publisher &>/dev/null; then
		error "mcp-publisher not found in PATH. Install it from https://github.com/modelcontextprotocol/registry"
	fi
	info "Publishing to MCP Registry..."
	mcp-publisher login github --token "$MCP_GITHUB_TOKEN"
	mcp-publisher publish
	info "Published to MCP Registry successfully!"
}

main() {
	preflight_checks
	push_branch
	resolve_version
	generate_changelog
	update_server_json
	commit_server_json
	tag_and_push
	build_binaries
	create_release
	publish_to_registry
}

main

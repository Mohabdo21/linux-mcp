#!/usr/bin/env bash
set -euo pipefail

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

usage() {
	cat <<EOF
Usage: $0 [OPTIONS] <binary>

Verify a linux-mcp binary using Sigstore/Cosign keyless verification.

Arguments:
  binary          Path to the binary to verify (e.g., linux-mcp or linux-mcp_static)

Options:
  -h, --help      Show this help message

Examples:
  $0 linux-mcp
  $0 --help

Download binaries and signatures from:
  https://github.com/Mohabdo21/linux-mcp/releases
EOF
}

BINARY=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	-h | --help)
		usage
		exit 0
		;;
	-*)
		error "Unknown option: $1"
		;;
	*)
		BINARY="$1"
		shift
		;;
	esac
done

if [[ -z "$BINARY" ]]; then
	error "No binary specified. Run '$0 --help' for usage."
fi

if ! command -v cosign &>/dev/null; then
	error "'cosign' not found. Install it: https://docs.sigstore.dev/cosign/installation/"
fi

if [[ ! -f "$BINARY" ]]; then
	error "Binary not found: $BINARY"
fi

BUNDLE="${BINARY}.sigstore.json"
if [[ ! -f "$BUNDLE" ]]; then
	error "Signature bundle not found: $BUNDLE

Download it from the release page:
  https://github.com/Mohabdo21/linux-mcp/releases"
fi

info "Verifying: $BINARY"
info "Bundle:    $BUNDLE"
echo ""

if cosign verify-blob \
	--bundle "$BUNDLE" \
	--certificate-identity "mohannadabdo21@hotmail.com" \
	--certificate-oidc-issuer "https://github.com/login/oauth" \
	"$BINARY" 2>&1; then
	echo ""
	info "Verification successful"
	info "Binary is authentic and signed by the linux-mcp maintainer"
else
	echo ""
	error "Verification failed

The binary may have been tampered with or the signature is invalid.
Download a fresh copy from:
  https://github.com/Mohabdo21/linux-mcp/releases"
fi

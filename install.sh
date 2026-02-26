#!/usr/bin/env bash
set -euo pipefail

# Detect GOBIN or fall back to ~/go/bin
GOBIN="${GOBIN:-${GOPATH:-$HOME/go}/bin}"

echo ""
echo "🌐 Installing DNSimple CLI"
echo ""
echo "What would you like the command to be called?"
echo ""
echo "  1) simple      (default — short & sweet)"
echo "  2) dnsimplectl (explicit)"
echo "  3) dnsimple    (middle ground)"
echo ""
printf "Choose [1/2/3]: "
read -r choice

case "${choice:-1}" in
    1|"") NAME="simple" ;;
    2)    NAME="dnsimplectl" ;;
    3)    NAME="dnsimple" ;;
    *)
        echo "Invalid choice. Using 'simple'."
        NAME="simple"
        ;;
esac

echo ""
echo "Building as '${NAME}'..."

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
go build -o "${GOBIN}/${NAME}" "${SCRIPT_DIR}"

echo ""
echo "✓ Installed to ${GOBIN}/${NAME}"
echo ""

# Check if GOBIN is in PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$GOBIN"; then
    echo "⚠ ${GOBIN} is not in your PATH."
    echo "  Add this to your shell profile:"
    echo "    export PATH=\"\$PATH:${GOBIN}\""
    echo ""
else
    echo "Run '${NAME} --help' to get started! 🎉"
fi

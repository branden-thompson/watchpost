#!/bin/sh
# Installer smoke test (make install-test): serves dist/ over a local HTTP server
# and runs scripts/install.sh against it — the same code path as the GitHub
# release download, minus GitHub. Fails on any installer error, on a checksum
# mismatch, or if the installed binary does not report the stamped version.
set -eu
cd "$(dirname "$0")/.."
command -v python3 >/dev/null 2>&1 || { echo "install-test: python3 is required for the local server" >&2; exit 1; }
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1])')
python3 -m http.server "$port" --bind 127.0.0.1 --directory dist >/dev/null 2>&1 &
srv=$!
trap 'kill $srv 2>/dev/null' EXIT INT TERM
i=0
until curl -fsS "http://127.0.0.1:$port/checksums.txt" >/dev/null 2>&1; do # wait for the server, bounded
  i=$((i+1)); [ "$i" -lt 50 ] || { echo "install-test: local server did not start" >&2; exit 1; }
  sleep 0.1
done
tmp=$(mktemp -d)
WATCHPOST_BASE_URL="http://127.0.0.1:$port" WATCHPOST_INSTALL_DIR="$tmp" sh scripts/install.sh
got=$("$tmp/watchpost" --version)
case "$got" in
  *0.0.0*) echo "install-test: installed binary is unstamped: $got" >&2; exit 1 ;;
esac
echo "install-test: OK — $got"
# Negative control: a tampered binary must be refused.
cp dist/checksums.txt "$tmp/checksums.bak"
sed 's/^[0-9a-f]\{8\}/deadbeef/' "$tmp/checksums.bak" > dist/checksums.txt
if WATCHPOST_BASE_URL="http://127.0.0.1:$port" WATCHPOST_INSTALL_DIR="$tmp/tampered" sh scripts/install.sh 2>/dev/null; then
  cp "$tmp/checksums.bak" dist/checksums.txt
  echo "install-test: a checksum mismatch must refuse to install" >&2; exit 1
fi
cp "$tmp/checksums.bak" dist/checksums.txt
echo "install-test: tamper control fired (OK)"
rm -rf "$tmp"

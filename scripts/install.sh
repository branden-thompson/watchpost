#!/bin/sh
# Watchpost installer — POSIX sh, no bash-isms; safe under `curl -fsSL <url> | sh`.
#
#   curl -fsSL https://raw.githubusercontent.com/branden-thompson/watchpost/main/scripts/install.sh | sh
#
# What it does: picks the release asset for this OS/arch, downloads it and the
# checksums file, verifies the SHA-256, installs to $WATCHPOST_INSTALL_DIR
# (default ~/.local/bin; /usr/local/bin when it is writable and ~/.local/bin is
# not on PATH), and prints the next step. Nothing runs with elevated rights.
#
# Knobs (environment):
#   WATCHPOST_VERSION      tag to install (default: latest release), e.g. v0.9.0
#   WATCHPOST_INSTALL_DIR  where the binary goes
#   WATCHPOST_BASE_URL     asset base URL (default: GitHub release downloads); the
#                          Makefile's install-test points this at a local server
#   WATCHPOST_REPO         owner/name (default branden-thompson/watchpost)
set -eu

REPO="${WATCHPOST_REPO:-branden-thompson/watchpost}"
BIN="watchpost"

say()  { printf '%s\n' "$*" >&2; }
fail() { say "install: $*"; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || fail "'$1' is required but not found — install it and rerun"; }
need curl
need uname
need mktemp

# --- platform -----------------------------------------------------------------
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*) fail "Windows: download watchpost-windows-amd64.exe from the release page instead" ;;
  *) fail "unsupported OS '$os' (linux and darwin are supported)" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) fail "unsupported architecture '$arch' (amd64 and arm64 are supported)" ;;
esac
asset="$BIN-$os-$arch"

# --- checksum tool --------------------------------------------------------------
if command -v sha256sum >/dev/null 2>&1; then
  sha() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
  sha() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  fail "neither sha256sum nor shasum found — cannot verify the download"
fi

# --- where the assets live ----------------------------------------------------------
version="${WATCHPOST_VERSION:-}"
case "$version" in ""|v*) ;; *) version="v$version" ;; esac # tags are v*: "0.9.0" means v0.9.0
if [ -n "${WATCHPOST_BASE_URL:-}" ]; then
  base="${WATCHPOST_BASE_URL%/}"
else
  if [ -z "$version" ]; then
    base="https://github.com/$REPO/releases/latest/download"
  else
    base="https://github.com/$REPO/releases/download/$version"
  fi
fi
# HTTPS only, TLS 1.2+, redirects included (GitHub's latest/download is a 302);
# a plain-http base is the Makefile's local install-test server.
case "$base" in
  https://*) fetch() { curl --proto '=https' --tlsv1.2 -fsSL "$base/$1" -o "$2"; } ;;
  *)         fetch() { curl -fsSL "$base/$1" -o "$2"; } ;;
esac

# --- download + verify ----------------------------------------------------------------
tmp=$(mktemp -d 2>/dev/null || mktemp -d -t watchpost)
trap 'rm -rf "$tmp"' EXIT INT TERM
say "watchpost: downloading $asset (${version:-latest})…"
fetch "$asset" "$tmp/$asset" || fail "download failed — check the release exists for $os/$arch (${version:-latest})"
fetch "checksums.txt" "$tmp/checksums.txt" || fail "checksums.txt is missing from the release — refusing to install an unverified binary"
want=$(grep " $asset\$" "$tmp/checksums.txt" | cut -d' ' -f1)
[ -n "$want" ] || fail "checksums.txt has no entry for $asset"
got=$(sha "$tmp/$asset")
[ "$got" = "$want" ] || fail "SHA-256 mismatch for $asset (got $got, want $want) — the download is corrupt or tampered with; nothing was installed"

# --- install ------------------------------------------------------------------------------
dir="${WATCHPOST_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
  [ -n "${HOME:-}" ] || fail "HOME is not set — set WATCHPOST_INSTALL_DIR to a writable directory"
  dir="$HOME/.local/bin"
  case ":${PATH:-}:" in
    *":$dir:"*) ;;
    *) [ -w /usr/local/bin ] && dir=/usr/local/bin ;;
  esac
fi
mkdir -p "$dir" || fail "cannot create $dir — set WATCHPOST_INSTALL_DIR to a writable directory"
chmod 0755 "$tmp/$asset"
mv "$tmp/$asset" "$dir/$BIN" || fail "cannot write $dir/$BIN — set WATCHPOST_INSTALL_DIR to a writable directory"
if ver=$("$dir/$BIN" --version 2>/dev/null); then
  say "watchpost: installed $ver to $dir/$BIN"
else
  say "watchpost: installed to $dir/$BIN, but it did not start — Linux builds need glibc (not Alpine/musl); run it once to see the loader's message"
fi
case ":${PATH:-}:" in
  *":$dir:"*) ;;
  *) say "watchpost: $dir is not on your PATH — add it, e.g.:  export PATH=\"$dir:\$PATH\"  (or log out and back in: Debian/Ubuntu add ~/.local/bin automatically)" ;;
esac
say "watchpost: next, run:  $BIN   (Setup opens on the first run)"

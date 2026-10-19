#!/usr/bin/env sh
set -eu

REPO="SinghAman21/openport"
VERSION="${OPENPORT_VERSION:-latest}"
INSTALL_DIR="${OPENPORT_INSTALL_DIR:-$HOME/.local/bin}"

fail() {
  printf 'openport install failed: %s\n' "$1" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v mktemp >/dev/null 2>&1 || fail "mktemp is required"

OS="$(uname -s)"
ARCH="$(uname -m)"

# Git Bash, MSYS2 and Cygwin can execute this bootstrap on Windows. Native
# PowerShell users should run install.ps1 directly because PowerShell cannot
# interpret a POSIX shell script.
case "$OS" in
  MINGW*|MSYS*|CYGWIN*)
    command -v powershell.exe >/dev/null 2>&1 || fail "PowerShell was not found"
    PS1_URL="${OPENPORT_INSTALL_PS1_URL:-https://openport.singhaman.me/install.ps1}"
    PS1_FILE="$(mktemp)"
    trap 'rm -f "$PS1_FILE"' EXIT
    printf 'Windows shell detected; handing off to PowerShell …\n'
    curl --fail --location --silent --show-error --proto '=https' --tlsv1.2 "$PS1_URL" -o "$PS1_FILE" || fail "could not download $PS1_URL"
    powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(cygpath -w "$PS1_FILE" 2>/dev/null || printf '%s' "$PS1_FILE")"
    exit $?
    ;;
  Linux) ;;
  *) fail "unsupported OS: $OS (supported: Linux, Git Bash/MSYS/Cygwin on Windows)" ;;
esac

case "$ARCH" in
  x86_64|amd64) ASSET="openport-linux-amd64" ;;
  aarch64|arm64) ASSET="openport-linux-arm64" ;;
  *) fail "unsupported Linux architecture: $ARCH (supported: x86_64, arm64)" ;;
esac

if [ "$VERSION" = "latest" ]; then
  URL="https://github.com/$REPO/releases/latest/download/$ASSET"
else
  case "$VERSION" in
    v*) TAG="$VERSION" ;;
    *) TAG="v$VERSION" ;;
  esac
  URL="https://github.com/$REPO/releases/download/$TAG/$ASSET"
fi

TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t openport)"
trap 'rm -rf "$TMP_DIR"' EXIT
TMP_BINARY="$TMP_DIR/openport"

printf 'Downloading OpenPort (%s) …\n' "$ASSET"
curl --fail --location --silent --show-error --proto '=https' --tlsv1.2 "$URL" -o "$TMP_BINARY" || fail "could not download $URL"

[ -s "$TMP_BINARY" ] || fail "downloaded file is empty"
chmod 755 "$TMP_BINARY"

mkdir -p "$INSTALL_DIR"
if ! mv "$TMP_BINARY" "$INSTALL_DIR/openport" 2>/dev/null; then
  if command -v install >/dev/null 2>&1; then
    install -m 755 "$TMP_BINARY" "$INSTALL_DIR/openport"
  else
    cp "$TMP_BINARY" "$INSTALL_DIR/openport"
    chmod 755 "$INSTALL_DIR/openport"
  fi
fi

printf '\nOpenPort installed to %s/openport\n' "$INSTALL_DIR"

case ":${PATH}:" in
  *":$INSTALL_DIR:"*)
    PATH_READY=1
    ;;
  *)
    PATH_READY=0
    ;;
esac

if [ "$PATH_READY" -eq 1 ]; then
  if "$INSTALL_DIR/openport" version >/dev/null 2>&1; then
    printf 'Verified: '
    "$INSTALL_DIR/openport" version || true
  else
    printf 'Installed successfully.\n'
  fi
else
  printf '\nAdd OpenPort to your current shell:\n'
  printf '  export PATH="%s:$PATH"\n' "$INSTALL_DIR"
  printf '\nThen run:\n  openport\n'
  printf '\nTo make this permanent, add that export to ~/.profile, ~/.bashrc, or ~/.zshrc.\n'
fi

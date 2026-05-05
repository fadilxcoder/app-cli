#!/usr/bin/env sh
# myapp installer
#
#   curl -fsSL https://raw.githubusercontent.com/<owner>/<repo>/main/install.sh | sh
#
# Environment overrides:
#   MYAPP_REPO     GitHub "owner/repo" (default: fadilxcoder/app-cli)
#   MYAPP_VERSION  release tag (default: latest)
#   MYAPP_BIN_DIR  install directory (default: /usr/local/bin)

set -eu

REPO="${MYAPP_REPO:-fadilxcoder/app-cli}"
VERSION="${MYAPP_VERSION:-latest}"
BIN_DIR="${MYAPP_BIN_DIR:-/usr/local/bin}"
BIN_NAME="myapp"

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!! \033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31mxx \033[0m %s\n' "$*" >&2; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || die "missing required tool: $1"; }
need uname
need install
if command -v curl >/dev/null 2>&1; then
    DL="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
    DL="wget -qO-"
else
    die "need curl or wget"
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) die "unsupported arch: $ARCH_RAW" ;;
esac
case "$OS" in
    linux|darwin) ;;
    *) die "unsupported OS: $OS" ;;
esac

ASSET="${BIN_NAME}-${OS}-${ARCH}"

if [ "$VERSION" = "latest" ]; then
    URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
else
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"
fi

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT
TMP_BIN="${TMPDIR}/${BIN_NAME}"

log "Downloading ${ASSET} from ${URL}"
if ! $DL "$URL" > "$TMP_BIN"; then
    die "download failed — verify MYAPP_REPO=${REPO} and that release ${VERSION} exists"
fi

# Reject HTML error pages masquerading as binaries.
if head -c 4 "$TMP_BIN" | grep -q '<'; then
    die "downloaded file looks like HTML, not a binary — bad URL?"
fi

chmod +x "$TMP_BIN"

if [ -w "$BIN_DIR" ] || [ "$(id -u)" -eq 0 ]; then
    install -m 0755 "$TMP_BIN" "${BIN_DIR}/${BIN_NAME}"
else
    log "Elevating with sudo to install into ${BIN_DIR}"
    sudo install -m 0755 "$TMP_BIN" "${BIN_DIR}/${BIN_NAME}"
fi

log "Installed: ${BIN_DIR}/${BIN_NAME}"
"${BIN_DIR}/${BIN_NAME}" --version || true

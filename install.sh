#!/usr/bin/env bash
set -e

REPO="bssn1337/dorkscan"
BINARY="dorkscan"
INSTALL_DIR="/usr/local/bin"

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; NC='\033[0m'

info()    { echo -e "  ${CYAN}▸${NC} $*"; }
success() { echo -e "  ${GREEN}✓${NC} $*"; }
warn()    { echo -e "  ${YELLOW}⚠${NC} $*"; }
error()   { echo -e "  ${RED}✗${NC} $*"; exit 1; }

# ── Banner ────────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}  ┌─────────────────────────────────────────┐${NC}"
echo -e "${BOLD}  │          DORKSCAN  INSTALLER            │${NC}"
echo -e "${BOLD}  │     Gatlab Security Research Tool       │${NC}"
echo -e "${BOLD}  └─────────────────────────────────────────┘${NC}"
echo ""

# ── Check: already installed? ─────────────────────────────────────────────────
if command -v dorkscan &>/dev/null; then
  CURRENT=$(dorkscan --version 2>/dev/null || echo "unknown")
  success "dorkscan sudah terinstall${DIM} (${CURRENT})${NC}"

  # Check for newer version
  info "Mengecek update..."
  LATEST=$(curl -skL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')

  if [ -n "$LATEST" ] && [ "$LATEST" != "$CURRENT" ]; then
    warn "Versi baru tersedia: ${LATEST} (terpasang: ${CURRENT})"
    echo -e "  ${DIM}Jalankan ulang install.sh untuk upgrade.${NC}"
    echo ""
    _show_usage
    exit 0
  else
    success "Sudah versi terbaru"
    echo ""
    _show_usage
    exit 0
  fi
fi

# ── Detect OS & Arch ──────────────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux) ;;
  *) error "OS tidak didukung: $OS (hanya Linux)" ;;
esac

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  armv7l)  error "ARM 32-bit belum didukung, gunakan ARM64" ;;
  *)       error "Arsitektur tidak didukung: $ARCH" ;;
esac

ASSET_NAME="dorkscan-linux-${ARCH}"
info "Platform: Linux/${ARCH}"

# ── Check dependencies ────────────────────────────────────────────────────────
info "Mengecek dependencies..."

MISSING=()
for dep in curl; do
  if ! command -v "$dep" &>/dev/null; then
    MISSING+=("$dep")
  fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
  warn "Dependency kurang: ${MISSING[*]}"
  info "Menginstall..."
  if command -v apt-get &>/dev/null; then
    apt-get update -qq && apt-get install -y -qq "${MISSING[@]}"
  elif command -v yum &>/dev/null; then
    yum install -y -q "${MISSING[@]}"
  elif command -v apk &>/dev/null; then
    apk add --quiet "${MISSING[@]}"
  else
    error "Package manager tidak ditemukan. Install manual: ${MISSING[*]}"
  fi
fi

success "Semua dependency tersedia"

# ── Fetch latest release ──────────────────────────────────────────────────────
info "Mengambil info versi terbaru dari GitHub..."

API_URL="https://api.github.com/repos/${REPO}/releases/latest"
RELEASE_JSON=$(curl -skL "$API_URL")

LATEST_TAG=$(echo "$RELEASE_JSON" | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/' | head -1)

if [ -z "$LATEST_TAG" ]; then
  error "Gagal mengambil info release dari GitHub. Cek koneksi internet."
fi

DOWNLOAD_URL=$(echo "$RELEASE_JSON" \
  | grep '"browser_download_url"' \
  | grep "${ASSET_NAME}" \
  | sed 's/.*"browser_download_url": *"\(.*\)".*/\1/' \
  | head -1)

if [ -z "$DOWNLOAD_URL" ]; then
  error "Binary '${ASSET_NAME}' tidak ditemukan di release ${LATEST_TAG}"
fi

info "Versi: ${LATEST_TAG}"

# ── Download ──────────────────────────────────────────────────────────────────
TMP=$(mktemp)
info "Mengunduh ${ASSET_NAME}..."

if ! curl -skL --progress-bar "$DOWNLOAD_URL" -o "$TMP"; then
  rm -f "$TMP"
  error "Download gagal"
fi

# ── Install ───────────────────────────────────────────────────────────────────
chmod +x "$TMP"

if [ ! -w "$INSTALL_DIR" ]; then
  if command -v sudo &>/dev/null; then
    sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"
  else
    error "Tidak ada akses tulis ke ${INSTALL_DIR} dan sudo tidak tersedia. Jalankan sebagai root."
  fi
else
  mv "$TMP" "${INSTALL_DIR}/${BINARY}"
fi

success "Terinstall di ${INSTALL_DIR}/${BINARY}"

# ── Verify ────────────────────────────────────────────────────────────────────
if ! command -v dorkscan &>/dev/null; then
  warn "${INSTALL_DIR} tidak ada di PATH. Tambahkan ke ~/.bashrc:"
  echo -e "       ${DIM}export PATH=\$PATH:${INSTALL_DIR}${NC}"
else
  success "Instalasi berhasil!"
fi

echo ""

# ── Show usage ────────────────────────────────────────────────────────────────
_show_usage() {
  echo -e "${BOLD}  ── Quick Start ──────────────────────────────────────${NC}"
  echo ""
  echo -e "  ${DIM}# Buat file API key (satu per baris)${NC}"
  echo -e "  echo 'API_KEY_SERPER' > keys.txt"
  echo ""
  echo -e "  ${DIM}# Scan domain .go.id dan .sch.id${NC}"
  echo -e "  ${CYAN}dorkscan scan${NC} -t .go.id,.ac.id,.sch.id -k \"slot,judi,togel\" --keys keys.txt -e"
  echo ""
  echo -e "  ${DIM}# Export hasil ke CSV${NC}"
  echo -e "  ${CYAN}dorkscan export${NC} --db results.db --format csv --out hasil.csv"
  echo ""
  echo -e "  ${DIM}# Lihat statistik${NC}"
  echo -e "  ${CYAN}dorkscan stats${NC} --db results.db"
  echo ""
  echo -e "  ${DIM}# Help lengkap${NC}"
  echo -e "  ${CYAN}dorkscan scan --help${NC}"
  echo ""
  echo -e "${BOLD}  ── Dork templates: dorks.txt${NC}"
  echo -e "${BOLD}  ── Repo: https://github.com/${REPO}${NC}"
  echo ""
}

_show_usage

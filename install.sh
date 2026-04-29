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

# ── Usage (definisi di atas agar bisa dipanggil kapan saja) ───────────────────
show_usage() {
  echo ""
  echo -e "${BOLD}  ── Quick Start ──────────────────────────────────────${NC}"
  echo ""
  echo -e "  ${DIM}# Buat file API key (satu per baris)${NC}"
  echo -e "  echo 'API_KEY_SERPER' > keys.txt"
  echo ""
  echo -e "  ${DIM}# Scan domain Indonesia${NC}"
  echo -e "  ${CYAN}dorkscan scan${NC} -t .go.id,.ac.id,.sch.id -k \"slot,judi,togel\" --keys keys.txt -e"
  echo ""
  echo -e "  ${DIM}# Export hasil ke CSV${NC}"
  echo -e "  ${CYAN}dorkscan export${NC} --format csv -o hasil.csv"
  echo ""
  echo -e "  ${DIM}# Lihat statistik${NC}"
  echo -e "  ${CYAN}dorkscan stats${NC}"
  echo ""
  echo -e "  ${DIM}# Help lengkap${NC}"
  echo -e "  ${CYAN}dorkscan scan --help${NC}"
  echo ""
  echo -e "${BOLD}  ── Repo: https://github.com/${REPO}${NC}"
  echo ""
}

# ── Banner ────────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}  ┌─────────────────────────────────────────┐${NC}"
echo -e "${BOLD}  │          DORKSCAN  INSTALLER            │${NC}"
echo -e "${BOLD}  │     Gatlab Security Research Tool       │${NC}"
echo -e "${BOLD}  └─────────────────────────────────────────┘${NC}"
echo ""

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
  *) error "Arsitektur tidak didukung: $ARCH" ;;
esac

ASSET_NAME="dorkscan-linux-${ARCH}"
info "Platform: Linux/${ARCH}"

# ── Fetch latest release ──────────────────────────────────────────────────────
info "Mengambil info versi terbaru dari GitHub..."

RELEASE_JSON=$(curl -skL "https://api.github.com/repos/${REPO}/releases/latest")
LATEST=$(echo "$RELEASE_JSON" | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/' | head -1)

if [ -z "$LATEST" ]; then
  error "Gagal mengambil info release dari GitHub. Cek koneksi internet."
fi

# ── Check: already installed & up to date? ───────────────────────────────────
if command -v dorkscan &>/dev/null; then
  CURRENT=$(dorkscan --version 2>/dev/null | awk '{print $NF}' || echo "")
  if [ "$CURRENT" = "$LATEST" ]; then
    success "dorkscan ${LATEST} sudah terinstall dan merupakan versi terbaru"
    show_usage
    exit 0
  fi
  warn "Update tersedia: ${LATEST} (terpasang: ${CURRENT:-unknown}) — mengupgrade..."
else
  info "Versi: ${LATEST}"
fi

# ── Check & install dependencies ─────────────────────────────────────────────
info "Mengecek dependencies..."
MISSING=()
for dep in curl; do
  command -v "$dep" &>/dev/null || MISSING+=("$dep")
done

if [ ${#MISSING[@]} -gt 0 ]; then
  warn "Dependency kurang: ${MISSING[*]} — menginstall..."
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

# ── Download ──────────────────────────────────────────────────────────────────
DOWNLOAD_URL=$(echo "$RELEASE_JSON" \
  | grep '"browser_download_url"' \
  | grep "${ASSET_NAME}" \
  | sed 's/.*"browser_download_url": *"\(.*\)".*/\1/' \
  | head -1)

if [ -z "$DOWNLOAD_URL" ]; then
  error "Binary '${ASSET_NAME}' tidak ditemukan di release ${LATEST}"
fi

TMP=$(mktemp)
info "Mengunduh ${ASSET_NAME} ${LATEST}..."

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

success "Terinstall di ${INSTALL_DIR}/${BINARY} (${LATEST})"

# ── Verify ────────────────────────────────────────────────────────────────────
if command -v dorkscan &>/dev/null; then
  success "Instalasi berhasil!"
else
  warn "${INSTALL_DIR} tidak ada di PATH. Tambahkan: export PATH=\$PATH:${INSTALL_DIR}"
fi

show_usage

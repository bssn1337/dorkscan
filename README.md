# Dorkscan

Autonomous domain harvester via Google Search API (Serper.dev). Dikembangkan untuk keperluan riset keamanan siber — mengumpulkan domain yang terindikasi terinfeksi konten ilegal berdasarkan dork query yang dapat dikonfigurasi sepenuhnya.

> Dikembangkan oleh **Gatlab Security Research** — [gatlab.id](https://gatlab.id)

---

## Install

```bash
curl -skL https://raw.githubusercontent.com/bssn1337/dorkscan/master/install.sh | bash
```

Script otomatis:
- Mendeteksi OS dan arsitektur (amd64 / arm64)
- Mengecek dan menginstall dependency yang kurang
- Mendownload binary terbaru dari GitHub Releases
- Menginstall ke `/usr/local/bin/dorkscan`

Jika `dorkscan` sudah terinstall, script akan mengecek versi terbaru dan menampilkan panduan penggunaan.

---

## Quick Start

```bash
# 1. Buat file API key (daftar gratis di serper.dev)
echo "API_KEY_ANDA" > keys.txt

# 2. Scan
dorkscan scan -t .go.id,.ac.id,.sch.id -k "slot,judi,togel" --keys keys.txt -e

# 3. Export hasil
dorkscan export --format csv -o hasil.csv

# 4. Lihat statistik
dorkscan stats
```

---

## Penggunaan

```
dorkscan [command] [flags]
```

### `scan` — Jalankan dork scan

```bash
dorkscan scan -t .go.id -k "slot,judi" --keys keys.txt
```

| Flag | Short | Default | Keterangan |
|------|-------|---------|------------|
| `--tld` | `-t` | — | Target TLD, pisahkan koma (`.go.id,.ac.id,.sch.id`) **[wajib]** |
| `--keyword` | `-k` | — | Kata kunci pencarian, pisahkan koma (`slot,judi,togel`) **[wajib]** |
| `--keys` | — | — | File berisi Serper API key, satu per baris |
| `--key` | — | — | Serper API key tunggal |
| `--out` | `-o` | `results.db` | Output SQLite database |
| `--enrich` | `-e` | `false` | Aktifkan enrichment: resolve IP, ISP/ASN, CMS detection |
| `--depth` | `-d` | `3` | Jumlah halaman per query (1 halaman = 10 hasil, maks 10) |
| `--limit` | `-l` | `0` | Batas maksimal domain (0 = unlimited) |
| `--delay` | — | `600` | Jeda antar API request dalam ms |
| `--concurrency` | — | `20` | Jumlah worker enrichment paralel |
| `--dork-file` | — | — | File template dork kustom |
| `--verbose` | `-v` | `false` | Tampilkan detail tiap domain yang ditemukan |

### `export` — Export data hasil scan

```bash
dorkscan export --format csv -o hasil.csv
dorkscan export --format json -o hasil.json
dorkscan export --format txt -o domains.txt
```

| Flag | Default | Keterangan |
|------|---------|------------|
| `--db` | `results.db` | SQLite database sumber |
| `--format` | `csv` | Format export: `csv`, `json`, `txt` |
| `--out` / `-o` | stdout | File output |

### `stats` — Statistik database

```bash
dorkscan stats
dorkscan stats --db hasil-scan.db
```

---

## Dork Templates

Secara default dorkscan menggunakan 3 template:

```
site:{tld} "{keyword}"
site:{tld} intitle:"{keyword}"
site:{tld} inurl:"{keyword}"
```

Untuk template kustom, buat file `dorks.txt`:

```
site:{tld} "{keyword}" intext:"daftar sekarang"
site:{tld} "{keyword}" intext:"link alternatif"
site:{tld} "{keyword}" filetype:php
```

Lalu jalankan dengan:

```bash
dorkscan scan -t .go.id -k "slot" --keys keys.txt --dork-file dorks.txt
```

---

## Format API Key

Buat file `keys.txt` — satu key per baris, baris diawali `#` diabaikan:

```
# Serper.dev API keys
# Daftar gratis di https://serper.dev (2500 query/bulan per akun)
abc123_api_key_1
def456_api_key_2
ghi789_api_key_3
```

Dengan 10 key (free tier) → **25.000 query/bulan** → potensi ratusan ribu domain.

---

## Contoh Output

**Terminal saat scan berjalan:**
```
  ▸ TLD target   : .go.id, .ac.id, .sch.id
  ▸ Keywords     : slot, judi, togel
  ▸ Dorks        : 9 kombinasi
  ▸ API keys     : 5 key
  ▸ Depth        : 3 halaman/query
  ▸ Enrichment   : true

  [1/9] site:.go.id "slot"
  ▸ halaman 1 — 10 result, 8 baru
  ▸ halaman 2 — 10 result, 7 baru
  ...

  ► Terkumpul: 247 domain
```

**Laporan akhir:**
```
══════════════════════════════════════════════
  DORKSCAN — Laporan Scan
══════════════════════════════════════════════
  Durasi        : 8m 14s
  Total Domain  : 247

  TLD Breakdown
  ─────────────────────────────────────
  .go.id                    124
  .sch.id                   83
  .ac.id                    40

  CMS Distribution
  ─────────────────────────────────────
  WordPress                 198
  Joomla                    27
  Unknown                   22

  Top ISP / Hosting Provider
  ─────────────────────────────────────
  PT Telkom Indonesia             89
  Biznet                          52
  IDCloudHost                     38
══════════════════════════════════════════════
```

---

## Build dari Source

```bash
git clone https://github.com/bssn1337/dorkscan.git
cd dorkscan
go mod tidy

# Build untuk Linux amd64
make linux

# Build untuk Linux arm64
make linux-arm

# Build lokal
make build
```

Membutuhkan Go 1.22+. Binary output di folder `dist/`.

---

## Deploy ke VPS

```bash
# Install sekali via curl
curl -skL https://raw.githubusercontent.com/bssn1337/dorkscan/master/install.sh | bash

# Setelah install, langsung jalankan
dorkscan scan -t .go.id,.sch.id -k "slot,judi,togel" --keys keys.txt -e -d 5
```

Binary adalah static binary (CGO_ENABLED=0) — tidak membutuhkan runtime apapun, berjalan di VPS manapun.

---

## Disclaimer

Tool ini dikembangkan untuk keperluan **riset keamanan siber dan monitoring domain** yang terindikasi terinfeksi konten ilegal. Penggunaan harus sesuai dengan hukum yang berlaku. Gatlab tidak bertanggung jawab atas penyalahgunaan tool ini.

---

<div align="center">
  <sub>Gatlab Security Research · <a href="https://gatlab.id">gatlab.id</a></sub>
</div>

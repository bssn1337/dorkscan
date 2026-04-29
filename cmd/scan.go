package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"

	"github.com/bssn1337/dorkscan/internal/dork"
	"github.com/bssn1337/dorkscan/internal/enrich"
	"github.com/bssn1337/dorkscan/internal/reporter"
	"github.com/bssn1337/dorkscan/internal/serper"
	"github.com/bssn1337/dorkscan/internal/storage"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Jalankan dork scan untuk mengumpulkan domain",
	Example: `  dorkscan scan -t .go.id -k "slot,judi" --keys keys.txt
  dorkscan scan -t .ac.id,.sch.id -k "togel,casino" --keys keys.txt -e -d 5
  dorkscan scan -t .go.id -k "judi" --key YOUR_API_KEY -o hasil.db`,
	RunE: runScan,
}

var (
	flagTLD         string
	flagKeyword     string
	flagKeysFile    string
	flagKey         string
	flagOut         string
	flagEnrich      bool
	flagDepth       int
	flagLimit       int
	flagDelay       int
	flagConcurrency int
	flagDorkFile    string
	flagVerbose     bool
)

func init() {
	scanCmd.Flags().StringVarP(&flagTLD, "tld", "t", "", "Target TLD — bisa lebih dari satu (contoh: .go.id,.ac.id,.sch.id) [wajib]")
	scanCmd.Flags().StringVarP(&flagKeyword, "keyword", "k", "", "Kata kunci pencarian, pisahkan koma (contoh: \"slot,judi,togel\") [wajib]")
	scanCmd.Flags().StringVar(&flagKeysFile, "keys", "", "Path ke file berisi Serper API key (satu per baris)")
	scanCmd.Flags().StringVar(&flagKey, "key", "", "Serper API key tunggal")
	scanCmd.Flags().StringVarP(&flagOut, "out", "o", "results.db", "Output database SQLite")
	scanCmd.Flags().BoolVarP(&flagEnrich, "enrich", "e", false, "Aktifkan enrichment: IP, ISP, CMS detection")
	scanCmd.Flags().IntVarP(&flagDepth, "depth", "d", 3, "Jumlah halaman per query (1 halaman = 10 hasil)")
	scanCmd.Flags().IntVarP(&flagLimit, "limit", "l", 0, "Batas maksimal domain yang dikumpulkan (0 = unlimited)")
	scanCmd.Flags().IntVar(&flagDelay, "delay", 600, "Jeda antar API request dalam ms")
	scanCmd.Flags().IntVar(&flagConcurrency, "concurrency", 20, "Jumlah worker enrichment paralel")
	scanCmd.Flags().StringVar(&flagDorkFile, "dork-file", "", "File template dork kustom (satu template per baris)")
	scanCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Tampilkan detail tiap domain yang ditemukan")

	scanCmd.MarkFlagsMutuallyExclusive("key", "keys")
}

func runScan(cmd *cobra.Command, args []string) error {
	if flagTLD == "" || flagKeyword == "" {
		return fmt.Errorf("--tld dan --keyword wajib diisi\n\nContoh:\n  dorkscan scan --tld .go.id --keyword \"slot,judi\" --keys keys.txt")
	}
	if flagKey == "" && flagKeysFile == "" {
		return fmt.Errorf("gunakan --key atau --keys untuk menyediakan Serper API key")
	}

	// Load API keys
	var keys []string
	if flagKey != "" {
		keys = []string{flagKey}
	} else {
		k, err := loadLines(flagKeysFile)
		if err != nil {
			return fmt.Errorf("gagal load keys: %w", err)
		}
		if len(k) == 0 {
			return fmt.Errorf("file keys kosong: %s", flagKeysFile)
		}
		keys = k
	}

	tlds := splitTrim(flagTLD)
	keywords := splitTrim(flagKeyword)

	var templates []string
	if flagDorkFile != "" {
		t, err := loadLines(flagDorkFile)
		if err != nil {
			return fmt.Errorf("gagal load dork file: %w", err)
		}
		templates = t
	}

	dorks := dork.Generate(tlds, keywords, templates)

	// Banner
	fmt.Println()
	fmt.Printf("  ▸ TLD target   : %s\n", strings.Join(tlds, ", "))
	fmt.Printf("  ▸ Keywords     : %s\n", strings.Join(keywords, ", "))
	fmt.Printf("  ▸ Dorks        : %d kombinasi\n", len(dorks))
	fmt.Printf("  ▸ API keys     : %d key\n", len(keys))
	fmt.Printf("  ▸ Depth        : %d halaman/query\n", flagDepth)
	fmt.Printf("  ▸ Enrichment   : %v\n", flagEnrich)
	fmt.Printf("  ▸ Output       : %s\n\n", flagOut)

	// Init DB
	db, err := storage.Open(flagOut)
	if err != nil {
		return fmt.Errorf("gagal buka database: %w", err)
	}
	defer db.Close()

	client := serper.New(keys)
	scanID := fmt.Sprintf("scan_%s", time.Now().Format("20060102_150405"))
	start := time.Now()

	var collected int64
	var skipped int64
	var queryCount int64

	// Pipeline: scan → enrich → save
	jobs := make(chan *storage.Domain, 200)
	var wg sync.WaitGroup

	// Start enricher workers (or plain saver if no enrichment)
	workers := 1
	if flagEnrich {
		workers = flagConcurrency
	}
	enricher := enrich.New(flagConcurrency)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range jobs {
				if flagEnrich {
					enricher.Enrich(d)
				}
				if err := db.Insert(d); err == nil {
					n := atomic.AddInt64(&collected, 1)
					if flagVerbose {
						cms := d.CMS
						if cms == "" {
							cms = "?"
						}
						isp := d.ISP
						if isp == "" {
							isp = "?"
						}
						fmt.Printf("  [+] %-45s  %-12s  %s\n", d.Domain, cms, isp)
					} else {
						fmt.Printf("\r  ► Terkumpul: %d domain", n)
					}
				}
			}
		}()
	}

	// Scan loop
	for i, d := range dorks {
		if flagLimit > 0 && atomic.LoadInt64(&collected) >= int64(flagLimit) {
			break
		}

		fmt.Printf("\n\n  [%d/%d] %s\n", i+1, len(dorks), d.Query)

		for page := 0; page < flagDepth; page++ {
			if flagLimit > 0 && atomic.LoadInt64(&collected) >= int64(flagLimit) {
				break
			}

			atomic.AddInt64(&queryCount, 1)
			results, err := client.Search(d.Query, page)
			if err != nil {
				fmt.Printf("  ✗ Error (page %d): %v\n", page+1, err)
				break
			}

			pageNew := 0
			for _, r := range results.Organic {
				domain := extractDomain(r.Link)
				if domain == "" {
					continue
				}
				if db.Exists(domain) {
					atomic.AddInt64(&skipped, 1)
					continue
				}
				entry := &storage.Domain{
					Domain:     domain,
					URL:        r.Link,
					Title:      r.Title,
					Snippet:    r.Snippet,
					KeywordHit: d.Keyword,
					DorkUsed:   d.Query,
					TLD:        d.TLD,
					ScanID:     scanID,
					FirstSeen:  time.Now(),
				}
				jobs <- entry
				pageNew++
			}

			fmt.Printf("  ▸ halaman %d — %d result, %d baru\n", page+1, len(results.Organic), pageNew)

			if len(results.Organic) < 10 {
				break
			}

			time.Sleep(time.Duration(flagDelay) * time.Millisecond)
		}
	}

	close(jobs)
	wg.Wait()

	fmt.Printf("\n\n  ✓ Scan selesai — %d domain baru, %d di-skip (sudah ada)\n", atomic.LoadInt64(&collected), atomic.LoadInt64(&skipped))

	stats := db.Stats()
	reporter.Print(stats, time.Since(start), flagOut)

	return nil
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host
}

func loadLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines, sc.Err()
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

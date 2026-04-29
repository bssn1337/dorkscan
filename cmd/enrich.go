package cmd

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"

	"github.com/bssn1337/dorkscan/internal/enrich"
	"github.com/bssn1337/dorkscan/internal/reporter"
	"github.com/bssn1337/dorkscan/internal/storage"
)

var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Isi data CMS, ISP, SSL untuk domain yang belum ter-enrich",
	Example: `  dorkscan enrich --db results.db
  dorkscan enrich --db scan-indo-20260429.db -c 30 -v`,
	RunE: runEnrich,
}

var (
	flagEnrichDB          string
	flagEnrichConcurrency int
	flagEnrichVerbose     bool
	flagEnrichLimit       int
)

func init() {
	enrichCmd.Flags().StringVar(&flagEnrichDB, "db", "results.db", "Database SQLite yang akan di-enrich")
	enrichCmd.Flags().IntVarP(&flagEnrichConcurrency, "concurrency", "c", 30, "Jumlah worker paralel")
	enrichCmd.Flags().BoolVarP(&flagEnrichVerbose, "verbose", "v", false, "Tampilkan detail tiap domain")
	enrichCmd.Flags().IntVarP(&flagEnrichLimit, "limit", "l", 0, "Batas domain yang di-enrich (0=semua)")
}

func runEnrich(cmd *cobra.Command, args []string) error {
	db, err := storage.Open(flagEnrichDB)
	if err != nil {
		return fmt.Errorf("gagal buka database: %w", err)
	}
	defer db.Close()

	domains, err := db.GetUnenriched(flagEnrichLimit)
	if err != nil {
		return fmt.Errorf("gagal baca domain: %w", err)
	}

	if len(domains) == 0 {
		fmt.Println("\n  ✓ Semua domain sudah ter-enrich")
		return nil
	}

	fmt.Printf("\n  ▸ Database      : %s\n", flagEnrichDB)
	fmt.Printf("  ▸ Domain pending: %d\n", len(domains))
	fmt.Printf("  ▸ Concurrency   : %d worker\n\n", flagEnrichConcurrency)

	start := time.Now()
	enricher := enrich.New(flagEnrichConcurrency)

	var done int64
	var failed int64
	total := int64(len(domains))

	jobs := make(chan *storage.Domain, flagEnrichConcurrency*2)
	var wg sync.WaitGroup

	for i := 0; i < flagEnrichConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range jobs {
				enricher.Enrich(d)
				if err := db.UpdateEnrich(d); err != nil {
					atomic.AddInt64(&failed, 1)
				} else {
					n := atomic.AddInt64(&done, 1)
					if flagEnrichVerbose {
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
						fmt.Printf("\r  ► Ter-enrich: %d/%d domain", n, total)
					}
				}
			}
		}()
	}

	for _, d := range domains {
		jobs <- d
	}
	close(jobs)
	wg.Wait()

	fmt.Printf("\n\n  ✓ Enrich selesai — %d berhasil, %d gagal (%.1fs)\n",
		atomic.LoadInt64(&done),
		atomic.LoadInt64(&failed),
		time.Since(start).Seconds(),
	)

	stats := db.Stats()
	reporter.Print(stats, time.Since(start), flagEnrichDB)
	return nil
}

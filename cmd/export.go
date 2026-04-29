package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bssn1337/dorkscan/internal/storage"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data hasil scan ke CSV, JSON, atau TXT",
	Example: `  dorkscan export --db results.db --format csv --out hasil.csv
  dorkscan export --db results.db --format json --out hasil.json
  dorkscan export --db results.db --format txt --out domains.txt`,
	RunE: runExport,
}

var (
	flagExportDB     string
	flagExportFormat string
	flagExportOut    string
)

func init() {
	exportCmd.Flags().StringVar(&flagExportDB, "db", "results.db", "SQLite database sumber")
	exportCmd.Flags().StringVar(&flagExportFormat, "format", "csv", "Format export: csv, json, txt")
	exportCmd.Flags().StringVarP(&flagExportOut, "out", "o", "", "File output (default: stdout)")
}

func runExport(cmd *cobra.Command, args []string) error {
	db, err := storage.Open(flagExportDB)
	if err != nil {
		return fmt.Errorf("gagal buka database: %w", err)
	}
	defer db.Close()

	domains, err := db.GetAll()
	if err != nil {
		return fmt.Errorf("gagal baca data: %w", err)
	}

	out := os.Stdout
	if flagExportOut != "" {
		f, err := os.Create(flagExportOut)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	switch flagExportFormat {
	case "csv":
		err = exportCSV(domains, out)
	case "json":
		err = exportJSON(domains, out)
	case "txt":
		err = exportTXT(domains, out)
	default:
		return fmt.Errorf("format tidak dikenal: %s (pilihan: csv, json, txt)", flagExportFormat)
	}

	if err != nil {
		return err
	}

	if flagExportOut != "" {
		fmt.Printf("✓ Export selesai: %s (%d domain)\n", flagExportOut, len(domains))
	}
	return nil
}

func exportCSV(domains []*storage.Domain, out *os.File) error {
	w := csv.NewWriter(out)
	defer w.Flush()

	w.Write([]string{
		"domain", "tld", "url", "keyword_hit", "cms", "server",
		"php_version", "isp", "asn", "ip", "country", "hosting",
		"status_code", "ssl", "first_seen", "scan_id",
	})

	for _, d := range domains {
		w.Write([]string{
			d.Domain, d.TLD, d.URL, d.KeywordHit,
			d.CMS, d.Server, d.PHPVersion,
			d.ISP, d.ASN, d.IP, d.Country,
			boolStr(d.Hosting),
			fmt.Sprintf("%d", d.StatusCode),
			boolStr(d.SSL),
			d.FirstSeen.Format("2006-01-02"),
			d.ScanID,
		})
	}
	return w.Error()
}

func exportJSON(domains []*storage.Domain, out *os.File) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(domains)
}

func exportTXT(domains []*storage.Domain, out *os.File) error {
	for _, d := range domains {
		fmt.Fprintln(out, d.Domain)
	}
	return nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

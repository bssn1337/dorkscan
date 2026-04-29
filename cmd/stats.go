package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bssn1337/dorkscan/internal/reporter"
	"github.com/bssn1337/dorkscan/internal/storage"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Tampilkan statistik dari database hasil scan",
	Example: `  dorkscan stats --db results.db`,
	RunE:  runStats,
}

var flagStatsDB string

func init() {
	statsCmd.Flags().StringVar(&flagStatsDB, "db", "results.db", "SQLite database yang akan dianalisis")
}

func runStats(cmd *cobra.Command, args []string) error {
	db, err := storage.Open(flagStatsDB)
	if err != nil {
		return fmt.Errorf("gagal buka database: %w", err)
	}
	defer db.Close()

	stats := db.Stats()
	reporter.Print(stats, 0, flagStatsDB)
	return nil
}

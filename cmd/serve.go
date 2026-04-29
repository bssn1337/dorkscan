package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bssn1337/dorkscan/internal/viewer"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Jalankan web UI untuk melihat hasil scan",
	Example: `  dorkscan serve --db results.db
  dorkscan serve --db scan-indo-20260429.db --port 9000`,
	RunE: runServe,
}

var (
	flagServeDB   string
	flagServePort int
)

func init() {
	serveCmd.Flags().StringVar(&flagServeDB, "db", "results.db", "SQLite database yang akan ditampilkan")
	serveCmd.Flags().IntVar(&flagServePort, "port", 8080, "Port server")
}

func runServe(cmd *cobra.Command, args []string) error {
	fmt.Printf("\n  ▸ Database : %s\n", flagServeDB)
	fmt.Printf("  ▸ Port     : %d\n", flagServePort)

	srv, err := viewer.New(flagServeDB, flagServePort)
	if err != nil {
		return fmt.Errorf("gagal buka database: %w", err)
	}
	return srv.Start()
}

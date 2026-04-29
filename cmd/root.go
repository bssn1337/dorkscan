package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	clReset  = "\033[0m"
	clBold   = "\033[1m"
	clDim    = "\033[2m"
	clCyan   = "\033[36m"
	clOrange = "\033[38;5;208m"
	clGreen  = "\033[32m"
	clYellow = "\033[33m"
	clGray   = "\033[38;5;245m"
)

var rootCmd = &cobra.Command{
	Use:          "dorkscan",
	Short:        "Dorkscan — autonomous domain collector via Google Search API",
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, clReset+"\033[31m✗\033[0m "+err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(statsCmd)

	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Root: custom banner
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Print(helpTemplate())
	})

	// Subcommands: styled flag list
	for _, sub := range []*cobra.Command{scanCmd, exportCmd, statsCmd} {
		sub.SetHelpFunc(subHelpFunc)
	}
}

func subHelpFunc(cmd *cobra.Command, args []string) {
	fmt.Printf("\n  "+clOrange+clBold+"dorkscan %s"+clReset+" — %s\n\n", cmd.Name(), cmd.Short)

	if cmd.Example != "" {
		fmt.Println(clBold + clCyan + "  EXAMPLES" + clReset)
		fmt.Println(cmd.Example)
	}

	fmt.Println(clBold + clCyan + "  FLAGS" + clReset)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		short := ""
		if f.Shorthand != "" {
			short = clOrange + "  -" + f.Shorthand + clReset + ", "
		} else {
			short = "      "
		}
		def := ""
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
			def = clGray + " (default: " + f.DefValue + ")" + clReset
		}
		fmt.Printf("  %s"+clOrange+"--%-18s"+clReset+"  %s%s\n",
			short, f.Name, f.Usage, def)
	})
	fmt.Println()
}

func helpTemplate() string {
	return clOrange + `
  ██████╗  ██████╗ ██████╗ ██╗  ██╗███████╗ ██████╗ █████╗ ███╗   ██╗
  ██╔══██╗██╔═══██╗██╔══██╗██║ ██╔╝██╔════╝██╔════╝██╔══██╗████╗  ██║
  ██║  ██║██║   ██║██████╔╝█████╔╝ ███████╗██║     ███████║██╔██╗ ██║
  ██║  ██║██║   ██║██╔══██╗██╔═██╗ ╚════██║██║     ██╔══██║██║╚██╗██║
  ██████╔╝╚██████╔╝██║  ██║██║  ██╗███████║╚██████╗██║  ██║██║ ╚████║
  ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝
` + clReset +
		clDim + `  Autonomous domain harvester — Gatlab Security Research
  Collect • Enrich • Analyze` + clReset + `

` + clBold + clCyan + `COMMANDS` + clReset + `
  ` + clOrange + `scan` + clReset + `    Jalankan dork scan untuk mengumpulkan domain
  ` + clOrange + `export` + clReset + `  Export hasil ke CSV, JSON, atau TXT
  ` + clOrange + `stats` + clReset + `   Tampilkan statistik dari database

` + clBold + clCyan + `QUICK START` + clReset + `
  ` + clDim + `# scan + enrich domain pemerintah & sekolah` + clReset + `
  dorkscan scan -t .go.id,.ac.id,.sch.id -k "slot,judi,togel" --keys keys.txt -e

  ` + clDim + `# export hasil ke CSV` + clReset + `
  dorkscan export --format csv -o hasil.csv

  ` + clDim + `# lihat statistik` + clReset + `
  dorkscan stats

` + clBold + clCyan + `HELP PER COMMAND` + clReset + `
  dorkscan ` + clOrange + `scan` + clReset + ` --help
  dorkscan ` + clOrange + `export` + clReset + ` --help
  dorkscan ` + clOrange + `stats` + clReset + ` --help

` + clGray + `  github.com/bssn1337/dorkscan` + clReset + `

`
}

func usageTemplate() string {
	return `{{if .Runnable}}` + clBold + `USAGE` + clReset + `
  {{.UseLine}}
{{end}}{{if .HasAvailableSubCommands}}` + clBold + `USAGE` + clReset + `
  {{.CommandPath}} [command]
{{end}}{{if gt (len .Aliases) 0}}
` + clBold + `ALIASES` + clReset + `
  {{.NameAndAliases}}
{{end}}{{if .HasExample}}
` + clBold + `EXAMPLES` + clReset + `
{{.Example}}
{{end}}{{if .HasAvailableLocalFlags}}
` + clBold + `FLAGS` + clReset + `
{{.LocalFlags.FlagUsages | trimRightSpace}}
{{end}}{{if .HasAvailableSubCommands}}
` + clBold + `COMMANDS` + clReset + `{{range .Commands}}{{if .IsAvailableCommand}}
  ` + clOrange + `{{rpad .Name .NamePadding}}` + clReset + ` {{.Short}}{{end}}{{end}}

Use "{{.CommandPath}} [command] --help" for more information.
{{end}}`
}

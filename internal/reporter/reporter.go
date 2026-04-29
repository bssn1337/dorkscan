package reporter

import (
	"fmt"
	"sort"
	"time"

	"github.com/bssn1337/dorkscan/internal/storage"
)

type pair struct {
	k string
	v int
}

func sorted(m map[string]int) []pair {
	out := make([]pair, 0, len(m))
	for k, v := range m {
		out = append(out, pair{k, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].v > out[j].v })
	return out
}

func Print(s *storage.Stats, dur time.Duration, outFile string) {
	fmt.Println()
	fmt.Println("══════════════════════════════════════════════")
	fmt.Println("  DORKSCAN — Laporan Scan")
	fmt.Println("══════════════════════════════════════════════")
	if dur > 0 {
		fmt.Printf("  Durasi        : %s\n", dur.Round(time.Second))
	}
	fmt.Printf("  Total Domain  : %d\n\n", s.Total)

	fmt.Println("  TLD Breakdown")
	fmt.Println("  ────────────────────────────────────")
	for _, p := range sorted(s.ByTLD) {
		fmt.Printf("  %-25s %d\n", p.k, p.v)
	}

	if len(s.ByCMS) > 0 {
		fmt.Println()
		fmt.Println("  CMS Distribution")
		fmt.Println("  ────────────────────────────────────")
		for _, p := range sorted(s.ByCMS) {
			fmt.Printf("  %-25s %d\n", p.k, p.v)
		}
	}

	if len(s.ByISP) > 0 {
		fmt.Println()
		fmt.Println("  Top ISP / Hosting Provider")
		fmt.Println("  ────────────────────────────────────")
		for _, p := range sorted(s.ByISP) {
			fmt.Printf("  %-35s %d\n", p.k, p.v)
		}
	}

	fmt.Println()
	fmt.Printf("  Database : %s\n", outFile)
	fmt.Println("══════════════════════════════════════════════")
}

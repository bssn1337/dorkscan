package viewer

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed static/index.html
var indexHTML []byte

type Server struct {
	db   *sql.DB
	port int
}

type Domain struct {
	ID         int    `json:"id"`
	Domain     string `json:"domain"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	TLD        string `json:"tld"`
	KeywordHit string `json:"keyword_hit"`
	CMS        string `json:"cms"`
	ISP        string `json:"isp"`
	IP         string `json:"ip"`
	SSL        bool   `json:"ssl"`
	StatusCode int    `json:"status_code"`
	FirstSeen  string `json:"first_seen"`
}

type Stats struct {
	Total  int            `json:"total"`
	ByTLD  map[string]int `json:"by_tld"`
	ByCMS  map[string]int `json:"by_cms"`
	ByISP  map[string]int `json:"by_isp"`
	ByKW   map[string]int `json:"by_keyword"`
	ByIP   map[string]int `json:"by_ip"`
}

type PageResult struct {
	Data    []Domain `json:"data"`
	Total   int      `json:"total"`
	Page    int      `json:"page"`
	PerPage int      `json:"per_page"`
	Pages   int      `json:"pages"`
}

func New(dbPath string, port int) (*Server, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	return &Server{db: db, port: port}, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/domains", s.handleDomains)
	mux.HandleFunc("/api/export", s.handleExport)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("\n  ✓ Server berjalan di http://localhost%s\n", addr)
	fmt.Printf("  ✓ Buka browser dan akses URL di atas\n")
	fmt.Printf("  ✗ Tekan Ctrl+C untuk berhenti\n\n")
	return http.ListenAndServe(addr, mux)
}

func (s *Server) buildWhere(q map[string][]string) (string, []interface{}) {
	get := func(k string) string { return strings.TrimSpace(strings.Join(q[k], "")) }
	where := "WHERE 1=1"
	args := []interface{}{}
	if v := get("search"); v != "" {
		where += " AND (domain LIKE ? OR title LIKE ?)"
		args = append(args, "%"+v+"%", "%"+v+"%")
	}
	if v := get("tld"); v != "" {
		where += " AND tld=?"
		args = append(args, v)
	}
	if v := get("isp"); v != "" {
		where += " AND isp=?"
		args = append(args, v)
	}
	if v := get("cms"); v != "" {
		where += " AND cms LIKE ?"
		args = append(args, v+"%")
	}
	if v := get("ip"); v != "" {
		where += " AND ip=?"
		args = append(args, v)
	}
	return where, args
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	format := strings.TrimSpace(r.URL.Query().Get("format"))
	if format == "" {
		format = "txt"
	}
	qmap := map[string][]string{}
	for k, v := range r.URL.Query() {
		qmap[k] = v
	}
	where, args := s.buildWhere(qmap)

	rows, err := s.db.Query(
		"SELECT domain, url, tld, COALESCE(keyword_hit,''), COALESCE(cms,''), COALESCE(isp,''), COALESCE(ip,''), ssl, status_code FROM domains "+where+" ORDER BY id DESC",
		args...,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"dorkscan-export.csv\"")
		fmt.Fprintf(w, "domain,url,tld,keyword,cms,isp,ip,ssl,status_code\n")
		for rows.Next() {
			var domain, url, tld, kw, cms, isp, ip string
			var ssl, status int
			rows.Scan(&domain, &url, &tld, &kw, &cms, &isp, &ip, &ssl, &status)
			fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%d,%d\n", domain, url, tld, kw, cms, isp, ip, ssl, status)
		}
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"dorkscan-domains.txt\"")
		for rows.Next() {
			var domain, url, tld, kw, cms, isp, ip string
			var ssl, status int
			rows.Scan(&domain, &url, &tld, &kw, &cms, &isp, &ip, &ssl, &status)
			fmt.Fprintf(w, "%s\n", domain)
		}
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := &Stats{
		ByTLD: make(map[string]int),
		ByCMS: make(map[string]int),
		ByISP: make(map[string]int),
		ByKW:  make(map[string]int),
		ByIP:  make(map[string]int),
	}

	s.db.QueryRow("SELECT COUNT(*) FROM domains").Scan(&stats.Total)

	queries := []struct {
		m   map[string]int
		sql string
	}{
		{stats.ByTLD, "SELECT COALESCE(tld,'?'), COUNT(*) FROM domains GROUP BY tld ORDER BY 2 DESC"},
		{stats.ByCMS, "SELECT COALESCE(NULLIF(cms,''),'Unknown'), COUNT(*) FROM domains GROUP BY cms ORDER BY 2 DESC LIMIT 10"},
		{stats.ByISP, "SELECT COALESCE(NULLIF(isp,''),'Unknown'), COUNT(*) FROM domains GROUP BY isp ORDER BY 2 DESC LIMIT 10"},
		{stats.ByKW, "SELECT COALESCE(keyword_hit,'?'), COUNT(*) FROM domains GROUP BY keyword_hit ORDER BY 2 DESC LIMIT 15"},
		{stats.ByIP, "SELECT ip, COUNT(*) as c FROM domains WHERE ip!='' GROUP BY ip HAVING c > 1 ORDER BY c DESC LIMIT 50"},
	}

	for _, q := range queries {
		rows, err := s.db.Query(q.sql)
		if err != nil {
			continue
		}
		for rows.Next() {
			var k string
			var v int
			rows.Scan(&k, &v)
			q.m[k] = v
		}
		rows.Close()
	}

	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleDomains(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 50
	offset := (page - 1) * perPage

	qmap := map[string][]string{}
	for k, v := range q {
		qmap[k] = v
	}
	where, args := s.buildWhere(qmap)

	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM domains "+where, args...).Scan(&total)

	pages := (total + perPage - 1) / perPage

	rows, err := s.db.Query(
		"SELECT id, domain, COALESCE(url,''), COALESCE(title,''), COALESCE(tld,''), COALESCE(keyword_hit,''), COALESCE(cms,''), COALESCE(isp,''), COALESCE(ip,''), ssl, status_code, COALESCE(first_seen,'') FROM domains "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, perPage, offset)...,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var d Domain
		var ssl int
		rows.Scan(&d.ID, &d.Domain, &d.URL, &d.Title, &d.TLD, &d.KeywordHit, &d.CMS, &d.ISP, &d.IP, &ssl, &d.StatusCode, &d.FirstSeen)
		d.SSL = ssl == 1
		if len(d.FirstSeen) > 10 {
			d.FirstSeen = d.FirstSeen[:10]
		}
		domains = append(domains, d)
	}

	if domains == nil {
		domains = []Domain{}
	}

	json.NewEncoder(w).Encode(PageResult{
		Data:    domains,
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Pages:   pages,
	})
}

package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Domain struct {
	Domain     string
	URL        string
	Title      string
	Snippet    string
	KeywordHit string
	DorkUsed   string
	TLD        string
	ScanID     string
	IP         string
	ISP        string
	ASN        string
	Country    string
	Hosting    bool
	CMS        string
	Server     string
	PHPVersion string
	StatusCode int
	SSL        bool
	FirstSeen  time.Time
}

type Stats struct {
	Total int
	ByTLD map[string]int
	ByCMS map[string]int
	ByISP map[string]int
}

type DB struct {
	db *sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS domains (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		domain      TEXT UNIQUE NOT NULL,
		url         TEXT,
		title       TEXT,
		snippet     TEXT,
		keyword_hit TEXT,
		dork_used   TEXT,
		tld         TEXT,
		scan_id     TEXT,
		ip          TEXT,
		isp         TEXT,
		asn         TEXT,
		country     TEXT,
		hosting     INTEGER DEFAULT 0,
		cms         TEXT,
		server      TEXT,
		php_version TEXT,
		status_code INTEGER DEFAULT 0,
		ssl         INTEGER DEFAULT 0,
		first_seen  DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_tld ON domains(tld);
	CREATE INDEX IF NOT EXISTS idx_cms ON domains(cms);
	CREATE INDEX IF NOT EXISTS idx_scan ON domains(scan_id);
	`)
	return err
}

func (d *DB) Insert(r *Domain) error {
	_, err := d.db.Exec(`
	INSERT OR IGNORE INTO domains
		(domain, url, title, snippet, keyword_hit, dork_used, tld, scan_id,
		 ip, isp, asn, country, hosting, cms, server, php_version, status_code, ssl, first_seen)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.Domain, r.URL, r.Title, r.Snippet, r.KeywordHit, r.DorkUsed, r.TLD, r.ScanID,
		r.IP, r.ISP, r.ASN, r.Country, b2i(r.Hosting),
		r.CMS, r.Server, r.PHPVersion, r.StatusCode, b2i(r.SSL), r.FirstSeen,
	)
	return err
}

func (d *DB) Exists(domain string) bool {
	var n int
	d.db.QueryRow("SELECT COUNT(*) FROM domains WHERE domain=?", domain).Scan(&n)
	return n > 0
}

func (d *DB) Stats() *Stats {
	s := &Stats{
		ByTLD: make(map[string]int),
		ByCMS: make(map[string]int),
		ByISP: make(map[string]int),
	}
	d.db.QueryRow("SELECT COUNT(*) FROM domains").Scan(&s.Total)

	for _, q := range []struct {
		m   map[string]int
		sql string
	}{
		{s.ByTLD, "SELECT COALESCE(tld,'?'), COUNT(*) FROM domains GROUP BY tld ORDER BY 2 DESC"},
		{s.ByCMS, "SELECT COALESCE(cms,'Unknown'), COUNT(*) FROM domains WHERE cms!='' GROUP BY cms ORDER BY 2 DESC LIMIT 10"},
		{s.ByISP, "SELECT COALESCE(isp,'Unknown'), COUNT(*) FROM domains WHERE isp!='' GROUP BY isp ORDER BY 2 DESC LIMIT 10"},
	} {
		rows, err := d.db.Query(q.sql)
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
	return s
}

func (d *DB) GetAll() ([]*Domain, error) {
	rows, err := d.db.Query(`
	SELECT domain, url, title, snippet, keyword_hit, dork_used, tld, scan_id,
	       ip, isp, asn, country, hosting, cms, server, php_version, status_code, ssl, first_seen
	FROM domains ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Domain
	for rows.Next() {
		var r Domain
		var hosting, ssl int
		var ts string
		rows.Scan(
			&r.Domain, &r.URL, &r.Title, &r.Snippet, &r.KeywordHit, &r.DorkUsed, &r.TLD, &r.ScanID,
			&r.IP, &r.ISP, &r.ASN, &r.Country, &hosting, &r.CMS, &r.Server, &r.PHPVersion,
			&r.StatusCode, &ssl, &ts,
		)
		r.Hosting = hosting == 1
		r.SSL = ssl == 1
		r.FirstSeen, _ = time.Parse("2006-01-02T15:04:05Z", ts)
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (d *DB) GetUnenriched(limit int) ([]*Domain, error) {
	q := `SELECT domain, url, tld, COALESCE(ip,''), COALESCE(isp,'') FROM domains
	      WHERE (isp IS NULL OR isp='')
	         OR ((cms IS NULL OR cms='') AND status_code=0)
	      ORDER BY id`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := d.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Domain
	for rows.Next() {
		var r Domain
		rows.Scan(&r.Domain, &r.URL, &r.TLD, &r.IP, &r.ISP)
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (d *DB) UpdateEnrich(r *Domain) error {
	_, err := d.db.Exec(`
	UPDATE domains SET
		ip          = CASE WHEN ?1 != '' THEN ?1 ELSE ip END,
		isp         = CASE WHEN ?2 != '' THEN ?2 ELSE isp END,
		asn         = CASE WHEN ?3 != '' THEN ?3 ELSE asn END,
		country     = CASE WHEN ?4 != '' THEN ?4 ELSE country END,
		hosting     = CASE WHEN ?5 != 0  THEN ?5 ELSE hosting END,
		cms         = CASE WHEN ?6 != '' THEN ?6 ELSE cms END,
		server      = CASE WHEN ?7 != '' THEN ?7 ELSE server END,
		php_version = CASE WHEN ?8 != '' THEN ?8 ELSE php_version END,
		status_code = CASE WHEN ?9 != 0  THEN ?9 ELSE status_code END,
		ssl         = CASE WHEN ?10 != 0 THEN ?10 ELSE ssl END
	WHERE domain=?11`,
		r.IP, r.ISP, r.ASN, r.Country, b2i(r.Hosting),
		r.CMS, r.Server, r.PHPVersion, r.StatusCode, b2i(r.SSL),
		r.Domain,
	)
	return err
}

func (d *DB) Close() error { return d.db.Close() }

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

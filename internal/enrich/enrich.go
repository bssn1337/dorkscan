package enrich

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bssn1337/dorkscan/internal/storage"
)

type Enricher struct {
	sem     chan struct{}
	httpCli *http.Client
	ipCli   *http.Client
}

func New(concurrency int) *Enricher {
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}
	return &Enricher{
		sem: make(chan struct{}, concurrency),
		httpCli: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		ipCli: &http.Client{Timeout: 5 * time.Second},
	}
}

type ipAPI struct {
	Status  string `json:"status"`
	ISP     string `json:"isp"`
	AS      string `json:"as"`
	Country string `json:"country"`
	Hosting bool   `json:"hosting"`
	Query   string `json:"query"`
}

func (e *Enricher) Enrich(d *storage.Domain) {
	e.sem <- struct{}{}
	defer func() { <-e.sem }()

	// Resolve IP
	ips, err := net.LookupHost(d.Domain)
	if err == nil && len(ips) > 0 {
		d.IP = ips[0]
		e.lookupISP(d)
	}

	e.detectTech(d)
}

func (e *Enricher) lookupISP(d *storage.Domain) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,isp,as,country,hosting", d.IP)
	resp, err := e.ipCli.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var r ipAPI
	if json.NewDecoder(resp.Body).Decode(&r) != nil || r.Status != "success" {
		return
	}
	d.ISP = r.ISP
	d.ASN = r.AS
	d.Country = r.Country
	d.Hosting = r.Hosting
}

func (e *Enricher) detectTech(d *storage.Domain) {
	// Try HTTPS first, fallback to HTTP
	var resp *http.Response
	var err error
	for _, scheme := range []string{"https", "http"} {
		resp, err = e.httpCli.Get(scheme + "://" + d.Domain)
		if err == nil {
			d.SSL = scheme == "https"
			break
		}
	}
	if err != nil || resp == nil {
		return
	}
	defer resp.Body.Close()

	d.StatusCode = resp.StatusCode
	d.Server = resp.Header.Get("Server")

	if powered := resp.Header.Get("X-Powered-By"); strings.HasPrefix(powered, "PHP/") {
		d.PHPVersion = strings.TrimPrefix(powered, "PHP/")
	}

	// Read up to 64KB of body for fingerprinting
	buf := make([]byte, 65536)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])

	d.CMS = detectCMS(body, resp.Header)
}

func detectCMS(body string, h http.Header) string {
	b := strings.ToLower(body)

	switch {
	case strings.Contains(b, "wp-content") || strings.Contains(b, "wp-includes") || strings.Contains(b, "wp-json"):
		return "WordPress"
	case strings.Contains(b, "/components/com_") || strings.Contains(b, "joomla"):
		return "Joomla"
	case strings.Contains(b, "drupal") || strings.Contains(b, "sites/default/files"):
		return "Drupal"
	case strings.Contains(b, "moodle") || strings.Contains(b, "/mod/forum"):
		return "Moodle"
	case strings.Contains(b, "catalog/view/theme") || strings.Contains(b, "opencart"):
		return "OpenCart"
	case strings.Contains(b, "laravel") || h.Get("Set-Cookie") != "" && strings.Contains(h.Get("Set-Cookie"), "laravel_session"):
		return "Laravel"
	case strings.Contains(b, "codeigniter") || (h.Get("Set-Cookie") != "" && strings.Contains(h.Get("Set-Cookie"), "ci_session")):
		return "CodeIgniter"
	}

	// Try generator meta tag
	if gen := metaContent(body, "generator"); gen != "" {
		return gen
	}

	return "Unknown"
}

func metaContent(body, name string) string {
	lower := strings.ToLower(body)
	needle := `name="` + strings.ToLower(name) + `"`
	idx := strings.Index(lower, needle)
	if idx == -1 {
		needle = `name='` + strings.ToLower(name) + `'`
		idx = strings.Index(lower, needle)
		if idx == -1 {
			return ""
		}
	}
	sub := body[idx:]
	ci := strings.Index(strings.ToLower(sub), `content="`)
	if ci == -1 {
		return ""
	}
	ci += 9
	end := strings.Index(sub[ci:], `"`)
	if end == -1 {
		return ""
	}
	v := sub[ci : ci+end]
	if len(v) > 80 {
		v = v[:80]
	}
	return v
}

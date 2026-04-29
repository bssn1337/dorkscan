package dork

import "strings"

type Dork struct {
	Query   string
	TLD     string
	Keyword string
}

var defaultTemplates = []string{
	`site:{tld} "{keyword}"`,
	`site:{tld} intitle:"{keyword}"`,
	`site:{tld} inurl:"{keyword}"`,
}

func Generate(tlds, keywords, templates []string) []Dork {
	if len(templates) == 0 {
		templates = defaultTemplates
	}

	seen := map[string]bool{}
	var dorks []Dork

	for _, tld := range tlds {
		for _, kw := range keywords {
			for _, tmpl := range templates {
				q := strings.ReplaceAll(tmpl, "{tld}", tld)
				q = strings.ReplaceAll(q, "{keyword}", kw)
				q = strings.TrimSpace(q)
				if !seen[q] {
					seen[q] = true
					dorks = append(dorks, Dork{
						Query:   q,
						TLD:     tld,
						Keyword: kw,
					})
				}
			}
		}
	}
	return dorks
}

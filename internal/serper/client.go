package serper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const apiURL = "https://google.serper.dev/search"

type Client struct {
	keys    []string
	idx     int
	mu      sync.Mutex
	httpCli *http.Client
}

type Result struct {
	Organic []Organic `json:"organic"`
}

type Organic struct {
	Link    string `json:"link"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

func New(keys []string) *Client {
	return &Client{
		keys: keys,
		httpCli: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) nextKey() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.keys[c.idx%len(c.keys)]
	c.idx++
	return key
}

func (c *Client) Search(query string, page int) (*Result, error) {
	payload := map[string]interface{}{
		"q":    query,
		"gl":   "id",
		"hl":   "id",
		"num":  100,
		"page": page + 1,
	}

	body, _ := json.Marshal(payload)

	// Try each key once on failure
	for attempt := 0; attempt < len(c.keys); attempt++ {
		key := c.nextKey()

		req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-API-KEY", key)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpCli.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 {
			// Key exhausted, rotate
			continue
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("serper HTTP %d", resp.StatusCode)
		}

		var r Result
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return nil, err
		}
		return &r, nil
	}

	return nil, fmt.Errorf("all API keys exhausted or rate limited")
}

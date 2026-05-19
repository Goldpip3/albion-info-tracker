// Package aodp talks to the Albion Online Data Project HTTP API for
// market prices. Read-only — we don't upload prices, only consume them
// to estimate the silver value of loot.
//
// API docs: https://www.albion-online-data.com/
// Endpoint:  https://west.albion-online-data.com/api/v2/stats/prices/<uniquename>?locations=...&qualities=...
//
// We poll opportunistically: when a loot event lands with an item we
// haven't priced recently, the engine queues a fetch via QueueFetch.
// The client batches requests and respects a per-item TTL (default 5
// min). All prices are cached in memory only — no disk persistence.
package aodp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultEndpoint is the West Albion server's price endpoint. Use
// "east" or "europe" subdomain for other regions; we hardcode West
// because that's where Goldpipe plays. Configurable via Client.Endpoint.
const DefaultEndpoint = "https://west.albion-online-data.com/api/v2/stats/prices"

// DefaultTTL is how long we trust a cached price before refreshing.
const DefaultTTL = 5 * time.Minute

// PriceEntry is one row from AODP's API — for a given uniquename in a
// given city at a given quality, what's the current market floor.
type PriceEntry struct {
	ItemTypeId string `json:"item_id"`
	City       string `json:"city"`
	Quality    int    `json:"quality"`
	SellPriceMin int64 `json:"sell_price_min"`
	SellPriceMinDate string `json:"sell_price_min_date"`
}

// Price is the engine-facing condensed price: the best (highest) sell
// floor across all cities/qualities for an item. This is what you'd
// net by listing the item in the highest-paying city.
type Price struct {
	UniqueName string
	Silver     int64
	City       string
	FetchedAt  time.Time
}

// Client is the AODP HTTP wrapper with in-memory price caching.
type Client struct {
	Endpoint string
	TTL      time.Duration
	HTTP     *http.Client

	mu     sync.Mutex
	cache  map[string]Price
	queue  chan string
	loop   sync.Once
}

// New returns a configured Client with sensible defaults. Call Start to
// run the background fetcher goroutine.
func New() *Client {
	return &Client{
		Endpoint: DefaultEndpoint,
		TTL:      DefaultTTL,
		HTTP:     &http.Client{Timeout: 8 * time.Second},
		cache:    make(map[string]Price),
		queue:    make(chan string, 128),
	}
}

// Start kicks off the background fetcher. Safe to call once. Context
// cancellation stops the loop.
func (c *Client) Start(ctx context.Context) {
	c.loop.Do(func() {
		go c.run(ctx)
	})
}

// Lookup returns the cached price for a uniquename, or (zero, false)
// when nothing's cached. Doesn't trigger a fetch — call QueueFetch for
// that.
func (c *Client) Lookup(uniqueName string) (Price, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.cache[uniqueName]
	if !ok {
		return Price{}, false
	}
	if time.Since(p.FetchedAt) > c.TTL {
		return p, true // stale but still return — caller decides whether to use
	}
	return p, true
}

// QueueFetch hints that we'd like a fresh price for uniqueName. Non-
// blocking; drops the request when the queue is full. Skips items we
// already have a fresh price for.
func (c *Client) QueueFetch(uniqueName string) {
	if uniqueName == "" {
		return
	}
	c.mu.Lock()
	p, ok := c.cache[uniqueName]
	if ok && time.Since(p.FetchedAt) < c.TTL {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	select {
	case c.queue <- uniqueName:
	default:
		// Queue full; AODP will get refreshed on the next loot event.
	}
}

// run is the background fetcher loop. Pulls items off the queue and
// fetches their prices, throttled by AODP's recommended 1 req/sec.
func (c *Client) run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case un := <-c.queue:
			c.fetch(ctx, un)
			// Pace ourselves between calls.
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}
}

func (c *Client) fetch(ctx context.Context, uniqueName string) {
	// Strip the @<level> suffix — AODP keys items by base uniquename and
	// quality. We pick highest-priced quality below.
	base := uniqueName
	if i := strings.IndexByte(base, '@'); i >= 0 {
		base = base[:i]
	}
	url := fmt.Sprintf(
		"%s/%s.json?locations=Caerleon,Bridgewatch,Lymhurst,Martlock,Thetford,FortSterling,BlackMarket",
		c.Endpoint, base,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gda-meter/1.0 (+albion-meter-web.pages.dev)")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return
	}
	var entries []PriceEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return
	}
	best := int64(0)
	bestCity := ""
	for _, e := range entries {
		if e.SellPriceMin > best {
			best = e.SellPriceMin
			bestCity = e.City
		}
	}
	c.mu.Lock()
	c.cache[uniqueName] = Price{
		UniqueName: uniqueName,
		Silver:     best,
		City:       bestCity,
		FetchedAt:  time.Now(),
	}
	c.mu.Unlock()
}

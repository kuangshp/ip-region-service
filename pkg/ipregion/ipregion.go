// Package ipregion provides offline, concurrency-safe IPv4/IPv6 region lookup.
package ipregion

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/service"
)

var (
	ErrInvalidIP   = errors.New("invalid IP address")
	ErrNotFound    = errors.New("IP region not found")
	ErrUnsupported = errors.New("IP version database not configured")
	ErrClosed      = errors.New("IP region client is closed")
)

// Config accepts current-format xdb files. An empty path disables that family.
type Config struct {
	IPv4Path string
	IPv6Path string
}

type Region struct {
	IP          string `json:"ip"`
	Version     string `json:"version"`
	Country     string `json:"country"`
	Province    string `json:"province"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
	CountryCode string `json:"country_code"`
	Raw         string `json:"raw"`
}

// Client must be constructed with New and must not be copied after first use.
// A mutex also protects upstream's mutable IO counter in full-memory mode.
type Client struct {
	mu     sync.Mutex
	engine *service.Ip2Region
	v4, v6 bool
	closed bool
}

func New(cfg Config) (*Client, error) {
	if cfg.IPv4Path == "" && cfg.IPv6Path == "" {
		return nil, errors.New("at least one xdb database is required")
	}
	var v4, v6 *service.Config
	var err error
	if cfg.IPv4Path != "" {
		v4, err = service.NewV4Config(service.BufferCache, cfg.IPv4Path, 1)
		if err != nil {
			return nil, fmt.Errorf("load IPv4 database: %w", err)
		}
	}
	if cfg.IPv6Path != "" {
		v6, err = service.NewV6Config(service.BufferCache, cfg.IPv6Path, 1)
		if err != nil {
			return nil, fmt.Errorf("load IPv6 database: %w", err)
		}
	}
	engine, err := service.NewIp2Region(v4, v6)
	if err != nil {
		return nil, fmt.Errorf("create region engine: %w", err)
	}
	return &Client{engine: engine, v4: v4 != nil, v6: v6 != nil}, nil
}

// Lookup accepts a bare IP address; ports, CIDRs and IPv6 zones are rejected.
// IPv4-mapped IPv6 is normalized to IPv4. Unknown fields ("0") become empty.
func (c *Client) Lookup(input string) (Region, error) {
	addr, err := netip.ParseAddr(strings.TrimSpace(input))
	if err != nil || addr.Zone() != "" {
		return Region{}, ErrInvalidIP
	}
	addr = addr.Unmap()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return Region{}, ErrClosed
	}
	if (addr.Is4() && !c.v4) || (addr.Is6() && !c.v6) {
		return Region{}, ErrUnsupported
	}
	raw, err := c.engine.Search(addr.String())
	if err != nil {
		return Region{}, fmt.Errorf("lookup IP: %w", err)
	}
	if raw == "" {
		return Region{}, ErrNotFound
	}
	fields := strings.Split(raw, "|")
	if len(fields) != 5 {
		return Region{}, fmt.Errorf("unsupported region field format: expected 5 fields, got %d", len(fields))
	}
	for i := range fields {
		if fields[i] == "0" {
			fields[i] = ""
		}
	}
	version := "IPv6"
	if addr.Is4() {
		version = "IPv4"
	}
	return Region{IP: addr.String(), Version: version, Country: fields[0], Province: fields[1], City: fields[2], ISP: fields[3], CountryCode: fields[4], Raw: raw}, nil
}

// Close is idempotent and waits for any running lookup to finish.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.engine.Close()
	c.closed = true
}

package ipregion

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := New(Config{IPv4Path: filepath.Join("..", "..", "data", "ip2region_v4.xdb"), IPv6Path: filepath.Join("..", "..", "data", "ip2region_v6.xdb")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}
func TestLookup(t *testing.T) {
	c := newTestClient(t)
	for _, tc := range []struct{ input, ip, version string }{
		{" 113.92.157.29 ", "113.92.157.29", "IPv4"},
		{"240e:3b7:3272:d8d0:db09:c067:8d59:539e", "240e:3b7:3272:d8d0:db09:c067:8d59:539e", "IPv6"},
		{"::ffff:113.92.157.29", "113.92.157.29", "IPv4"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := c.Lookup(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.IP != tc.ip || got.Version != tc.version || got.Country != "中国" || got.Province != "广东省" || got.City != "深圳市" || got.ISP != "电信" || got.CountryCode != "CN" {
				t.Fatalf("unexpected region: %+v", got)
			}
		})
	}
	for _, ip := range []string{"", "999.1.1.1", "example.com", "1.1.1.1:80", "1.1.1.0/24", "fe80::1%en0"} {
		if _, err := c.Lookup(ip); !errors.Is(err, ErrInvalidIP) {
			t.Errorf("%q: %v", ip, err)
		}
	}
}
func TestConfigurationAndClose(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("empty config accepted")
	}
	if _, err := New(Config{IPv4Path: "missing.xdb"}); err == nil {
		t.Fatal("missing file accepted")
	}
	if _, err := New(Config{IPv4Path: "../../data/ip2region_v6.xdb"}); err == nil {
		t.Fatal("wrong IP version accepted")
	}
	c, err := New(Config{IPv4Path: "../../data/ip2region_v4.xdb"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Lookup("2001:4860:4860::8888"); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	c.Close()
	c.Close()
	if _, err := c.Lookup("1.1.1.1"); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
func TestConcurrentLookupAndClose(t *testing.T) {
	c := newTestClient(t)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ip := "113.92.157.29"
				if j%2 == 0 {
					ip = "240e:3b7:3272:d8d0:db09:c067:8d59:539e"
				}
				got, err := c.Lookup(ip)
				if err != nil || got.City != "深圳市" {
					t.Errorf("lookup: %+v, %v", got, err)
					return
				}
			}
		}()
	}
	wg.Wait()
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Lookup("1.1.1.1")
			if err != nil && !errors.Is(err, ErrClosed) {
				t.Error(err)
			}
		}()
	}
	c.Close()
	wg.Wait()
}

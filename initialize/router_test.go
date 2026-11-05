package initialize

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/box-im/ip-region-service/pkg/ipregion"
	"github.com/gin-gonic/gin"
)

func TestHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, err := ipregion.New(ipregion.Config{IPv4Path: "../data/ip2region_v4.xdb", IPv6Path: "../data/ip2region_v6.xdb"})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	router := Router(client)
	for _, tc := range []struct {
		name, method, path, body, contentType string
		status, code                          int
	}{
		{"get", "GET", "/api/v1/ip?ip=113.92.157.29", "", "", 200, 0},
		{"post", "POST", "/api/v1/ip", `{"ip":"113.92.157.29"}`, "application/json", 200, 0},
		{"ipv6", "GET", "/api/v1/ip?ip=240e:3b7:3272:d8d0:db09:c067:8d59:539e", "", "", 200, 0},
		{"missing", "GET", "/api/v1/ip", "", "", 400, 400},
		{"invalid", "GET", "/api/v1/ip?ip=bad", "", "", 400, 400},
		{"empty", "POST", "/api/v1/ip", `{}`, "application/json", 400, 400},
		{"malformed", "POST", "/api/v1/ip", `{`, "application/json", 400, 400},
		{"wrong type", "POST", "/api/v1/ip", `{"ip":123}`, "application/json", 400, 400},
		{"too large", "POST", "/api/v1/ip", `{"extra":"` + strings.Repeat("x", 5000) + `","ip":"1.1.1.1"}`, "application/json", 400, 400},
		{"content type", "POST", "/api/v1/ip", `ip=1.1.1.1`, "application/x-www-form-urlencoded", 415, 415},
		{"health", "GET", "/healthz", "", "", 200, 0},
		{"unknown", "GET", "/missing", "", "", 404, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			var body struct {
				Code int             `json:"code"`
				Data ipregion.Region `json:"data"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if res.Code != tc.status || body.Code != tc.code {
				t.Fatalf("status=%d body=%s", res.Code, res.Body)
			}
			if tc.status == 200 && tc.path != "/healthz" && body.Data.City != "深圳市" {
				t.Fatalf("wrong mapping: %s", res.Body)
			}
		})
	}
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/ip", nil)
	req.Header.Set("Origin", "https://frontend.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 204 || res.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("preflight failed: %+v", res)
	}
}

type failingLookup struct{ err error }

func (f failingLookup) Lookup(string) (ipregion.Region, error) { return ipregion.Region{}, f.err }
func TestLookupErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		err    error
		status int
	}{
		{ipregion.ErrNotFound, 404}, {ipregion.ErrUnsupported, 422}, {errors.New("secret database path"), 500},
	} {
		res := httptest.NewRecorder()
		Router(failingLookup{tc.err}).ServeHTTP(res, httptest.NewRequest("GET", "/api/v1/ip?ip=1.1.1.1", nil))
		if res.Code != tc.status || strings.Contains(res.Body.String(), "secret") {
			t.Fatalf("status=%d body=%s", res.Code, res.Body)
		}
	}
}

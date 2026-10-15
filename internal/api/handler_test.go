package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mrsibe/banner-fingerprint/internal/fingerprint"
)

func TestIdentifyHTTP(t *testing.T) {
	e, err := fingerprint.New()
	if err != nil {
		t.Fatal(err)
	}
	handler := New(e)
	for _, tt := range []struct {
		name, method, path, body string
		status                   int
	}{
		{"batch", "POST", "/api/v1/identify", `[{"ip":"192.0.2.1","port":80,"banner":"SSH-2.0-OpenSSH_8.9p1"},{"ip":"::1","port":22,"banner":"unknown"}]`, 200},
		{"health", "GET", "/healthz", "", 200},
		{"method", "GET", "/api/v1/identify", "", 405},
		{"not found", "POST", "/missing", "[]", 404},
		{"empty", "POST", "/api/v1/identify", "[]", 400},
		{"null", "POST", "/api/v1/identify", "null", 400},
		{"object", "POST", "/api/v1/identify", "{}", 400},
		{"malformed", "POST", "/api/v1/identify", "[", 400},
		{"extra", "POST", "/api/v1/identify", "[] []", 400},
		{"unknown field", "POST", "/api/v1/identify", `[{"extra":true}]`, 400},
		{"bad IP", "POST", "/api/v1/identify", `[{"ip":"localhost","port":22,"banner":"x"}]`, 400},
		{"bad port", "POST", "/api/v1/identify", `[{"ip":"::1","port":0,"banner":"x"}]`, 400},
		{"missing banner", "POST", "/api/v1/identify", `[{"ip":"::1","port":22}]`, 400},
		{"both banners", "POST", "/api/v1/identify", `[{"ip":"::1","port":22,"banner":"x","banner_base64":"eA=="}]`, 400},
		{"bad base64", "POST", "/api/v1/identify", `[{"ip":"::1","port":22,"banner_base64":"???"}]`, 400},
		{"oversized banner", "POST", "/api/v1/identify", `[{"ip":"::1","port":22,"banner":"` + strings.Repeat("x", MaxBannerBytes+1) + `"}]`, 400},
		{"too many", "POST", "/api/v1/identify", `[` + strings.Repeat(`{"ip":"::1","port":22,"banner":"x"},`, MaxBatchSize) + `{"ip":"::1","port":22,"banner":"x"}]`, 400},
		{"too large", "POST", "/api/v1/identify", strings.Repeat(" ", MaxBodyBytes) + `[]`, 413},
		{"large trailing whitespace", "POST", "/api/v1/identify", `[]` + strings.Repeat(" ", MaxBodyBytes), 413},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			if w.Code != tt.status {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
			if tt.name == "batch" {
				var results []fingerprint.IdentifyResult
				if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
					t.Fatal(err)
				}
				if len(results) != 2 || results[0].Product != "OpenSSH" || results[0].Port != 80 || results[1].Confidence != 0 {
					t.Fatalf("unexpected results: %+v", results)
				}
			}
		})
	}
}

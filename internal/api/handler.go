// Package api implements the bounded batch identification HTTP interface.
package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/mrsibe/banner-fingerprint/internal/fingerprint"
)

const (
	MaxBodyBytes   = 4 << 20
	MaxBatchSize   = 1000
	MaxBannerBytes = 64 << 10
)

func New(engine *fingerprint.Engine) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/v1/identify", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var inputs []fingerprint.ScanInput
		if err := dec.Decode(&inputs); err != nil {
			decodeError(w, err)
			return
		}
		var extra any
		if err := dec.Decode(&extra); err != io.EOF {
			if err == nil {
				fail(w, http.StatusBadRequest, "expected a single JSON array")
			} else {
				decodeError(w, err)
			}
			return
		}
		if len(inputs) == 0 || len(inputs) > MaxBatchSize {
			fail(w, http.StatusBadRequest, "batch must contain 1 to 1000 items")
			return
		}
		results := make([]fingerprint.IdentifyResult, 0, len(inputs))
		for i, in := range inputs {
			if net.ParseIP(in.IP) == nil || in.Port < 1 || in.Port > 65535 {
				fail(w, http.StatusBadRequest, fmt.Sprintf("item %d: invalid IP or port", i))
				return
			}
			if len(in.Banner) > MaxBannerBytes || len(in.BannerBase64) > base64.StdEncoding.EncodedLen(MaxBannerBytes) {
				fail(w, http.StatusBadRequest, fmt.Sprintf("item %d: banner exceeds 65536 bytes", i))
				return
			}
			if in.BannerBase64 != "" {
				data, err := base64.StdEncoding.Strict().DecodeString(in.BannerBase64)
				if err != nil || len(data) > MaxBannerBytes {
					fail(w, http.StatusBadRequest, fmt.Sprintf("item %d: invalid or oversized banner_base64", i))
					return
				}
			}
			result, err := engine.Identify(in)
			if err != nil {
				fail(w, http.StatusBadRequest, fmt.Sprintf("item %d: %s", i, err))
				return
			}
			results = append(results, result)
		}
		writeJSON(w, http.StatusOK, results)
	})
	return mux
}

func decodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		fail(w, http.StatusRequestEntityTooLarge, "request body exceeds 4 MiB")
		return
	}
	fail(w, http.StatusBadRequest, "invalid JSON request")
}
func fail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

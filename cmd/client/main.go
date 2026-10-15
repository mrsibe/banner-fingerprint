package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mrsibe/banner-fingerprint/internal/api"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	input := flag.String("input", "examples/input.json", "input JSON file")
	server := flag.String("server", "http://localhost:8080", "server base URL")
	timeout := flag.Duration("timeout", 15*time.Second, "HTTP request timeout")
	flag.Parse()
	f, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, api.MaxBodyBytes+1))
	if err != nil {
		return err
	}
	if len(data) > api.MaxBodyBytes {
		return fmt.Errorf("input exceeds 4 MiB")
	}
	if !json.Valid(data) {
		return fmt.Errorf("input is not valid JSON")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(*server, "/")+"/fingerprint", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, api.MaxBodyBytes+1))
	if err != nil {
		return err
	}
	if len(body) > api.MaxBodyBytes {
		return fmt.Errorf("response exceeds 4 MiB")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err != nil {
		return fmt.Errorf("invalid server response: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, pretty.String())
	return err
}

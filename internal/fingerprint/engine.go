// Package fingerprint identifies passive service banners without network probing.
package fingerprint

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mrsibe/banner-fingerprint/rules"
)

// ScanInput describes one already collected banner. Exactly one banner field is required.
type ScanInput struct {
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	Banner       string `json:"banner,omitempty"`
	BannerBase64 string `json:"banner_base64,omitempty"`
}

// IdentifyResult preserves input identity; empty fields and confidence zero mean unknown.
type IdentifyResult struct {
	IP         string  `json:"ip"`
	Port       int     `json:"port"`
	Protocol   string  `json:"protocol"`
	Product    string  `json:"product"`
	Version    string  `json:"version"`
	OSHint     string  `json:"os_hint"`
	Confidence float64 `json:"confidence"`
}

type rule struct {
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	Product      string `json:"product"`
	Regex        string `json:"regex"`
	VersionGroup int    `json:"version_group"`
	Priority     int    `json:"priority"`
	compiled     *regexp.Regexp
}

// Engine is immutable after construction and safe for concurrent use.
type Engine struct{ rules []rule }

func New() (*Engine, error) { return load(rules.JSON) }

func load(data []byte) (*Engine, error) {
	var file struct {
		Rules []rule `json:"rules"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("decode rules: %w", err)
	}
	if len(file.Rules) == 0 {
		return nil, fmt.Errorf("empty rules")
	}
	seen := make(map[string]bool)
	for i := range file.Rules {
		r := &file.Rules[i]
		if r.Name == "" || r.Protocol == "" || seen[r.Name] {
			return nil, fmt.Errorf("invalid or duplicate rule %q", r.Name)
		}
		seen[r.Name] = true
		re, err := regexp.Compile(r.Regex)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.Name, err)
		}
		if r.Regex == "" || r.VersionGroup < 0 || r.VersionGroup > re.NumSubexp() {
			return nil, fmt.Errorf("invalid capture group or pattern in %s", r.Name)
		}
		r.compiled = re
	}
	return &Engine{rules: file.Rules}, nil
}

// Identify uses only banner contents, never the port, as identification evidence.
// Binary MySQL input accepts a protocol-10 payload or a complete handshake packet.
func (e *Engine) Identify(in ScanInput) (IdentifyResult, error) {
	out := IdentifyResult{IP: in.IP, Port: in.Port}
	if (in.Banner == "") == (in.BannerBase64 == "") {
		return out, fmt.Errorf("exactly one of banner and banner_base64 is required")
	}
	data := []byte(in.Banner)
	if in.BannerBase64 != "" {
		var err error
		data, err = base64.StdEncoding.Strict().DecodeString(in.BannerBase64)
		if err != nil || len(data) == 0 {
			return out, fmt.Errorf("banner_base64 must encode nonempty bytes")
		}
		if version, ok := mysqlVersion(data); ok {
			out.Protocol = "MySQL"
			out.Product = "MySQL"
			out.Version = version
			out.Confidence = .95
			if strings.Contains(strings.ToLower(version), "mariadb") {
				out.Product = "MariaDB"
			}
			out.OSHint = osHint(version)
			return out, nil
		}
	}
	text := string(data)
	isHTTP := strings.HasPrefix(text, "HTTP/")
	// Do not treat HTTP body content as headers or another service's banner.
	if isHTTP {
		text = strings.SplitN(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n", 2)[0]
	}
	headerText := strings.SplitN(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n", 2)[0]
	bestPriority, bestSpecificity, bestLength := -int(^uint(0)>>1)-1, -1, -1
	for _, r := range e.rules {
		if isHTTP && r.Protocol != "HTTP" {
			continue
		}
		candidate := text
		if r.Protocol == "HTTP" {
			candidate = headerText
		}
		match := r.compiled.FindStringSubmatch(candidate)
		if match == nil {
			continue
		}
		version := ""
		if r.VersionGroup > 0 {
			version = match[r.VersionGroup]
		}
		specificity := 0
		if r.Product != "" {
			specificity = 1
		}
		if version != "" {
			specificity++
		}
		if r.Priority < bestPriority || (r.Priority == bestPriority && (specificity < bestSpecificity || (specificity == bestSpecificity && len(match[0]) <= bestLength))) {
			continue
		}
		bestPriority, bestSpecificity, bestLength = r.Priority, specificity, len(match[0])
		out.Protocol, out.Product, out.Version = r.Protocol, r.Product, version
		out.Confidence = .70
		if r.Product != "" {
			out.Confidence = .85
			if version != "" {
				out.Confidence = .95
			}
		}
	}
	if out.Protocol != "" {
		out.OSHint = osHint(text)
	}
	return out, nil
}

var mysqlVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+[A-Za-z0-9._+~:-]*$`)

func mysqlVersion(data []byte) (string, bool) {
	// A complete packet has a three-byte little-endian length and sequence ID 0.
	if len(data) >= 5 && data[4] == 10 && data[3] == 0 {
		n := int(data[0]) | int(data[1])<<8 | int(data[2])<<16
		if n != len(data)-4 {
			return "", false
		}
		data = data[4:]
	}
	if len(data) < 2 || data[0] != 10 {
		return "", false
	}
	end := bytes.IndexByte(data[1:], 0)
	if end < 0 || end > 255 {
		return "", false
	}
	version := string(data[1 : 1+end])
	// Require connection ID, auth seed and the following filler byte, not just a prefix.
	tail := data[2+end:]
	if len(tail) < 13 || tail[12] != 0 || !mysqlVersionPattern.MatchString(version) {
		return "", false
	}
	return version, true
}

var osPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"Ubuntu", regexp.MustCompile(`(?i)\bubuntu\b`)},
	{"Debian", regexp.MustCompile(`(?i)\bdebian\b`)},
	{"Windows", regexp.MustCompile(`(?i)\b(?:windows|win32|win64)\b`)},
	{"FreeBSD", regexp.MustCompile(`(?i)\bfreebsd\b`)},
	{"CentOS", regexp.MustCompile(`(?i)\bcentos\b`)},
	{"Linux", regexp.MustCompile(`(?i)\blinux\b`)},
}

func osHint(s string) string {
	for _, p := range osPatterns {
		if p.re.MatchString(s) {
			return p.name
		}
	}
	return ""
}

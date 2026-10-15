package fingerprint

import (
	"encoding/base64"
	"testing"
)

func TestExamplesAndFallbacks(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, banner, protocol, product, version, os string
		confidence                                   float64
	}{
		{"openssh", "SSH-2.0-OpenSSH_8.9p1 Ubuntu-3ubuntu0.6", "SSH", "OpenSSH", "8.9p1", "Ubuntu", .95},
		{"nginx", "HTTP/1.1 200 OK\r\nServer: nginx/1.24.0\r\n", "HTTP", "nginx", "1.24.0", "", .95},
		{"apache", "HTTP/1.1 200 OK\r\nServer: Apache/2.4.57 (Debian)\r\n", "HTTP", "Apache httpd", "2.4.57", "Debian", .95},
		{"redis", "# Server\r\nredis_version:7.2.4\r\n", "Redis", "Redis", "7.2.4", "", .95},
		{"ftp", "220 ProFTPD 1.3.8 Server ready", "FTP", "ProFTPD", "1.3.8", "", .95},
		{"jetty", "HTTP/1.1 200 OK\r\nServer: Jetty(9.4.54.v20240208)\r\n", "HTTP", "Jetty", "9.4.54.v20240208", "", .95},
		{"unknown", "unrecognized service", "", "", "", "", 0},
		{"ssh protocol only", "SSH-2.0-custom", "SSH", "", "", "", .70},
		{"http protocol only", "HTTP/1.1 404 Not Found\r\n", "HTTP", "", "", "", .70},
		{"product only", "HTTP/1.1 200 OK\r\nServer: nginx\r\n", "HTTP", "nginx", "", "", .85},
		{"no reflected product", "HTTP/1.1 200 OK\r\n\r\nnginx/1.2.3", "HTTP", "", "", "", .70},
		{"no port inference", "hello Ubuntu", "", "", "", "", 0},
		{"HTTP body SSH", "HTTP/1.1 200 OK\r\n\r\nOpenSSH_9.9", "HTTP", "", "", "", .70},
		{"HTTP body full SSH", "HTTP/1.1 200 OK\r\n\r\nSSH-2.0-OpenSSH_9.9", "HTTP", "", "", "", .70},
		{"HTTP body fake header", "HTTP/1.1 200 OK\r\n\r\nServer: nginx/1.24.0\r\n", "HTTP", "", "", "", .70},
		{"LF body fake header", "HTTP/1.1 200 OK\n\nServer: nginx/1.24.0\n", "HTTP", "", "", "", .70},
		{"HTTP body Redis", "HTTP/1.1 200 OK\r\n\r\nredis_version:7.2.4\r\n", "HTTP", "", "", "", .70},
		{"HTTP body OS", "HTTP/1.1 200 OK\r\n\r\nUbuntu", "HTTP", "", "", "", .70},
		{"Jetty boundary", "HTTP/1.1 200 OK\r\nServer: JettyNotJetty\r\n\r\n", "HTTP", "", "", "", .70},
		{"IIS boundary", "HTTP/1.1 200 OK\r\nServer: Microsoft-IISAlternative\r\n\r\n", "HTTP", "", "", "", .70},
		{"unframed openssh", "OpenSSH_9.9", "", "", "", "", 0},
		{"vsftpd", "220 (vsFTPd 3.0.5)", "FTP", "vsftpd", "3.0.5", "", .95},
		{"iis", "HTTP/1.1 200 OK\r\nServer: Microsoft-IIS/10.0\r\n", "HTTP", "Microsoft IIS", "10.0", "", .95},
		{"redis auth", "-NOAUTH Authentication required.\r\n", "Redis", "", "", "", .70},
		{"mysql text", "MySQL/8.0.36", "MySQL", "MySQL", "8.0.36", "", .95},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Deliberately inconsistent port: banner evidence must win.
			got, err := e.Identify(ScanInput{IP: "192.0.2.1", Port: 443, Banner: tt.banner})
			if err != nil {
				t.Fatal(err)
			}
			want := IdentifyResult{IP: "192.0.2.1", Port: 443, Protocol: tt.protocol, Product: tt.product, Version: tt.version, OSHint: tt.os, Confidence: tt.confidence}
			if got != want {
				t.Fatalf("got %+v; want %+v", got, want)
			}
		})
	}
}

func handshake(version string) []byte {
	payload := append([]byte{10}, []byte(version)...)
	payload = append(payload, 0, 1, 0, 0, 0, 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 0)
	return payload
}
func TestMySQLBinary(t *testing.T) {
	e, _ := New()
	for _, version := range []string{"8.0.36", "5.5.5-10.11.6-MariaDB"} {
		payload := handshake(version)
		packet := append([]byte{byte(len(payload)), 0, 0, 0}, payload...)
		for _, data := range [][]byte{payload, packet} {
			got, err := e.Identify(ScanInput{BannerBase64: base64.StdEncoding.EncodeToString(data)})
			product := "MySQL"
			if version != "8.0.36" {
				product = "MariaDB"
			}
			if err != nil || got.Protocol != "MySQL" || got.Version != version || got.Product != product {
				t.Fatalf("got %+v, %v", got, err)
			}
		}
	}
	for _, data := range [][]byte{{10}, []byte("\x0a8.0.36\x00"), {40, 0, 0, 0, 10, '8', 0}, handshake("not-a-version")} {
		got, err := e.Identify(ScanInput{BannerBase64: base64.StdEncoding.EncodeToString(data)})
		if err != nil || got.Protocol != "" {
			t.Fatalf("malformed binary identified: %+v %v", got, err)
		}
	}
}
func TestInvalidInput(t *testing.T) {
	e, _ := New()
	for _, in := range []ScanInput{{}, {Banner: "x", BannerBase64: "eA=="}, {BannerBase64: "!!!"}, {BannerBase64: "\n"}} {
		if _, err := e.Identify(in); err == nil {
			t.Fatalf("accepted %+v", in)
		}
	}
}
func TestRuleSelection(t *testing.T) {
	// Priority wins, then product/version specificity, then matched length, then file order.
	for _, tt := range []struct{ rules, want string }{
		{`[{"name":"low","protocol":"A","regex":"abc","priority":1},{"name":"high","protocol":"B","regex":"a","priority":2}]`, "B"},
		{`[{"name":"generic","protocol":"A","regex":"abc"},{"name":"specific","protocol":"B","product":"p","regex":"a"}]`, "B"},
		{`[{"name":"short","protocol":"A","regex":"a"},{"name":"long","protocol":"B","regex":"abc"}]`, "B"},
		{`[{"name":"first","protocol":"A","regex":"abc"},{"name":"second","protocol":"B","regex":"abc"}]`, "A"},
	} {
		e, err := load([]byte(`{"rules":` + tt.rules + `}`))
		if err != nil {
			t.Fatal(err)
		}
		got, _ := e.Identify(ScanInput{Banner: "abc"})
		if got.Protocol != tt.want {
			t.Fatalf("got %s want %s", got.Protocol, tt.want)
		}
	}
}
func TestInvalidRules(t *testing.T) {
	for _, s := range []string{`{`, `{"rules":[]}`, `{"rules":[{"name":"x","protocol":"X","regex":"["}]}`, `{"rules":[{"name":"x","protocol":"X","regex":"x","version_group":1}]}`, `{"rules":[{"name":"x","protocol":"X","regex":"x"},{"name":"x","protocol":"X","regex":"x"}]}`} {
		if _, err := load([]byte(s)); err == nil {
			t.Fatalf("accepted %s", s)
		}
	}
}

func FuzzIdentify(f *testing.F) {
	e, err := New()
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte("SSH-2.0-OpenSSH_8.9p1"))
	f.Add(handshake("8.0.36"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		if _, err := e.Identify(ScanInput{BannerBase64: base64.StdEncoding.EncodeToString(data)}); err != nil {
			t.Fatal(err)
		}
	})
}

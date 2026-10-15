package fingerprint

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

// These expected values are maintained separately from engine output, so rule
// regressions cannot silently regenerate their own passing expectations.
func TestTaskSamples(t *testing.T) {
	inputJSON, err := os.ReadFile("../../examples/task-input.json")
	if err != nil {
		t.Fatal(err)
	}
	expectedJSON, err := os.ReadFile("../../examples/task-expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var inputs []ScanInput
	var expected []IdentifyResult
	if err := json.Unmarshal(inputJSON, &inputs); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(expectedJSON, &expected); err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 20 || len(expected) != 20 {
		t.Fatal("expected all 20 task samples")
	}
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for i, in := range inputs {
		t.Run(in.IP, func(t *testing.T) {
			got, err := e.Identify(in)
			if err != nil || got != expected[i] {
				t.Fatalf("got %+v, %v; want %+v", got, err, expected[i])
			}
			// Binary/text transport and port changes must not change fingerprint evidence.
			if in.Banner != "" {
				encoded := in
				encoded.Banner = ""
				encoded.BannerBase64 = base64.StdEncoding.EncodeToString([]byte(in.Banner))
				got, err = e.Identify(encoded)
				if err != nil || got != expected[i] {
					t.Fatalf("base64 got %+v, %v; want %+v", got, err, expected[i])
				}
			}
			in.Port = 54321
			got, err = e.Identify(in)
			want := expected[i]
			want.Port = in.Port
			if err != nil || got != want {
				t.Fatalf("port influenced result: %+v, %v", got, err)
			}
		})
	}
}

func TestTruncatedMySQLNegativeCases(t *testing.T) {
	e, _ := New()
	for _, data := range [][]byte{
		[]byte("J\x00\x00\x01\x0a8.0.32\x00"),    // Wrong sequence.
		[]byte("J\x00\x00\x00\x0b8.0.32\x00"),    // Wrong protocol.
		[]byte("J\x00\x00\x00\x0a8.0.32"),        // Missing terminator.
		[]byte("\x01\x00\x00\x00\x0a8.0.32\x00"), // Impossible packet length.
		[]byte("J\x00\x00\x00\x0anot-a-version\x00"),
	} {
		got, err := e.Identify(ScanInput{BannerBase64: base64.StdEncoding.EncodeToString(data)})
		if err != nil || got.Protocol != "unknown" {
			t.Fatalf("malformed prefix accepted: %+v %v", got, err)
		}
	}
}

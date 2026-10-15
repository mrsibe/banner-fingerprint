// Package rules contains the immutable, embedded fingerprint database.
package rules

import _ "embed"

// JSON is the bundled rule set. Changes require rebuilding the binary.
//
//go:embed fingerprints.json
var JSON []byte

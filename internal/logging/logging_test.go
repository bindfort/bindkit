package logging

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	got := Redact("Authorization: Bearer secret123 api_key=abc token=def")
	for _, secret := range []string{"secret123", "abc", "def"} {
		if strings.Contains(got, secret) {
			t.Fatalf("secret %q was not redacted: %s", secret, got)
		}
	}
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeEndpoint(t *testing.T) {
	in := "https://user:pass@localhost:9200"
	out := sanitizeEndpoint(in)
	if out == in {
		t.Fatalf("expected redaction, got same value: %s", out)
	}
	if filepath.Ext(out) == "" && out == "" {
		t.Fatal("unexpected empty output")
	}
}

func TestValidateOutputPath(t *testing.T) {
	cases := []struct{
		name string
		path string
		ok   bool
	}{
		{"relative ok", "inventory.json", true},
		{"absolute not ok", "/tmp/inventory.json", false},
		{"traversal not ok", "../../etc/passwd", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateOutputPath(c.path)
			if (err == nil) != c.ok {
				t.Fatalf("expected ok=%v, got err=%v", c.ok, err)
			}
		})
	}
}

func TestWriteOutputSafely(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	data := []byte("{}")
	if err := writeOutputSafely(path, data); err != nil {
		t.Fatalf("writeOutputSafely error: %v", err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	mode := st.Mode().Perm()
	if mode != 0o600 {
		t.Fatalf("expected 0600 perms, got %o", mode)
	}
} 
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "hoard", "config.env")
	in := Config{VaultRoot: "/v", DefaultScope: "personal", MirrorExclude: "/tmp/a/*:/tmp/b/*", Disable: "nudge"}
	if err := Write(p, in); err != nil {
		t.Fatal(err)
	}
	out, err := Read(p)
	if err != nil || out != in {
		t.Fatalf("got %+v, %v", out, err)
	}
}

func TestReadParsesShellStyle(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.env")
	os.WriteFile(p, []byte("# c\nHOARD_VAULT_ROOT=\"/v\"\nHOARD_DEFAULT_SCOPE=work\nOTHER=1\n"), 0o644)
	c, err := Read(p)
	if err != nil || c.VaultRoot != "/v" || c.DefaultScope != "work" {
		t.Fatalf("got %+v, %v", c, err)
	}
}

func TestReadMissing(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "nope"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
}

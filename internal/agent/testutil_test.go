package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Duro-Labs/hoard"
	"github.com/Duro-Labs/hoard/internal/assets"
	"github.com/Duro-Labs/hoard/internal/env"
)

func testEnv(t *testing.T) env.Env {
	t.Helper()
	h := t.TempDir()
	e := env.Resolve(h, func(string) string { return "" })
	if err := assets.Extract(hoard.Assets, e.DataDir); err != nil {
		t.Fatal(err)
	}
	return e
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func write(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func backups(t *testing.T, dir string) []string {
	m, _ := filepath.Glob(filepath.Join(dir, "*.hoard-bak-*"))
	return m
}

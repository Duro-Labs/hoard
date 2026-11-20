package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Duro-Labs/hoard/internal/config"
	"github.com/Duro-Labs/hoard/internal/env"
)

func fixed() time.Time { return time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC) }

func TestInstallRequiresVaultFirstTime(t *testing.T) {
	e := env.Resolve(t.TempDir(), func(string) string { return "" })
	var out, errb bytes.Buffer
	if code := run([]string{"install", "--agent", "claude"}, e, &out, &errb, fixed); code == 0 {
		t.Fatal("want failure without --vault")
	}
	if !strings.Contains(errb.String(), "--vault") {
		t.Fatal(errb.String())
	}
}

func TestInstallWritesConfigAndAssets(t *testing.T) {
	h := t.TempDir()
	e := env.Resolve(h, func(string) string { return "" })
	vault := filepath.Join(h, "vault")
	os.MkdirAll(vault, 0o755)
	var out, errb bytes.Buffer
	if code := run([]string{"install", "--agent", "claude", "--vault", vault}, e, &out, &errb, fixed); code != 0 {
		t.Fatal(errb.String())
	}
	c, err := config.Read(e.ConfigFile)
	if err != nil || c.VaultRoot != vault || c.DefaultScope != "personal" {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := os.Stat(filepath.Join(e.DataDir, "hooks", "_lib.sh")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run([]string{"install", "--agent", "codex"}, e, &out, &errb, fixed); code != 0 {
		t.Fatal("second agent should reuse config:", errb.String())
	}
}

func TestInstallRejectsMissingVaultDir(t *testing.T) {
	e := env.Resolve(t.TempDir(), func(string) string { return "" })
	var out, errb bytes.Buffer
	if code := run([]string{"install", "--agent", "claude", "--vault", "/does/not/exist"}, e, &out, &errb, fixed); code == 0 {
		t.Fatal("want failure")
	}
}

func TestDoctorReportsConfig(t *testing.T) {
	e := env.Resolve(t.TempDir(), func(string) string { return "" })
	var out, errb bytes.Buffer
	run([]string{"doctor"}, e, &out, &errb, fixed)
	if !strings.Contains(out.String(), "config") {
		t.Fatal(out.String())
	}
}

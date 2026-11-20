package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	VaultRoot, DefaultScope, MirrorExclude, Disable string
}

func (c *Config) field(k string) *string {
	switch k {
	case "HOARD_VAULT_ROOT":
		return &c.VaultRoot
	case "HOARD_DEFAULT_SCOPE":
		return &c.DefaultScope
	case "HOARD_MIRROR_EXCLUDE":
		return &c.MirrorExclude
	case "HOARD_DISABLE":
		return &c.Disable
	}
	return nil
}

func Read(path string) (Config, error) {
	var c Config
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if p := c.field(k); ok && p != nil {
			*p = strings.TrimSuffix(strings.TrimPrefix(v, `"`), `"`)
		}
	}
	return c, s.Err()
}

func Write(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	for _, kv := range [][2]string{{"HOARD_VAULT_ROOT", c.VaultRoot}, {"HOARD_DEFAULT_SCOPE", c.DefaultScope}, {"HOARD_MIRROR_EXCLUDE", c.MirrorExclude}, {"HOARD_DISABLE", c.Disable}} {
		if strings.ContainsAny(kv[1], "\"\n") {
			return fmt.Errorf("%s: value may not contain quotes or newlines", kv[0])
		}
		fmt.Fprintf(&b, "%s=\"%s\"\n", kv[0], kv[1])
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

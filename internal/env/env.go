package env

import (
	"os"
	"path/filepath"
)

type Env struct {
	Home, ConfigFile, DataDir, StateDir, ClaudeDir, CodexDir, HermesHome string
}

func FromOS() (Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Env{}, err
	}
	return Resolve(home, os.Getenv), nil
}

func Resolve(home string, getenv func(string) string) Env {
	or := func(k, def string) string {
		if v := getenv(k); v != "" {
			return v
		}
		return def
	}
	return Env{
		Home:       home,
		ConfigFile: filepath.Join(or("XDG_CONFIG_HOME", filepath.Join(home, ".config")), "hoard", "config.env"),
		DataDir:    filepath.Join(or("XDG_DATA_HOME", filepath.Join(home, ".local", "share")), "hoard"),
		StateDir:   filepath.Join(or("XDG_STATE_HOME", filepath.Join(home, ".local", "state")), "hoard"),
		ClaudeDir:  or("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude")),
		CodexDir:   or("CODEX_HOME", filepath.Join(home, ".codex")),
		HermesHome: or("HERMES_HOME", filepath.Join(home, ".hermes")),
	}
}

func (e Env) Hook(name string) string { return filepath.Join(e.DataDir, "hooks", name) }

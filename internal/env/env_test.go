package env

import "testing"

func TestResolveDefaults(t *testing.T) {
	e := Resolve("/h", func(string) string { return "" })
	want := Env{Home: "/h", ConfigFile: "/h/.config/hoard/config.env", DataDir: "/h/.local/share/hoard", StateDir: "/h/.local/state/hoard", ClaudeDir: "/h/.claude", CodexDir: "/h/.codex", HermesHome: "/h/.hermes"}
	if e != want {
		t.Fatalf("got %+v", e)
	}
}

func TestResolveOverrides(t *testing.T) {
	env := map[string]string{"XDG_CONFIG_HOME": "/c", "XDG_DATA_HOME": "/d", "XDG_STATE_HOME": "/s", "CLAUDE_CONFIG_DIR": "/cl", "CODEX_HOME": "/co", "HERMES_HOME": "/he"}
	e := Resolve("/h", func(k string) string { return env[k] })
	if e.ConfigFile != "/c/hoard/config.env" || e.DataDir != "/d/hoard" || e.StateDir != "/s/hoard" || e.ClaudeDir != "/cl" || e.CodexDir != "/co" || e.HermesHome != "/he" {
		t.Fatalf("got %+v", e)
	}
	if e.Hook("recall.sh") != "/d/hoard/hooks/recall.sh" {
		t.Fatal(e.Hook("recall.sh"))
	}
}

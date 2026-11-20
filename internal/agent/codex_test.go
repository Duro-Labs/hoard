package agent

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexInstallUninstall(t *testing.T) {
	e := testEnv(t)
	hp := filepath.Join(e.CodexDir, "hooks.json")
	write(t, hp, `{"hooks":{"PreToolUse":[{"matcher":"apply_patch","hooks":[{"type":"command","command":"/u/g.sh","timeout":10}]}]}}`)
	write(t, filepath.Join(e.CodexDir, "AGENTS.md"), "mine\n")
	c := Codex{}
	if err := c.Install(e, "t1"); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	json.Unmarshal([]byte(read(t, hp)), &doc)
	if !jsonHasCommand(doc, "PreToolUse", "/u/g.sh") || !jsonHasCommand(doc, "UserPromptSubmit", e.Hook("recall.sh")) || !jsonHasCommand(doc, "PostToolUse", e.Hook("after-write.sh")) {
		t.Fatalf("hooks.json: %s", read(t, hp))
	}
	agents := read(t, filepath.Join(e.CodexDir, "AGENTS.md"))
	if !strings.Contains(agents, "# HOARD") || !strings.HasPrefix(agents, "mine\n") {
		t.Fatalf("AGENTS.md: %q", agents)
	}
	if err := c.Uninstall(e, "t2"); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(e.CodexDir, "AGENTS.md")) != "mine\n" {
		t.Fatal("AGENTS.md not restored")
	}
	json.Unmarshal([]byte(read(t, hp)), &doc)
	if jsonHasCommand(doc, "UserPromptSubmit", e.Hook("recall.sh")) {
		t.Fatal("hook not removed")
	}
}

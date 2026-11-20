package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const userSettings = `{"model":"x","hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"/u/guard.sh"}]}]}}`

func TestClaudeInstallMergesAndIsIdempotent(t *testing.T) {
	e := testEnv(t)
	sp := filepath.Join(e.ClaudeDir, "settings.json")
	write(t, sp, userSettings)
	write(t, filepath.Join(e.ClaudeDir, "CLAUDE.md"), "my rules\n")
	c := Claude{}
	if err := c.Install(e, "t1"); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	json.Unmarshal([]byte(read(t, sp)), &doc)
	if doc["model"] != "x" || !jsonHasCommand(doc, "PreToolUse", "/u/guard.sh") {
		t.Fatal("user settings lost")
	}
	if !jsonHasCommand(doc, "UserPromptSubmit", e.Hook("recall.sh")) || !jsonHasCommand(doc, "PostToolUse", e.Hook("after-write.sh")) {
		t.Fatal("hoard hooks missing")
	}
	md := read(t, filepath.Join(e.ClaudeDir, "CLAUDE.md"))
	if !strings.HasPrefix(md, "my rules\n") || !strings.Contains(md, "@"+filepath.Join(e.DataDir, "rules", "HOARD.md")) {
		t.Fatalf("CLAUDE.md: %q", md)
	}
	if _, err := os.Stat(filepath.Join(e.ClaudeDir, "skills", "hoard-tidy", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if n := len(backups(t, e.ClaudeDir)); n != 2 {
		t.Fatalf("want 2 backups, got %d", n)
	}
	before := read(t, sp)
	if err := c.Install(e, "t2"); err != nil {
		t.Fatal(err)
	}
	if read(t, sp) != before || len(backups(t, e.ClaudeDir)) != 2 {
		t.Fatal("second install changed something")
	}
}

func TestClaudeUninstallRestores(t *testing.T) {
	e := testEnv(t)
	sp := filepath.Join(e.ClaudeDir, "settings.json")
	write(t, sp, userSettings)
	write(t, filepath.Join(e.ClaudeDir, "CLAUDE.md"), "my rules\n")
	c := Claude{}
	c.Install(e, "t1")
	if err := c.Uninstall(e, "t2"); err != nil {
		t.Fatal(err)
	}
	var got, want map[string]any
	json.Unmarshal([]byte(read(t, sp)), &got)
	json.Unmarshal([]byte(userSettings), &want)
	gb, _ := json.Marshal(got)
	wb, _ := json.Marshal(want)
	if string(gb) != string(wb) {
		t.Fatalf("settings not restored: %s", gb)
	}
	if read(t, filepath.Join(e.ClaudeDir, "CLAUDE.md")) != "my rules\n" {
		t.Fatal("CLAUDE.md not restored")
	}
	if _, err := os.Stat(filepath.Join(e.ClaudeDir, "skills", "hoard-tidy")); !os.IsNotExist(err) {
		t.Fatal("skill not removed")
	}
}

func TestClaudeInstallIntoEmptyDir(t *testing.T) {
	e := testEnv(t)
	if err := (Claude{}).Install(e, "t1"); err != nil {
		t.Fatal(err)
	}
	if fs := (Claude{}).Check(e); fs[0].Level != "ok" {
		t.Fatalf("%+v", fs)
	}
}

func TestSkillRefusesUnmanagedDir(t *testing.T) {
	e := testEnv(t)
	write(t, filepath.Join(e.ClaudeDir, "skills", "hoard-tidy", "SKILL.md"), "mine")
	if err := (Claude{}).Install(e, "t1"); err == nil {
		t.Fatal("expected refusal")
	}
}

func TestBlocks(t *testing.T) {
	for _, orig := range []string{"", "a\n"} {
		in := upsertBlock(orig, "x")
		if !hasBlock(in) || upsertBlock(in, "x") != in {
			t.Fatalf("upsert %q -> %q", orig, in)
		}
		if got := removeBlock(in); got != orig {
			t.Fatalf("remove: %q want %q", got, orig)
		}
	}
	if !strings.Contains(upsertBlock(upsertBlock("", "old"), "new"), "new") {
		t.Fatal("block not refreshed")
	}
}

func TestRemoveJSONHooksPreservesNonStringCommand(t *testing.T) {
	doc := map[string]any{"hooks": map[string]any{"PostToolUse": []any{map[string]any{"hooks": []any{map[string]any{"command": 42}}}}}}
	removeJSONHooks(doc, []string{"/hoard/hook.sh"})
	if got := groupsOf(doc, "PostToolUse"); len(got) != 1 {
		t.Fatalf("unexpected groups: %#v", got)
	}
}

package agent

import (
	"path/filepath"
	"strings"
	"testing"
)

const hermesCfg = `# my config
model: x
hooks:
  pre_tool_call:
    - {matcher: terminal, command: /u/g.sh, timeout: 10, fail_closed: true}
`

func TestHermesInstallUninstall(t *testing.T) {
	e := testEnv(t)
	cp := filepath.Join(e.HermesHome, "config.yaml")
	write(t, cp, hermesCfg)
	h := Hermes{}
	if err := h.Install(e, "t1"); err != nil {
		t.Fatal(err)
	}
	got := read(t, cp)
	for _, want := range []string{"# my config", "/u/g.sh", "pre_llm_call", e.Hook("recall.sh"), "post_tool_call", "write_file|patch", e.Hook("after-write.sh")} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if err := h.Install(e, "t2"); err != nil || read(t, cp) != got {
		t.Fatal("second install changed config")
	}
	if !strings.Contains(read(t, filepath.Join(e.HermesHome, "SOUL.md")), "# HOARD") {
		t.Fatal("SOUL.md missing rules")
	}
	if err := h.Uninstall(e, "t3"); err != nil {
		t.Fatal(err)
	}
	after := read(t, cp)
	if strings.Contains(after, "pre_llm_call") || strings.Contains(after, "post_tool_call") || !strings.Contains(after, "/u/g.sh") || !strings.Contains(after, "# my config") {
		t.Fatalf("uninstall result:\n%s", after)
	}
}

func TestHermesInstallNoConfig(t *testing.T) {
	e := testEnv(t)
	if err := (Hermes{}).Install(e, "t1"); err != nil {
		t.Fatal(err)
	}
	if fs := (Hermes{}).Check(e); fs[0].Level != "ok" {
		t.Fatalf("%+v", fs)
	}
}

func TestRemoveYAMLHooksKeepsExistingEmptyEvent(t *testing.T) {
	in := []byte("hooks:\n  pre_tool_call: []\n  pre_llm_call:\n    - command: /hoard/recall.sh\n")
	out, err := removeYAMLHooks(in, []string{"/hoard/recall.sh"})
	if err != nil || !strings.Contains(string(out), "pre_tool_call: []") {
		t.Fatalf("got %s, %v", out, err)
	}
}

package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Duro-Labs/hoard/internal/env"
)

type Finding struct{ Level, Msg string }

type Agent interface {
	Name() string
	Install(e env.Env, stamp string) error
	Uninstall(e env.Env, stamp string) error
	Check(e env.Env) []Finding
}

type hookSpec struct {
	Event, Matcher, Command string
	Timeout                 int
}

const hookTimeout = 10

func All() []Agent { return []Agent{Claude{}, Codex{}} }

func ByName(name string) (Agent, error) {
	for _, a := range All() {
		if a.Name() == name {
			return a, nil
		}
	}
	return nil, fmt.Errorf("unknown agent %q (want claude, codex or hermes)", name)
}

func commands(specs []hookSpec) []string {
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Command
	}
	return out
}
func checkCommon(name, cfg, rules, skill string, load func() (map[string]any, error), specs []hookSpec, notes []Finding) []Finding {
	var out []Finding
	doc, err := load()
	missing := err != nil
	for _, s := range specs {
		if missing || !jsonHasCommand(doc, s.Event, s.Command) {
			missing = true
		}
	}
	if missing {
		out = append(out, Finding{"fail", name + ": hooks not wired in " + cfg + " (run hoard install --agent " + name + ")"})
	} else {
		out = append(out, Finding{"ok", name + ": hooks wired in " + cfg})
	}
	if b, err := os.ReadFile(rules); err != nil || !hasBlock(string(b)) {
		out = append(out, Finding{"warn", name + ": rules block missing from " + rules})
	}
	if _, err := os.Stat(filepath.Join(skill, skillMarker)); err != nil {
		out = append(out, Finding{"warn", name + ": hoard-tidy skill missing at " + skill})
	}
	return append(out, notes...)
}

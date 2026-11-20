package agent

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Duro-Labs/hoard/internal/env"
)

type Hermes struct{}

func (Hermes) Name() string { return "hermes" }

func (Hermes) specs(e env.Env) []hookSpec {
	return []hookSpec{
		{Event: "pre_llm_call", Command: e.Hook("recall.sh"), Timeout: hookTimeout},
		{Event: "post_tool_call", Matcher: "write_file|patch", Command: e.Hook("after-write.sh"), Timeout: hookTimeout},
	}
}

func (Hermes) paths(e env.Env) (cfg, rules, skill string) {
	return filepath.Join(e.HermesHome, "config.yaml"), filepath.Join(e.HermesHome, "SOUL.md"), filepath.Join(e.HermesHome, "skills", "hoard", "hoard-tidy")
}

func editYAML(path, stamp string, fn func([]byte) ([]byte, error)) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	out, err := fn(b)
	if err != nil {
		return err
	}
	return writeIfChanged(path, out, stamp)
}

func (h Hermes) Install(e env.Env, stamp string) error {
	cfg, rules, skill := h.paths(e)
	body, err := rulesText(e)
	if err != nil {
		return err
	}
	if err := installSkill(e, skill); err != nil {
		return err
	}
	if err := editYAML(cfg, stamp, func(b []byte) ([]byte, error) { return mergeYAMLHooks(b, h.specs(e)) }); err != nil {
		return err
	}
	return editText(rules, stamp, func(t string) string { return upsertBlock(t, body) })
}

func (h Hermes) Uninstall(e env.Env, stamp string) error {
	cfg, rules, skill := h.paths(e)
	if err := editYAML(cfg, stamp, func(b []byte) ([]byte, error) { return removeYAMLHooks(b, commands(h.specs(e))) }); err != nil {
		return err
	}
	if err := editText(rules, stamp, removeBlock); err != nil {
		return err
	}
	return removeSkill(skill)
}

func (h Hermes) Check(e env.Env) []Finding {
	cfg, rules, skill := h.paths(e)
	return checkCommon(h.Name(), cfg, rules, skill, func() (map[string]any, error) {
		b, err := os.ReadFile(cfg)
		if err != nil {
			return nil, err
		}
		return yamlHooksAsJSON(b)
	}, h.specs(e), []Finding{
		{"warn", "hermes: shell hooks need consent once; start hermes with --accept-hooks or approve the prompt"},
		{"warn", "hermes: post_tool_call output is ignored, so mirror and nudge notes never reach the agent (copies still happen)"},
	})
}

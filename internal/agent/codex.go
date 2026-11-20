package agent

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Duro-Labs/hoard/internal/env"
)

type Codex struct{}

func (Codex) Name() string { return "codex" }

func (Codex) specs(e env.Env) []hookSpec {
	return []hookSpec{
		{Event: "UserPromptSubmit", Command: e.Hook("recall.sh"), Timeout: hookTimeout},
		{Event: "PostToolUse", Matcher: "apply_patch", Command: e.Hook("after-write.sh"), Timeout: hookTimeout},
	}
}

func (Codex) paths(e env.Env) (hooks, rules, skill string) {
	return filepath.Join(e.CodexDir, "hooks.json"), filepath.Join(e.CodexDir, "AGENTS.md"), filepath.Join(e.CodexDir, "skills", "hoard-tidy")
}

func rulesText(e env.Env) (string, error) {
	b, err := os.ReadFile(filepath.Join(e.DataDir, "rules", "HOARD.md"))
	return string(b), err
}

func (c Codex) Install(e env.Env, stamp string) error {
	hooks, rules, skill := c.paths(e)
	body, err := rulesText(e)
	if err != nil {
		return err
	}
	if err := installSkill(e, skill); err != nil {
		return err
	}
	if err := editJSON(hooks, stamp, func(d map[string]any) { mergeJSONHooks(d, c.specs(e)) }); err != nil {
		return err
	}
	return editText(rules, stamp, func(t string) string { return upsertBlock(t, body) })
}

func (c Codex) Uninstall(e env.Env, stamp string) error {
	hooks, rules, skill := c.paths(e)
	if err := editJSON(hooks, stamp, func(d map[string]any) { removeJSONHooks(d, commands(c.specs(e))) }); err != nil {
		return err
	}
	if err := editText(rules, stamp, removeBlock); err != nil {
		return err
	}
	return removeSkill(skill)
}

func (c Codex) Check(e env.Env) []Finding {
	hooks, rules, skill := c.paths(e)
	return checkCommon(c.Name(), hooks, rules, skill, func() (map[string]any, error) {
		var d map[string]any
		b, err := os.ReadFile(hooks)
		if err == nil {
			err = json.Unmarshal(b, &d)
		}
		return d, err
	}, c.specs(e), []Finding{{"warn", "codex: new hooks need trusting once; Codex prompts on the next session start"}})
}

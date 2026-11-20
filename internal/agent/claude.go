package agent

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Duro-Labs/hoard/internal/env"
)

type Claude struct{}

func (Claude) Name() string { return "claude" }

func (Claude) specs(e env.Env) []hookSpec {
	return []hookSpec{
		{Event: "UserPromptSubmit", Command: e.Hook("recall.sh"), Timeout: hookTimeout},
		{Event: "PostToolUse", Matcher: "Write|Edit", Command: e.Hook("after-write.sh"), Timeout: hookTimeout},
	}
}

func (Claude) paths(e env.Env) (settings, rules, skill string) {
	return filepath.Join(e.ClaudeDir, "settings.json"), filepath.Join(e.ClaudeDir, "CLAUDE.md"), filepath.Join(e.ClaudeDir, "skills", "hoard-tidy")
}

func (c Claude) Install(e env.Env, stamp string) error {
	settings, rules, skill := c.paths(e)
	if err := installSkill(e, skill); err != nil {
		return err
	}
	if err := editJSON(settings, stamp, func(d map[string]any) { mergeJSONHooks(d, c.specs(e)) }); err != nil {
		return err
	}
	ref := "@" + filepath.Join(e.DataDir, "rules", "HOARD.md")
	return editText(rules, stamp, func(t string) string { return upsertBlock(t, ref) })
}

func (c Claude) Uninstall(e env.Env, stamp string) error {
	settings, rules, skill := c.paths(e)
	if err := editJSON(settings, stamp, func(d map[string]any) { removeJSONHooks(d, commands(c.specs(e))) }); err != nil {
		return err
	}
	if err := editText(rules, stamp, removeBlock); err != nil {
		return err
	}
	return removeSkill(skill)
}

func (c Claude) Check(e env.Env) []Finding {
	settings, rules, skill := c.paths(e)
	return checkCommon(c.Name(), settings, rules, skill, func() (map[string]any, error) {
		var d map[string]any
		b, err := os.ReadFile(settings)
		if err == nil {
			err = json.Unmarshal(b, &d)
		}
		return d, err
	}, c.specs(e), nil)
}

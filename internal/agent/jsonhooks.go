package agent

func groupsOf(doc map[string]any, event string) []any {
	hooks, _ := doc["hooks"].(map[string]any)
	g, _ := hooks[event].([]any)
	return g
}

func jsonHasCommand(doc map[string]any, event, command string) bool {
	for _, g := range groupsOf(doc, event) {
		gm, _ := g.(map[string]any)
		hs, _ := gm["hooks"].([]any)
		for _, h := range hs {
			if hm, _ := h.(map[string]any); hm["command"] == command {
				return true
			}
		}
	}
	return false
}

func mergeJSONHooks(doc map[string]any, specs []hookSpec) {
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		doc["hooks"] = hooks
	}
	for _, s := range specs {
		if jsonHasCommand(doc, s.Event, s.Command) {
			continue
		}
		g := map[string]any{"hooks": []any{map[string]any{"type": "command", "command": s.Command, "timeout": s.Timeout}}}
		if s.Matcher != "" {
			g["matcher"] = s.Matcher
		}
		hooks[s.Event] = append(groupsOf(doc, s.Event), g)
	}
}

func removeJSONHooks(doc map[string]any, commands []string) {
	drop := map[string]bool{}
	for _, c := range commands {
		drop[c] = true
	}
	hooks, _ := doc["hooks"].(map[string]any)
	for event := range hooks {
		var keptGroups []any
		for _, g := range groupsOf(doc, event) {
			gm, _ := g.(map[string]any)
			hs, _ := gm["hooks"].([]any)
			var kept []any
			for _, h := range hs {
				hm, _ := h.(map[string]any)
				command, _ := hm["command"].(string)
				if !drop[command] {
					kept = append(kept, h)
				}
			}
			if len(kept) > 0 {
				gm["hooks"] = kept
				keptGroups = append(keptGroups, gm)
			}
		}
		if len(keptGroups) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = keptGroups
		}
	}
	if hooks != nil && len(hooks) == 0 {
		delete(doc, "hooks")
	}
}

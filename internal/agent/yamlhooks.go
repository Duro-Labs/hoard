package agent

import (
	"bytes"
	"strconv"

	"gopkg.in/yaml.v3"
)

func parseYAML(src []byte) (*yaml.Node, *yaml.Node, error) {
	var doc yaml.Node
	if len(bytes.TrimSpace(src)) > 0 {
		if err := yaml.Unmarshal(src, &doc); err != nil {
			return nil, nil, err
		}
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	return &doc, doc.Content[0], nil
}

func mapGet(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func mapDel(m *yaml.Node, key string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}

func mapAdd(m *yaml.Node, key string, v *yaml.Node) {
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, v)
}

func scalar(v string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Value: v} }

func encodeYAML(doc *yaml.Node) ([]byte, error) {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return b.Bytes(), enc.Close()
}

func seqHasCommand(seq *yaml.Node, command string) bool {
	for _, it := range seq.Content {
		if c := mapGet(it, "command"); c != nil && c.Value == command {
			return true
		}
	}
	return false
}

func mergeYAMLHooks(src []byte, specs []hookSpec) ([]byte, error) {
	doc, root, err := parseYAML(src)
	if err != nil {
		return nil, err
	}
	hooks := mapGet(root, "hooks")
	if hooks == nil || hooks.Kind != yaml.MappingNode {
		mapDel(root, "hooks")
		hooks = &yaml.Node{Kind: yaml.MappingNode}
		mapAdd(root, "hooks", hooks)
	}
	changed := false
	for _, s := range specs {
		seq := mapGet(hooks, s.Event)
		if seq == nil || seq.Kind != yaml.SequenceNode {
			mapDel(hooks, s.Event)
			seq = &yaml.Node{Kind: yaml.SequenceNode}
			mapAdd(hooks, s.Event, seq)
		}
		if seqHasCommand(seq, s.Command) {
			continue
		}
		item := &yaml.Node{Kind: yaml.MappingNode}
		if s.Matcher != "" {
			mapAdd(item, "matcher", scalar(s.Matcher))
		}
		mapAdd(item, "command", scalar(s.Command))
		mapAdd(item, "timeout", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(s.Timeout)})
		seq.Content = append(seq.Content, item)
		changed = true
	}
	if !changed {
		return src, nil
	}
	return encodeYAML(doc)
}

func removeYAMLHooks(src []byte, cmds []string) ([]byte, error) {
	doc, root, err := parseYAML(src)
	if err != nil {
		return nil, err
	}
	hooks := mapGet(root, "hooks")
	if hooks == nil || hooks.Kind != yaml.MappingNode {
		return src, nil
	}
	drop := map[string]bool{}
	for _, c := range cmds {
		drop[c] = true
	}
	changed := false
	for i := 0; i+1 < len(hooks.Content); {
		seq := hooks.Content[i+1]
		if seq.Kind == yaml.SequenceNode {
			var kept []*yaml.Node
			removed := false
			for _, it := range seq.Content {
				if c := mapGet(it, "command"); c != nil && drop[c.Value] {
					changed = true
					removed = true
					continue
				}
				kept = append(kept, it)
			}
			seq.Content = kept
			if removed && len(kept) == 0 {
				hooks.Content = append(hooks.Content[:i], hooks.Content[i+2:]...)
				continue
			}
		}
		i += 2
	}
	if len(hooks.Content) == 0 {
		mapDel(root, "hooks")
	}
	if !changed {
		return src, nil
	}
	return encodeYAML(doc)
}

func yamlHooksAsJSON(src []byte) (map[string]any, error) {
	_, root, err := parseYAML(src)
	if err != nil {
		return nil, err
	}
	events := map[string]any{}
	if hooks := mapGet(root, "hooks"); hooks != nil && hooks.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(hooks.Content); i += 2 {
			var hs []any
			for _, it := range hooks.Content[i+1].Content {
				if c := mapGet(it, "command"); c != nil {
					hs = append(hs, map[string]any{"command": c.Value})
				}
			}
			events[hooks.Content[i].Value] = []any{map[string]any{"hooks": hs}}
		}
	}
	return map[string]any{"hooks": events}, nil
}

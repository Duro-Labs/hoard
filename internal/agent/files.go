package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	blockBegin = "<!-- hoard:begin -->"
	blockEnd   = "<!-- hoard:end -->"
)

func writeIfChanged(path string, b []byte, stamp string) error {
	old, err := os.ReadFile(path)
	switch {
	case err == nil:
		if bytes.Equal(old, b) {
			return nil
		}
		if err := os.WriteFile(path+".hoard-bak-"+stamp, old, 0o600); err != nil {
			return err
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func editJSON(path, stamp string, fn func(map[string]any)) error {
	doc := map[string]any{}
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if len(bytes.TrimSpace(b)) > 0 {
		if err := json.Unmarshal(b, &doc); err != nil {
			return err
		}
	}
	before, _ := json.Marshal(doc)
	fn(doc)
	after, _ := json.Marshal(doc)
	if len(b) > 0 && bytes.Equal(before, after) {
		return nil
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return writeIfChanged(path, append(out, '\n'), stamp)
}

func editText(path, stamp string, fn func(string) string) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	out := fn(string(b))
	if out == string(b) {
		return nil
	}
	return writeIfChanged(path, []byte(out), stamp)
}

func blockSpan(text string) (int, int, bool) {
	i := strings.Index(text, blockBegin)
	if i < 0 {
		return 0, 0, false
	}
	j := strings.Index(text[i:], blockEnd)
	if j < 0 {
		return 0, 0, false
	}
	k := i + j + len(blockEnd)
	if k < len(text) && text[k] == '\n' {
		k++
	}
	return i, k, true
}

func hasBlock(text string) bool { _, _, ok := blockSpan(text); return ok }

func upsertBlock(text, body string) string {
	block := blockBegin + "\n" + strings.TrimRight(body, "\n") + "\n" + blockEnd + "\n"
	if i, k, ok := blockSpan(text); ok {
		return text[:i] + block + text[k:]
	}
	if text == "" {
		return block
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text + "\n" + block
}

func removeBlock(text string) string {
	i, k, ok := blockSpan(text)
	if !ok {
		return text
	}
	before, after := text[:i], text[k:]
	if after == "" && strings.HasSuffix(before, "\n\n") {
		before = before[:len(before)-1]
	}
	return before + after
}

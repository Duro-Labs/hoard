package assets

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestExtract(t *testing.T) {
	src := fstest.MapFS{"hooks/_lib.sh": {Data: []byte("x")}, "rules/HOARD.md": {Data: []byte("r")}}
	dst := t.TempDir()
	if err := Extract(src, dst); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(filepath.Join(dst, "hooks", "_lib.sh"))
	if err != nil || st.Mode().Perm() != 0o755 {
		t.Fatalf("lib: %v %v", st, err)
	}
	st, err = os.Stat(filepath.Join(dst, "rules", "HOARD.md"))
	if err != nil || st.Mode().Perm() != 0o644 {
		t.Fatalf("rules: %v %v", st, err)
	}
}

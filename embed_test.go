package hoard

import (
	"io/fs"
	"testing"
)

func TestAssetsIncludeUnderscoreFiles(t *testing.T) {
	for _, p := range []string{"hooks/_lib.sh", "hooks/recall.sh", "hooks/after-write.sh", "rules/HOARD.md", "skills/tidy/SKILL.md", "skills/tidy/scripts/scan.sh"} {
		if _, err := fs.Stat(Assets, p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

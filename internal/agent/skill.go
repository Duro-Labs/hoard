package agent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Duro-Labs/hoard/internal/assets"
	"github.com/Duro-Labs/hoard/internal/env"
)

const skillMarker = ".hoard-managed"

func installSkill(e env.Env, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		if _, err := os.Stat(filepath.Join(dest, skillMarker)); err != nil {
			return fmt.Errorf("%s exists and was not installed by hoard; move it away first", dest)
		}
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := assets.Extract(os.DirFS(filepath.Join(e.DataDir, "skills", "tidy")), dest); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, skillMarker), nil, 0o644)
}

func removeSkill(dest string) error {
	if _, err := os.Stat(filepath.Join(dest, skillMarker)); err != nil {
		return nil
	}
	return os.RemoveAll(dest)
}

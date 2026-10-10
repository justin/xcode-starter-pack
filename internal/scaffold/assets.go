package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
)

// assets contains the build-configuration and support files that every
// generated project receives. Keeping these files beside the scaffolder makes
// the released binary self-contained.
//
//go:embed all:assets
var assets embed.FS

func writeBaseAssets(outputPath string) error {
	base, err := fs.Sub(assets, "assets/base")
	if err != nil {
		return fmt.Errorf("load starter assets: %w", err)
	}

	return fs.WalkDir(base, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) == ".DS_Store" {
			return nil
		}

		contents, err := fs.ReadFile(base, path)
		if err != nil {
			return err
		}
		return writeBytes(filepath.Join(outputPath, filepath.FromSlash(path)), contents, 0o644)
	})
}

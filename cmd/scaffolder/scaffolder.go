package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io/fs"
	"log"
	"os"
)

//go:embed template.zip
var template []byte

func main() {
	zfs, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		log.Fatal(err)
	}
	tfs, err := fs.Sub(zfs, "template")
	if err != nil {
		log.Fatal(err)
	}

	if err := fs.WalkDir(tfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil { //nolint:gosec
				return fmt.Errorf("scaffolder: creating directory: %w", err)
			}
			return nil
		}
		content, err := fs.ReadFile(tfs, path)
		if err != nil {
			return fmt.Errorf("scaffolder: reading embedded file: %w", err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil { //nolint:gosec
			return fmt.Errorf("scaffolder: writing file: %w", err)
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}

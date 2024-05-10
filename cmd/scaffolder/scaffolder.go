package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
)

//go:embed template
var template embed.FS

func main() {
	tfs, err := fs.Sub(template, "template")
	if err != nil {
		log.Fatal(err)
	}

	fs.WalkDir(tfs, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			return nil
		}
		content, err := fs.ReadFile(tfs, path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return err
		}
		return nil
	})
}

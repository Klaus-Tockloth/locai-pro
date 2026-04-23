package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed locai-pro.yaml
var locaiProYaml []byte

/*
writeConfig writes the embedded default configuration to file named 'locai-pro.yaml'.
*/
func writeConfig() error {
	filename := "locai-pro.yaml"
	err := os.WriteFile(filename, locaiProYaml, 0600)
	if err != nil {
		return fmt.Errorf("embed: error [%v] at os.WriteFile(), file = [%s]", err, filename)
	}
	return nil
}

//go:embed prompt-input.html
var locAIPromptInputHTML []byte

/*
writePromptInput writes the embedded HTML content for prompt input (locAIPromptInputHTML) to a file.
*/
func writePromptInput() error {
	filename := "prompt-input.html"
	err := os.WriteFile(filename, locAIPromptInputHTML, 0600)
	if err != nil {
		return fmt.Errorf("embed: error [%v] at os.WriteFile(), file = [%s]", err, filename)
	}
	return nil
}

//go:embed README.md
var readmeBytes []byte

/*
writeReadme writes the embedded README.md to the current directory.
*/
func writeReadme() error {
	filename := "README.md"
	err := os.WriteFile(filename, readmeBytes, 0600)
	if err != nil {
		return fmt.Errorf("embed: error [%v] at os.WriteFile(), file = [%s]", err, filename)
	}
	return nil
}

//go:embed locai-pro.png
var locaiProPngBytes []byte

/*
writeLocAIProPng writes the embedded locai-pro.png to the current directory.
*/
func writeLocAIProPng() error {
	filename := "locai-pro.png"
	err := os.WriteFile(filename, locaiProPngBytes, 0600)
	if err != nil {
		return fmt.Errorf("embed: error [%v] at os.WriteFile(), file = [%s]", err, filename)
	}
	return nil
}

//go:embed assets
var assetsFS embed.FS

/*
writeAssets writes all embedded files from the 'assets/' directory
to the provided base path.
*/
func writeAssets(basepath string) error {
	err := fs.WalkDir(assetsFS, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(basepath, path)

		if d.IsDir() {
			err := os.MkdirAll(targetPath, 0700)
			if err != nil {
				return err
			}
		} else {
			content, err := assetsFS.ReadFile(path)
			if err != nil {
				return err
			}

			err = os.WriteFile(targetPath, content, 0600)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("embed: error [%v] at writeAssets() exploring assetsFS", err)
	}
	return nil
}

//go:embed system-instruction.txt
var systemInstructionTxtBytes []byte

/*
writeSystemInstruction writes the embedded system-instruction.txt to the current directory.
*/
func writeSystemInstruction() error {
	filename := "system-instruction.txt"
	err := os.WriteFile(filename, systemInstructionTxtBytes, 0600)
	if err != nil {
		return fmt.Errorf("embed: error [%v] at os.WriteFile(), file = [%s]", err, filename)
	}
	return nil
}

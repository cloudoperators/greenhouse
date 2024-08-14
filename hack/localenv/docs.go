//go:build docs

package main

import (
	"github.com/cloudoperators/greenhouse/hack/localenv/cmd"
	"log"
	"os"

	"github.com/spf13/cobra/doc"
)

func createDirIfNotExist(dir string) string {
	// Check if the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Create the directory
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Fatalf("error creating directory: %v", err)
		}
	}
	return dir
}

func main() {
	commands := cmd.GetCommands()
	for _, c := range commands {
		err := doc.GenMarkdownTree(c, createDirIfNotExist("./docs"))
		if err != nil {
			log.Printf("error generating markdown tree: %v", err)
		}
	}
}

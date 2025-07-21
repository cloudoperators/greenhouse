// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build dev

package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/cobra/doc"

	"github.com/cloudoperators/greenhouse/internal/cmd"
)

var removeLinks = regexp.MustCompile(`(?s)### SEE ALSO.*`)
var removeOptsInherited = regexp.MustCompile(`(?s)### Options inherited from parent commands.*`)

// docsTemplateData - data for the docs template
// add more fields as needed
type docsTemplateData struct {
	Intro    string // intro markdown content
	Commands string // cobra generated command markdown content
	DocGen   string // doc gen markdown content
}

// extend the template for future additions
const docsTemplate = `{{.Intro}}
{{.Commands}}
{{.DocGen}}
`

func getDocsDir() string {
	// Determine the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %s", err.Error())
	}

	// Check if the current working directory is hack/localenv
	var outputPath string
	if filepath.Base(filepath.Dir(cwd)) == "dev-env" {
		outputPath = cwd
	} else {
		outputPath = filepath.Join(cwd, "dev-env")
	}
	return outputPath
}

// getDevDocsIntro - get the intro markdown content from dev-env/templates/_intro.md
func getTemplate(cwd string, template string) ([]byte, error) {
	return os.ReadFile(filepath.Join(cwd, "templates", template))
}

func stitchMarkdown(data docsTemplateData) ([]byte, error) {
	t := template.Must(template.New("docs").Parse(docsTemplate))
	var output bytes.Buffer
	err := t.Execute(&output, data)
	if err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// auto generate dev commands documentation in greenhousectl
func main() {
	commands := cmd.GenerateDevDocs()
	docs := make([]string, 0, len(commands))
	for _, command := range commands {
		buf := new(bytes.Buffer)
		err := doc.GenMarkdownCustom(command, buf, func(s string) string { return "" })
		if err != nil {
			log.Fatalf("Error generating command docs: %s", err.Error())
		}
		if buf.Len() > 0 {
			content := buf.String()
			content = removeLinks.ReplaceAllString(content, "")
			content = removeOptsInherited.ReplaceAllString(content, "")
			docs = append(docs, content)
		}
	}
	outputPath := "./docs/contribute"
	docsDir := getDocsDir()
	intro, err := getTemplate(docsDir, "_intro.md")
	if err != nil {
		log.Fatalf("error getting intro: %s", err.Error())
	}
	docGen, err := getTemplate(docsDir, "_generate-docs.md")
	if err != nil {
		log.Fatalf("error getting doc gen: %s", err.Error())
	}
	docData := docsTemplateData{
		Intro:    string(intro),
		Commands: strings.Join(docs, ""),
		DocGen:   string(docGen),
	}
	markdown, err := stitchMarkdown(docData)
	if err != nil {
		log.Fatalf("Error generating markdown: %s", err.Error())
	}
	err = os.WriteFile(filepath.Join(outputPath, "local-dev.md"), markdown, 0644)
	if err != nil {
		log.Fatalf("Error writing docs: %s", err.Error())
	}
}

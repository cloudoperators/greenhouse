// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/template"

	flag "github.com/spf13/pflag"
)

var (
	GitBranch,
	GitCommit,
	GitState,
	BuildDate string
	GoVersion = runtime.Version()

	versionRequested = false

	versionInfoTmpl = `
{{.program}}, revision: {{.revision}}, branch: {{.branch}}, state: {{.state}}
  build date:       {{.buildDate}}
  go version:       {{.goVersion}}
`
)

func init() {
	flag.BoolVar(&versionRequested, "version", false, "Display version and exit")
}

func ShowVersionAndExit(programName string) {
	if versionRequested {
		fmt.Println(strings.TrimSpace(GetVersionTemplate(programName)))
		os.Exit(0)
	}
}

func GetVersionTemplate(programName string) string {
	m := map[string]string{
		"program":   programName,
		"revision":  GitCommit,
		"state":     GitState,
		"branch":    GitBranch,
		"buildDate": BuildDate,
		"goVersion": GoVersion,
	}
	t := template.Must(template.New("version").Parse(versionInfoTmpl))
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "version", m); err != nil {
		panic(err)
	}
	return buf.String()
}

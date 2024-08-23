// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"
	"fmt"
	"github.com/vladimirvivien/gexe"
	"io"
	"os"
	"strings"
)

type ShellPipe struct {
	Shells []Shell
}

type Shell struct {
	Cmd  string
	Vars map[string]string
}

// Exec executes a set of commands serially and pipes the output of each command to the next
func (s ShellPipe) Exec() error {
	exec := gexe.New()
	if len(s.Shells) == 0 {
		return errors.New("empty commands")
	}
	if len(s.Shells) == 1 {
		return errors.New("too few commands to pipe")
	}
	commands := make([]string, 0)
	for _, shell := range s.Shells {
		if strings.TrimSpace(shell.Cmd) == "" {
			return errors.New("empty command found")
		}
		for k, v := range shell.Vars {
			exec.SetVar(k, v)
		}
		commands = append(commands, shell.Cmd)
	}
	pipe := exec.Commands(commands...).Pipe()
	errs := make([]string, 0)
	for _, p := range pipe.Procs() {
		if err := p.Err(); err != nil {
			errs = append(errs, err.Error())
		}
		out, _ := io.ReadAll(p.Out())
		if strings.TrimSpace(string(out)) != "" {
			Log(string(out))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("error executing command: %s", strings.Join(errs, "\n"))
	}
	return nil
}

// ExecWithResult executes the shell command and returns the output of the command
func (s Shell) ExecWithResult() (string, error) {
	exec := gexe.New()
	setVars(exec, s.Vars)
	if err := s.checkEmptyCommand(); err != nil {
		return "", err
	}
	proc := exec.RunProc(s.Cmd)
	if err := proc.Err(); err != nil {
		return "", err
	}
	return proc.Result(), nil
}

func (s Shell) checkEmptyCommand() error {
	if strings.TrimSpace(s.Cmd) == "" {
		return errors.New("empty command")
	}
	return nil
}

func setVars(exec *gexe.Echo, vars map[string]string) {
	for k, v := range vars {
		exec.SetVar(k, v)
	}
}

// Exec executes a single shell command
func (s Shell) Exec() error {
	exec := gexe.New()
	setVars(exec, s.Vars)
	if err := s.checkEmptyCommand(); err != nil {
		return err
	}
	proc := exec.NewProc(s.Cmd)
	proc.SetStdout(os.Stdout)
	proc.SetStderr(os.Stderr)
	if err := proc.Run().Err(); err != nil {
		LogErr("error running command: %s", s.Cmd)
		return err
	}
	return nil
}

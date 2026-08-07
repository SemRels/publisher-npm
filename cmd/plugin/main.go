// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The publisher-npm Authors

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	plugin "github.com/SemRels/publisher-npm/internal/plugin"
)

const pluginSchemaVersion = 1

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "plugin_schema_version=%d\n", pluginSchemaVersion)
		_, _ = fmt.Fprintf(os.Stderr, "publisher-npm: resolve working directory: %v\n", err)
		os.Exit(1)
	}

	os.Exit(run(context.Background(), os.Stdout, os.Stderr, os.Getenv, cwd, exec.LookPath, plugin.YarnVersion, os.ReadFile, os.Stat, plugin.ExecRunner{}))
}

func run(
	ctx context.Context,
	stdout, stderr io.Writer,
	getenv func(string) string,
	cwd string,
	lookPath func(string) (string, error),
	yarnVersion func(context.Context) (string, error),
	readFile func(string) ([]byte, error),
	stat func(string) (os.FileInfo, error),
	runner plugin.Runner,
) int {
	_, _ = fmt.Fprintf(stderr, "plugin_schema_version=%d\n", pluginSchemaVersion)

	config, err := plugin.ConfigFromEnv(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "publisher-npm:", err)
		return 1
	}

	plan, err := plugin.PlanPublish(ctx, cwd, config, plugin.Dependencies{
		LookPath:    lookPath,
		YarnVersion: yarnVersion,
		ReadFile:    readFile,
		Stat:        stat,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "publisher-npm:", err)
		return 1
	}

	if config.DryRun {
		_, _ = fmt.Fprintf(stdout, "publisher-npm: [dry-run] would publish %s@%s with %s\n", plan.Package.Name, plan.Package.Version, plan.Command.String())
		return 0
	}

	if err := runner.Run(ctx, plan.Command, stdout, stderr); err != nil {
		var commandErr *plugin.CommandError
		if errors.As(err, &commandErr) {
			_, _ = fmt.Fprintln(stderr, "publisher-npm:", commandErr)
			return commandErr.ExitCode
		}
		_, _ = fmt.Fprintln(stderr, "publisher-npm:", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "publisher-npm: published %s@%s with tag %s\n", plan.Package.Name, plan.Package.Version, plan.Tag)
	return 0
}

// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The publisher-npm Authors

package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	plugin "github.com/SemRels/publisher-npm/internal/plugin"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	called int
	err    error
}

func (f *fakeRunner) Run(context.Context, plugin.Command, io.Writer, io.Writer) error {
	f.called++
	return f.err
}

func TestRunDryRunDoesNotExecute(t *testing.T) {
	t.Parallel()

	cwd := testDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(cwd, "package.json"), []byte(`{"name":"demo","version":"1.2.3"}`), 0o644))

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	runner := &fakeRunner{}

	exitCode := run(
		context.Background(),
		stdout,
		stderr,
		func(key string) string {
			switch key {
			case "SEMREL_NEXT_VERSION":
				return "v1.2.3"
			case "SEMREL_DRY_RUN":
				return "true"
			default:
				return ""
			}
		},
		cwd,
		func(string) (string, error) {
			return "", errors.New("lookPath should not be called")
		},
		func(context.Context) (string, error) {
			return "", errors.New("yarnVersion should not be called")
		},
		os.ReadFile,
		os.Stat,
		runner,
	)

	require.Equal(t, 0, exitCode)
	require.Equal(t, 0, runner.called)
	require.Contains(t, stdout.String(), "would publish demo@1.2.3")
	require.Contains(t, stderr.String(), "plugin_schema_version=1")
}

func TestRunRequiresTokenOutsideDryRun(t *testing.T) {
	t.Parallel()

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	runner := &fakeRunner{}

	exitCode := run(
		context.Background(),
		stdout,
		stderr,
		func(key string) string {
			if key == "SEMREL_VERSION" {
				return "1.2.3"
			}
			return ""
		},
		testDir(t),
		func(name string) (string, error) { return name, nil },
		nil,
		os.ReadFile,
		os.Stat,
		runner,
	)

	require.Equal(t, 1, exitCode)
	require.Equal(t, 0, runner.called)
	require.Contains(t, stderr.String(), "SEMREL_PLUGIN_NPM_TOKEN is required")
}

func testDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "publisher-npm-main-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(dir)) })
	return dir
}

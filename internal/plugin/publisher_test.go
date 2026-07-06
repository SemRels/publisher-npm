// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The publisher-npm Authors

package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectPackageManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		files             []string
		yarnVersion       string
		allowYarnProbe    bool
		expectedManager   PackageManager
		expectProbeCalled bool
	}{
		{name: "pnpm lock wins", files: []string{"pnpm-lock.yaml"}, expectedManager: PackageManagerPNPM},
		{name: "yarn berry via yarnrc", files: []string{"yarn.lock", ".yarnrc.yml"}, expectedManager: PackageManagerYarnBerry},
		{name: "yarn berry via version", files: []string{"yarn.lock"}, yarnVersion: "4.9.1", allowYarnProbe: true, expectedManager: PackageManagerYarnBerry, expectProbeCalled: true},
		{name: "yarn classic via version", files: []string{"yarn.lock"}, yarnVersion: "1.22.22", allowYarnProbe: true, expectedManager: PackageManagerYarnClassic, expectProbeCalled: true},
		{name: "npm default", expectedManager: PackageManagerNPM},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := testDir(t)
			for _, file := range tc.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte("lockfile"), 0o644))
			}

			probeCalled := false
			manager, err := DetectPackageManager(context.Background(), dir, os.Stat, func(context.Context) (string, error) {
				probeCalled = true
				return tc.yarnVersion, nil
			}, tc.allowYarnProbe)
			require.NoError(t, err)
			require.Equal(t, tc.expectedManager, manager)
			require.Equal(t, tc.expectProbeCalled, probeCalled)
		})
	}
}

func TestBuildPublishCommand(t *testing.T) {
	t.Parallel()

	config := Config{Tag: "next", Token: "secret", Provenance: true, Access: "public"}
	command := BuildPublishCommand(PackageManagerPNPM, "/repo", config)
	require.Equal(t, "pnpm", command.Name)
	require.Equal(t, []string{"publish", "--tag", "next", "--provenance", "--access", "public"}, command.Args)
	require.Equal(t, "/repo", command.Dir)
	require.Equal(t, "secret", command.ExtraEnv["NODE_AUTH_TOKEN"])
}

func TestBuildPublishCommandYarnBerry(t *testing.T) {
	t.Parallel()

	command := BuildPublishCommand(PackageManagerYarnBerry, "/repo", Config{Tag: "beta", Access: "restricted"})
	require.Equal(t, "yarn", command.Name)
	require.Equal(t, []string{"npm", "publish", "--tag", "beta", "--access", "restricted"}, command.Args)
}

func TestPlanPublishDryRunSkipsCommandLookup(t *testing.T) {
	t.Parallel()

	dir := testDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"demo","version":"1.2.3"}`), 0o644))

	lookPathCalled := false
	plan, err := PlanPublish(context.Background(), dir, Config{Version: "1.2.3", DryRun: true, Tag: "latest", PackageFile: defaultPackageFile}, Dependencies{
		LookPath: func(string) (string, error) {
			lookPathCalled = true
			return "", nil
		},
		YarnVersion: func(context.Context) (string, error) {
			return "", nil
		},
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
	})
	require.NoError(t, err)
	require.False(t, lookPathCalled)
	require.Equal(t, PackageManagerNPM, plan.Manager)
	require.Equal(t, "demo", plan.Package.Name)
	require.Equal(t, "1.2.3", plan.Package.Version)
}

func testDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "publisher-npm-test-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(dir)) })
	return dir
}

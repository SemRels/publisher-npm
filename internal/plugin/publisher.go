// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The publisher-npm Authors

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const defaultPackageFile = "package.json"

type PackageManager string

const (
	PackageManagerNPM         PackageManager = "npm"
	PackageManagerPNPM        PackageManager = "pnpm"
	PackageManagerYarnBerry   PackageManager = "yarn-berry"
	PackageManagerYarnClassic PackageManager = "yarn-classic"
)

type Config struct {
	Version     string
	DryRun      bool
	Token       string
	Tag         string
	Provenance  bool
	Access      string
	PackageFile string
}

type Command struct {
	Name     string
	Args     []string
	Dir      string
	ExtraEnv map[string]string
}

type Plan struct {
	Manager PackageManager
	Command Command
	Package PackageMetadata
	Tag     string
}

type PackageMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Dependencies struct {
	LookPath    func(string) (string, error)
	YarnVersion func(context.Context) (string, error)
	ReadFile    func(string) ([]byte, error)
	Stat        func(string) (fs.FileInfo, error)
}

type Runner interface {
	Run(context.Context, Command, io.Writer, io.Writer) error
}

type ExecRunner struct{}

type CommandError struct {
	Command  Command
	Err      error
	ExitCode int
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("%s failed: %v", e.Command.String(), e.Err)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

func (c Command) String() string {
	return strings.TrimSpace(strings.Join(append([]string{c.Name}, c.Args...), " "))
}

func (r ExecRunner) Run(ctx context.Context, command Command, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = os.Environ()
	for key, value := range command.ExtraEnv {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	if err := cmd.Run(); err != nil {
		exitCode := 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return &CommandError{Command: command, Err: err, ExitCode: exitCode}
	}
	return nil
}

func ConfigFromEnv(getenv func(string) string) (Config, error) {
	dryRun, err := parseBool(getenv("SEMREL_DRY_RUN"), false)
	if err != nil {
		return Config{}, fmt.Errorf("parse SEMREL_DRY_RUN: %w", err)
	}

	provenance, err := parseBool(getenv("SEMREL_PLUGIN_NPM_PROVENANCE"), false)
	if err != nil {
		return Config{}, fmt.Errorf("parse SEMREL_PLUGIN_NPM_PROVENANCE: %w", err)
	}

	version := strings.TrimSpace(getenv("SEMREL_VERSION"))
	if version == "" {
		version = strings.TrimSpace(getenv("SEMREL_NEXT_VERSION"))
	}
	if version == "" {
		return Config{}, errors.New("SEMREL_VERSION is required")
	}

	access := strings.TrimSpace(getenv("SEMREL_PLUGIN_NPM_ACCESS"))
	if access != "" && access != "public" && access != "restricted" {
		return Config{}, fmt.Errorf("SEMREL_PLUGIN_NPM_ACCESS must be public or restricted, got %q", access)
	}

	return Config{
		Version:     strings.TrimPrefix(version, "v"),
		DryRun:      dryRun,
		Token:       strings.TrimSpace(getenv("SEMREL_PLUGIN_NPM_TOKEN")),
		Tag:         defaultString(strings.TrimSpace(getenv("SEMREL_PLUGIN_NPM_TAG")), "latest"),
		Provenance:  provenance,
		Access:      access,
		PackageFile: defaultPackageFile,
	}, nil
}

func PlanPublish(ctx context.Context, cwd string, config Config, deps Dependencies) (Plan, error) {
	manager, err := DetectPackageManager(ctx, cwd, deps.Stat, deps.YarnVersion, !config.DryRun)
	if err != nil {
		return Plan{}, err
	}
	if !config.DryRun && config.Token == "" {
		return Plan{}, errors.New("SEMREL_PLUGIN_NPM_TOKEN is required unless SEMREL_DRY_RUN=true")
	}

	command := BuildPublishCommand(manager, cwd, config)
	if !config.DryRun {
		if _, err := deps.LookPath(command.Name); err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return Plan{}, fmt.Errorf("publish command %q not found on PATH", command.Name)
			}
			return Plan{}, fmt.Errorf("locate publish command %q: %w", command.Name, err)
		}
	}

	pkg := LoadPackageMetadata(deps.ReadFile, filepath.Join(cwd, config.PackageFile), config.Version)
	if pkg.Version == "" {
		pkg.Version = config.Version
	}

	return Plan{Manager: manager, Command: command, Package: pkg, Tag: config.Tag}, nil
}

func DetectPackageManager(
	ctx context.Context,
	cwd string,
	stat func(string) (fs.FileInfo, error),
	yarnVersion func(context.Context) (string, error),
	allowYarnVersionProbe bool,
) (PackageManager, error) {
	if fileExists(stat, filepath.Join(cwd, "pnpm-lock.yaml")) {
		return PackageManagerPNPM, nil
	}
	if fileExists(stat, filepath.Join(cwd, "yarn.lock")) {
		if fileExists(stat, filepath.Join(cwd, ".yarnrc.yml")) {
			return PackageManagerYarnBerry, nil
		}
		if allowYarnVersionProbe && yarnVersion != nil {
			version, err := yarnVersion(ctx)
			if err == nil && parseMajor(version) >= 2 {
				return PackageManagerYarnBerry, nil
			}
		}
		return PackageManagerYarnClassic, nil
	}
	return PackageManagerNPM, nil
}

func BuildPublishCommand(manager PackageManager, cwd string, config Config) Command {
	command := Command{
		Dir:      cwd,
		ExtraEnv: map[string]string{},
	}

	switch manager {
	case PackageManagerPNPM:
		command.Name = "pnpm"
		command.Args = []string{"publish"}
	case PackageManagerYarnBerry:
		command.Name = "yarn"
		command.Args = []string{"npm", "publish"}
	case PackageManagerYarnClassic:
		command.Name = "yarn"
		command.Args = []string{"publish", "--non-interactive"}
	default:
		command.Name = "npm"
		command.Args = []string{"publish"}
	}

	command.Args = append(command.Args, "--tag", config.Tag)
	if config.Provenance {
		command.Args = append(command.Args, "--provenance")
	}
	if config.Access != "" {
		command.Args = append(command.Args, "--access", config.Access)
	}
	if config.Token != "" {
		command.ExtraEnv["NODE_AUTH_TOKEN"] = config.Token
	}

	return command
}

func YarnVersion(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "yarn", "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func LoadPackageMetadata(readFile func(string) ([]byte, error), path, fallbackVersion string) PackageMetadata {
	metadata := PackageMetadata{Name: "(unknown)", Version: fallbackVersion}
	if readFile == nil {
		return metadata
	}
	data, err := readFile(path)
	if err != nil {
		return metadata
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return PackageMetadata{Name: "(unknown)", Version: fallbackVersion}
	}
	if strings.TrimSpace(metadata.Name) == "" {
		metadata.Name = "(unknown)"
	}
	if strings.TrimSpace(metadata.Version) == "" {
		metadata.Version = fallbackVersion
	}
	return metadata
}

func parseBool(raw string, defaultValue bool) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, err
	}
	return value, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func fileExists(stat func(string) (fs.FileInfo, error), path string) bool {
	if stat == nil {
		return false
	}
	_, err := stat(path)
	return err == nil
}

var leadingDigits = regexp.MustCompile(`^(\d+)`)

func parseMajor(version string) int {
	match := leadingDigits.FindString(strings.TrimSpace(version))
	if match == "" {
		return 0
	}
	major, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}
	return major
}

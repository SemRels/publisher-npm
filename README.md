# publisher-npm

[![Latest Release](https://img.shields.io/github/v/release/SemRels/publisher-npm?label=version&color=blue)](https://github.com/SemRels/publisher-npm/releases/latest)
[![CI](https://github.com/SemRels/publisher-npm/actions/workflows/ci.yml/badge.svg)](https://github.com/SemRels/publisher-npm/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/SemRels/publisher-npm)](LICENSE)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/SemRels/publisher-npm/badge)](https://securityscorecards.dev/viewer/?uri=github.com/SemRels/publisher-npm)

`publisher-npm` is a semrel publisher plugin that runs `npm publish`, `pnpm publish`, `yarn npm publish`, or `yarn publish` after semrel has selected and written the release version.

This plugin is distributed as the standalone Go binary `semrel-plugin-publisher-npm`. Semrel executes the binary as a subprocess, provides plugin configuration through `SEMREL_PLUGIN_*` environment variables, provides release context through `SEMREL_*` environment variables, reads standard output, and treats exit code `0` as success and any non-zero exit code as failure. Install the binary in `~/.semrel/plugins/` or anywhere on your `$PATH`.

## Features

- Autodetects `pnpm`, Yarn Berry, Yarn Classic, or `npm`
- Uses `NODE_AUTH_TOKEN` sourced from `SEMREL_PLUGIN_NPM_TOKEN`
- Supports dist-tags, npm provenance, and scoped package access flags
- Mirrors semrel dry-run mode without invoking the publish command

## Installation

### Binary

```bash
go install github.com/SemRels/publisher-npm/cmd/plugin@latest
```

### Docker

```bash
docker pull ghcr.io/semrels/publisher-npm:latest
```

## Configuration

```yaml
plugins:
  - name: publisher-npm
    path: ~/.semrel/plugins/semrel-plugin-publisher-npm
    env:
      SEMREL_PLUGIN_NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
      SEMREL_PLUGIN_NPM_TAG: latest
      SEMREL_PLUGIN_NPM_PROVENANCE: "true"
      SEMREL_PLUGIN_NPM_ACCESS: public
```

## `.semrel.yaml` example

```yaml
tagPrefix: "v"

plugins:
  - uses: condition-github-actions
    phase: condition

  - uses: updater-npm
    phase: update
    env:
      SEMREL_PLUGIN_UPDATE_LOCKFILE: "true"

  - uses: publisher-npm
    phase: publish
    env:
      SEMREL_PLUGIN_NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
      SEMREL_PLUGIN_NPM_TAG: latest
      SEMREL_PLUGIN_NPM_PROVENANCE: "true"
      SEMREL_PLUGIN_NPM_ACCESS: public
```

## `SEMREL_PLUGIN_*` variables

| Name | Required | Description | Default |
| --- | --- | --- | --- |
| `SEMREL_PLUGIN_NPM_TOKEN` | Required unless `SEMREL_DRY_RUN=true` | npm auth token passed to the publish tool as `NODE_AUTH_TOKEN`. | _none_ |
| `SEMREL_PLUGIN_NPM_TAG` | Optional | Dist-tag passed as `--tag <value>`. | `latest` |
| `SEMREL_PLUGIN_NPM_PROVENANCE` | Optional | When `true`, appends `--provenance` to the publish command. | `false` |
| `SEMREL_PLUGIN_NPM_ACCESS` | Optional | When set to `public` or `restricted`, appends `--access <value>`. Useful for scoped packages. | _unset_ |

## `SEMREL_*` release context used

| Variable | Description |
| --- | --- |
| `SEMREL_VERSION` | Resolved release version for the current run. |
| `SEMREL_NEXT_VERSION` | Next version computed by semrel for the release. |
| `SEMREL_DRY_RUN` | Whether semrel is running in dry-run mode. |

## Package-manager detection

`publisher-npm` inspects the current working directory in this order:

1. `pnpm-lock.yaml` → `pnpm publish`
2. `yarn.lock` + `.yarnrc.yml` → Yarn Berry `yarn npm publish`
3. `yarn.lock` + `yarn --version >= 2` → Yarn Berry `yarn npm publish`
4. `yarn.lock` → Yarn Classic `yarn publish`
5. no matching lockfile → `npm publish`

## Dry-run behavior

When `SEMREL_DRY_RUN=true`, the plugin prints the package name, version, and publish command it would use, but it does not call `npm`, `pnpm`, or `yarn`.

## Example output

```text
publisher-npm: [dry-run] would publish @semrels/demo@1.2.3 with npm publish --tag latest --provenance --access public
```

## Development

```bash
go build ./...
go test ./...
go vet ./...
```

## License

Apache-2.0

# publisher-npm

[![Latest Release](https://img.shields.io/github/v/release/SemRels/publisher-npm?label=version&color=blue)](https://github.com/SemRels/publisher-npm/releases/latest)
[![CI](https://github.com/SemRels/publisher-npm/actions/workflows/ci.yml/badge.svg)](https://github.com/SemRels/publisher-npm/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/SemRels/publisher-npm)](LICENSE)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/SemRels/publisher-npm/badge)](https://securityscorecards.dev/viewer/?uri=github.com/SemRels/publisher-npm)

`publisher-npm` is a semrel publisher plugin that runs `npm publish`, `pnpm publish`, `yarn npm publish`, or `yarn publish` after semrel has selected and written the release version.

This plugin is distributed as the standalone Go binary `semrel-plugin-publisher-npm`. Semrel executes the binary as a subprocess, provides plugin configuration through `SEMREL_PLUGIN_*` environment variables, provides release context through `SEMREL_*` environment variables, reads standard output, and treats exit code `0` as success and any non-zero exit code as failure. Install the binary in `~/.semrel/plugins/` or anywhere on your `$PATH`.

## Features

- Autodetects `pnpm`, Yarn Berry, Yarn Classic, or `npm`
- Uses `NODE_AUTH_TOKEN` sourced from `SEMREL_PLUGIN_NPM_TOKEN`, or publishes tokenless via npm trusted publishing (OIDC)
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

semrel passes each `args:` entry to the plugin as `SEMREL_PLUGIN_<KEY>` (key uppercased, `-`/`.`/spaces replaced by `_`). Plugin variables already set in the process environment, for example by a CI workflow, work as well, and empty `args:` values are ignored.

```yaml
plugins:
  - uses: publisher-npm
    args:
      npm_token: ${NPM_TOKEN}     # SEMREL_PLUGIN_NPM_TOKEN, omit for OIDC publishing
      npm_tag: latest             # SEMREL_PLUGIN_NPM_TAG
      npm_provenance: "true"      # SEMREL_PLUGIN_NPM_PROVENANCE
      npm_access: public          # SEMREL_PLUGIN_NPM_ACCESS
```

## `.semrel.yaml` example

```yaml
tagPrefix: "v"

plugins:
  - uses: condition-github-actions
    phase: condition

  - uses: updater-npm
    args:
      update_lockfile: "true"

  - uses: publisher-npm
    args:
      npm_tag: latest
      npm_provenance: "true"
      npm_access: public
    # no token needed with OIDC, see below; otherwise export
    # SEMREL_PLUGIN_NPM_TOKEN from a secret in your CI workflow
```

## Tokenless publishing with OIDC

On GitHub Actions with `permissions: id-token: write`, the plugin does not need a token. When `ACTIONS_ID_TOKEN_REQUEST_URL` is set and `SEMREL_PLUGIN_NPM_TOKEN` is empty, `NODE_AUTH_TOKEN` is left unset and npm authenticates through [trusted publishing](https://docs.npmjs.com/trusted-publishers).

Requirements:

- npm >= 11.5.1 on the runner.
- A trusted publisher (repository and workflow filename) configured for each package on npmjs.com.
- Leave `SEMREL_PLUGIN_NPM_TOKEN` unset (an empty value is ignored). If a token is set, it takes precedence.

## `SEMREL_PLUGIN_*` variables

| Name | Required | Description | Default |
| --- | --- | --- | --- |
| `SEMREL_PLUGIN_NPM_TOKEN` | Required unless `SEMREL_DRY_RUN=true` or OIDC is available | npm auth token passed to the publish tool as `NODE_AUTH_TOKEN`. Omit it to publish with [npm trusted publishing](#tokenless-publishing-with-oidc). | _none_ |
| `SEMREL_PLUGIN_NPM_TAG` | Optional | Dist-tag passed as `--tag <value>`. | `latest` |
| `SEMREL_PLUGIN_NPM_PROVENANCE` | Optional | When `true`, appends `--provenance` to the publish command. | `false` |
| `SEMREL_PLUGIN_NPM_ACCESS` | Optional | When set to `public` or `restricted`, appends `--access <value>`. Useful for scoped packages. | _unset_ |

## `SEMREL_*` release context used

| Variable | Description |
| --- | --- |
| `SEMREL_VERSION` | Resolved release version for the current run. |
| `SEMREL_NEXT_VERSION` | Next version computed by semrel for the release. |
| `SEMREL_DRY_RUN` | Whether semrel is running in dry-run mode. |
| `ACTIONS_ID_TOKEN_REQUEST_URL` | Set by GitHub Actions when `permissions.id-token: write` is granted; enables tokenless publishing. |

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

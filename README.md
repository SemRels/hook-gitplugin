# hook-gitplugin

[![Latest Release](https://img.shields.io/github/v/release/SemRels/hook-gitplugin?label=version\&color=blue)](https://github.com/SemRels/hook-gitplugin/releases/latest)

Pushes release tags and related git updates to another repository.

This plugin is distributed as the standalone Go binary `semrel-plugin-hook-gitplugin`. Semrel executes the binary as a subprocess, provides plugin configuration through `SEMREL_PLUGIN_*` environment variables, provides release context through `SEMREL_*` environment variables, reads standard output, and treats exit code `0` as success and any non-zero exit code as failure. Install the binary in `~/.semrel/plugins/` or anywhere on your `$PATH`.

## Installation

### Binary

```bash
go install github.com/SemRels/hook-gitplugin/cmd/plugin@latest
```

### Docker

Pre-built, multi-platform images (linux/amd64, linux/arm64) are published to the GitHub Container Registry on every release:

```bash
docker pull ghcr.io/semrels/hook-gitplugin:latest
```

Images are signed with [cosign](https://github.com/sigstore/cosign) and include a full SBOM attestation. Verify the signature:

```bash
cosign verify ghcr.io/semrels/hook-gitplugin:latest \
  --certificate-identity-regexp 'https://github.com/SemRels/hook-gitplugin/.github/workflows/release.yml.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```


## Configuration

```yaml
plugins:
  - name: hook-gitplugin
    path: ~/.semrel/plugins/semrel-plugin-hook-gitplugin
    env:
      SEMREL_PLUGIN_REPO: "https://github.com/acme/releases-mirror.git"
      SEMREL_PLUGIN_BRANCH: "main"
      SEMREL_PLUGIN_TOKEN: "${GIT_TOKEN}"
```

## `SEMREL_PLUGIN_*` variables

| Name | Required | Description | Default |
| --- | --- | --- | --- |
| `SEMREL_PLUGIN_REPO` | Required | Git repository URL that receives the tag or release updates. | None |
| `SEMREL_PLUGIN_BRANCH` | Optional | Branch to update in the target repository. | main |
| `SEMREL_PLUGIN_TOKEN` | Optional | Token used when authenticating to the target repository. | None |

## `SEMREL_*` release context used

| Variable | Description |
| --- | --- |
| `SEMREL_VERSION` | Resolved release version for the current run. |
| `SEMREL_TAG_NAME` | Git tag name semrel will create or publish. |
| `SEMREL_NEXT_VERSION` | Next version computed by semrel for the release. |
| `SEMREL_BRANCH` | Git branch associated with the current release run. |
| `SEMREL_TAG_PREFIX` | Configured tag prefix used when composing release tags. |
| `SEMREL_DRY_RUN` | Whether semrel is running in dry-run mode. |

## Example behavior

The plugin clones or updates the target repository, pushes the release tag and related refs, and prints the remote operations it performed. In dry-run mode it only logs the intended git actions.

## License

Apache-2.0

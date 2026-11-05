# Contributing

Thanks for contributing to linux-mcp. Keep it small, focused, and tested.

## Setup

- Go 1.26+ and Linux
- `make build` to compile
- `make demo` to regenerate the demo gif

## Dev loop

Before pushing, make sure everything passes:

```bash
make check   # fmt, vet, lint (golangci-lint)
make test    # go test -race -v ./...
```

[pre-commit](https://pre-commit.com) hooks (format, lint, vet, gitleaks) run on commit - `pre-commit install` to enable them.

## Guidelines

- One logical change per PR, as small as possible.
- New tools/resources live in `tools/`, follow the existing handler pattern.
- No new deps without good reason.
- Include tests with every behavior change or bug fix.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org): `type(scope): summary` in lowercase, no trailing period. e.g. `feat(tools): add disk I/O stats`, `fix(network): resolve port parse bug`.

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`, `ci`.

## Pull requests

- Target `main`.
- Use the PR template: What & Why, Verification, Breaking Changes.
- Link related issues (e.g. `Closes #123`).
- CI and pre-commit checks must pass.

## Issues

Use the bug report or feature request template. Include OS/kernel/version info and exact steps for bugs.

## Security

Found a vulnerability? Don't open a public issue - email the maintainers privately.

## Release

Release from a clean `main` synchronized with `origin/main`, on Linux amd64 with a matching Go target and GNU `timeout`. Commit and push fixes before starting, including any changes made by formatting or lint autofixes.

```bash
make release RELEASE_VERSION=vX.Y.Z
```

`VERSION=vX.Y.Z` remains supported. Bare `make release` prompts with the next patch version. Run `scripts/release.sh --help` for usage.

The release script tags the final source commit before building in a detached worktree. Both binaries must report that tag, its commit, and a clean checkout in their Go build metadata, and return the expected application version through MCP initialization. Signing stays local and uses the maintainer identity expected by `scripts/verify-signature.sh`.

- Tracked `server.json` is the manifest template, not the file to publish directly.
- Generated `bin/server.json` contains the release version, download URL, and checksum of the final binary. It is uploaded as the `server.json` release asset.
- Verification, signing, uploads, and publication use private staging files. `bin/` retains output copies, so a concurrent development build cannot replace publication inputs.
- The registry receives that generated manifest. No checksum commit is added to the tagged source.
- Published tags and assets must not be moved, deleted, or replaced. Release corrections under a new version.

If only registry publication fails after the release assets are available, use:

```bash
scripts/release.sh --resume-publish vX.Y.Z
```

This downloads and verifies the existing artifacts, signatures, provenance, and manifest before republishing, without rebuilding or signing again. Alternatively, run the current **Retry MCP Registry Publication** workflow with that release's explicit `release_tag`; it checks the existing primary binary against the downloaded manifest before publication. Older releases without a `server.json` asset cannot use these recovery paths.

For a partial upload, retain `bin/` and repair only missing assets using the original files. Never rebuild or replace assets already uploaded. Once all assets are available, resume publication as above.

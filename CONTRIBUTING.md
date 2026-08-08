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

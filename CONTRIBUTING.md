# Contributing

Contributions are welcome. Please open an issue before starting significant work.

## Prerequisites

- Go 1.26.1 or later
- [just](https://github.com/casey/just) task runner
- [golangci-lint](https://golangci-lint.run) installed automatically by `just lint`

## Dev Setup

```bash
git clone https://github.com/wood-jp/xerrors
cd xerrors
```

No other setup is required.

## Common Tasks

| Command       | What it does                                              |
|---------------|-----------------------------------------------------------|
| `just test`       | Run tests with race detector, shuffle, and coverage   |
| `just lint`       | Run golangci-lint                                     |
| `just tidy`       | `go mod tidy`, `go fix`, `go fmt`                     |
| `just vuln`       | Run govulncheck                                       |
| `just actionlint` | Lint GitHub Actions workflow files                    |

CI runs `go vet`, `govulncheck`, `go test -race`, and `golangci-lint` on every PR.
Changes to `.github/workflows/` also trigger `actionlint` in CI. Run `just actionlint` locally before pushing.

## Making Changes

- Branch naming: `feature/<short-description>` or `fix/<short-description>`
- Keep changes focused. One concern per PR please.
- `golangci-lint` must pass.
- All new code must have tests; table-driven, using stdlib `testing` only
- Use `t.Parallel()` in every test and subtest that can safely run concurrently
- Test files use `package foo_test` (black-box) unless white-box access is needed

## Pull Requests

- PR title must follow [Conventional Commits](https://www.conventionalcommits.org/). This is enforced by CI.
  - Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `build`
  - Example: `feat(errclass): add WithOnlyEqual option`
- All PRs are squash-merged to keep `main` history linear
- At least one human review is required before merge

## LLM Usage

There should be a human involved. Contributions created solely by bots will likely be rejected. Feel free to employ any and all tools at your disposal, including LLMs. However you, and only you, are responsible for the final result.

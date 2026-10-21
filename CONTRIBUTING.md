# Contributing to KubeOsprey

Thanks for contributing! KubeOsprey is an independently maintained fork of
[kite-org/kite](https://github.com/kite-org/kite) — a single-binary
Kubernetes workspace for multi-cluster operations.

## Development

```bash
git clone https://github.com/fadewalk/kube-osprey.git
cd kube-osprey
make deps     # Go modules + ui/pnpm + docs/pnpm
make dev      # backend on :8080 + Vite dev server with hot reload
```

Go 1.26, Node.js ^20.19 (or >=22.12), pnpm and Make are required.
E2E tests run against a local kind cluster:

```bash
make e2e-test            # kind up + full Playwright suite
```

## Before Opening a PR

- AI-assisted coding is welcome — maintainers also use AI. Please review
  AI-generated changes yourself before submitting, and keep the code clean:
  no AI noise, duplicated code, or unnecessary comments.
- For new features, please open a feature request issue first.
- Run the checks locally:

```bash
make pre-commit          # gofmt/vet/golangci-lint + eslint/prettier
go test ./...
pnpm --dir ui run test
```

- Keep changes scoped to what the PR describes.
- Follow the existing commit style (conventional commits:
  `feat:` / `fix:` / `docs:` / `chore:` / `ci:`).
- New user-visible UI strings go through i18n — update **both**
  `ui/src/i18n/locales/en.json` and `zh.json`.
- Sensitive values (kubeconfigs, API keys, certificates) are stored via
  `model.SecretString`; never log them.

## Merge strategy

This fork has diverged from upstream: feature work lands on the `dev`
branch, `main` is fast-forwarded after verification. Upstream changes are
merged into `dev` (never the other way around). PRs should target `dev`.

## Reporting issues

Use the issue templates. For security vulnerabilities, do **not** open a
public issue — see [SECURITY.md](SECURITY.md).

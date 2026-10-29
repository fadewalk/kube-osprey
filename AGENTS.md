# AGENTS.md

KubeOsprey is a single-binary Kubernetes console: Go backend, React/Vite frontend in
`ui/`, and built frontend assets embedded from `static/`.

It began as a fork of `kite-org/kite` but has fully diverged: the Go module
path is `github.com/fadewalk/kube-osprey`, environment variables use the
`KUBE_OSPREY_*` prefix, and the chart lives at `charts/kube-osprey` (published
as `oci://ghcr.io/fadewalk/charts/kube-osprey`). Do not reintroduce `kite`
identifiers. A small set of compatibility shims and external contracts keep
the legacy names and must stay: `KITE_*` env vars and the `kite<ID>-` API key
prefix are still accepted (deprecation warning), `requires.kite` in plugin
manifests and the SDK-pinned federation names (`@kite-dev/plugin-sdk`,
`kite_plugin_*`, `kite-plugin-*` i18n namespace, host name `kite`) are read
as-is, and `charts.kitehq.dev` / `plugins.kitehq.dev` are external upstream
services.

## Scope

- Make only requested or clearly necessary changes. Keep them local; avoid
  unrelated refactors, extra features, and speculative abstractions or fallbacks.
- Do not write new tests unless the user explicitly asks.
- Let code explain itself. Add comments only when needed to explain non-obvious
  intent or constraints, and keep them brief.

## Commands and verification

- Local development: `make dev`. Full build: `make build`.
- Go: run existing tests for the affected package, e.g. `go test ./pkg/resources`.
- Frontend: `pnpm --dir ui exec tsc --noEmit -p tsconfig.app.json`.
  Use `tsconfig.node.json` for `ui/vite.config.ts`. The `type-check` script's
  root config has no source files, so use these explicit project commands.
- Before committing, `make pre-commit` must pass. It formats and lints the whole
  repository; inspect its changes and keep the commit scoped.

Other commands are in `Makefile` and `ui/package.json`. Choose checks relevant to
the change and stop once they pass. For documentation-only edits, review the
diff and referenced paths or commands; skip application tests and builds.

## Project conventions

- Use existing Kubernetes resource handlers and clients. Preserve cluster scope,
  namespace scope, and `_all` behavior, including existing auth, RBAC, and AI tool
  authorization paths.
- When adding or renaming resource surfaces, keep `pkg/common/resource.go`,
  `pkg/resources/handler.go`, `ui/src/lib/resource-catalog.ts`, and the relevant
  routes/pages/components in sync.
- Store sensitive persisted values with `model.SecretString`; never log secrets.
- When changing runtime environment variables, check both
  `pkg/common/common.go` and the relevant values/templates under `charts/kite`.
  Settings managed by `internal/config.go` must remain read-only in the UI.
- Releases are cut by pushing a `vX.Y.Z` tag on `main`. The release workflow
  stamps the chart version from the tag, so do not bump
  `charts/kite/Chart.yaml` by hand.

## UI

- Keep UI simple and consistent with existing visual and interaction patterns.
- Reuse existing components; place reusable feature components under
  `ui/src/components`.
- Treat `ui/src/components/ui` as shadcn-managed primitives; edit them only when
  the primitive itself must change.
- Use i18n for user-visible text and keep `ui/src/i18n/locales/en.json` and
  `zh.json` in sync.

## Generated files

- Do not hand-edit `static/`, `kube-osprey`, `bin/`, or dependency/cache directories.
  Regenerate frontend assets with `make frontend`.
- Change `go.sum`, `ui/pnpm-lock.yaml`, or `e2e/pnpm-lock.yaml` only when
  dependency changes require it.

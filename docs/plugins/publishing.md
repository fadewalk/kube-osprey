---
outline: deep
---

# Publishing Plugins

Plugins are packaged as `.tar.gz` archives. Distribute them directly or host your own catalog — these are the recommended ways to make plugins available to KubeOsprey users. You can also contribute a plugin to the upstream [kite-plugins](https://github.com/kite-org/kite-plugins) repository, which maintains the official catalog of the Kite plugin ecosystem that KubeOsprey is compatible with.

## Version Rules

- Use a **semantic version without a `v` prefix**, such as `0.1.3`.
- **Package contents for the same ID and version are immutable.** KubeOsprey stores assets by their SHA-256 digest. Any content change, even a single string, requires a version increment; otherwise, KubeOsprey rejects the installation.
- `engines.kite` specifies the KubeOsprey versions the plugin supports. Only stable releases (`X.Y.Z` or `vX.Y.Z`) enforce this range during installation and when displaying the catalog. Development, prerelease, and other non-release builds skip KubeOsprey version matching. All builds still check SDK compatibility.

## README

The plugin's root `README.md` is copied into `dist/` and included in the archive. When distributed through a catalog, KubeOsprey can preview this README. Use full repository URLs for links to files that are not included in the package.

## Private Distribution

### Direct Installation

Upload the `.tar.gz` archive through **Plugin management → Install from file**. This works for internal use and small groups. The archive limit is 50 MiB.

### Host a Static Catalog

Generate static files for a plugin catalog:

```sh
pnpm run catalog --base-url https://plugins.example.com/
```

Output:

```text
dist/catalog/
├── catalog.json          # Catalog index: metadata, versions, download URLs, SHA-256
├── packages/             # Plugin archives
└── readmes/              # README preview files
```

Upload `dist/catalog/` to any HTTP(S) static host, then enter the `catalog.json` URL under **Settings → General → Plugin catalog** in KubeOsprey. To host archives separately from the catalog:

```sh
pnpm run catalog \
  --base-url https://plugins.example.com/ \
  --package-base-url https://downloads.example.com/plugins/
```

Upload `packages/` first, then publish `catalog.json` and `readmes/`.

## Contribute to the Upstream Kite Catalog

Submitting a completed plugin to [kite-org/kite-plugins](https://github.com/kite-org/kite-plugins) is the contribution flow of the upstream Kite plugin ecosystem. Once a contribution is reviewed and merged upstream, the plugin becomes available in the catalog that KubeOsprey uses by default. For internal or self-managed plugins, prefer [Private Distribution](#private-distribution) instead.

1. Fork and clone the repository.
2. Place the plugin source in `plugins/<plugin ID>/`, with the directory name matching `name` in `package.json`. Include a `README.md` describing the plugin's features and usage. Do not commit `dist/` or installation archives.
3. Run `pnpm install` at the repository root, then submit a PR with the plugin source and updated `pnpm-lock.yaml`.

After the PR is merged into `main`, the publishing workflow builds unpublished plugin versions and updates these catalogs:

| Catalog | URL |
| ------- | --- |
| GitHub Pages | `https://kite-org.github.io/kite-plugins/catalog.json` |
| EdgeOne | `https://plugins.kitehq.dev/catalog.json` (KubeOsprey's default catalog, maintained by the upstream ecosystem) |

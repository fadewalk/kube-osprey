# Frequently Asked Questions (FAQ)

## Data Sharing

KubeOsprey does not collect any analytics or telemetry data. The optional
usage reporting that existed in upstream Kite has been removed from this
fork — no installation statistics or usage data ever leave your cluster.

## Permission Issues

If you encounter an error message like the following when accessing resources:

```txt
User admin does not have permission to get configmaps in namespace kube-osprey in cluster in-cluster
```

This means that user `admin` does not have permission to access `configmaps` resources in the `kube-osprey` namespace.

You need to refer to the [RBAC Configuration Guide](./config/rbac-config) to configure user permissions.

## Managed Kubernetes Cluster Connection Issues

If you're using a managed Kubernetes cluster (AKS, EKS, GKE, etc.) and encounter authentication errors when adding the cluster to KubeOsprey, this is usually because the default kubeconfig uses `exec` plugins that require CLI tools (like `kubelogin`, `aws`, or `gcloud`).

KubeOsprey runs as a server-side application and cannot execute these client-side authentication tools. Instead, you should use Service Account token-based authentication.

Please refer to the [Managed Kubernetes Cluster Configuration Guide](./config/managed-k8s-auth) for detailed instructions on creating and using Service Account tokens for authentication.

## Persistence Issues

KubeOsprey supports the use of SQLite, MySQL, or PostgreSQL as databases.

You can configure the database connection string using the `DB_DSN` environment variable, and specify the type of database using `DB_TYPE` (default is `sqlite`).

- If SQLite is used, data is stored within the container. This means that if the container is deleted, the data will be lost. To persistently store data, you need to mount a persistent volume at the `/data` path. You can set the environment variable `DB_DSN=/data/db.sqlite`. (Note: `/data` isn’t the default path. You can choose any other path as needed, but make sure the path specified in `DB_DSN` matches the mounted path.)

- If MySQL or PostgreSQL is used, you need to provide the appropriate connection string, such as `DB_DSN=user:password@tcp(host:port)/dbname`.

It’s recommended to install KubeOsprey using a Helm Chart. This makes it easier to configure persistent storage and database connections.

## SQLite with hostPath Storage

If you're using SQLite as the database and encountering an "out of memory" error when using `hostPath` for persistent storage:

```txt
panic: failed to connect database: unable to open database file: out of memory (14)
```

This issue is related to the pure Go SQLite driver used by KubeOsprey (to avoid CGO dependencies). The driver has limitations when accessing database files on certain storage backends.

**Solution**: Add SQLite connection options to improve compatibility with hostPath storage. In your Helm values, set:

```yaml
db:
  sqlite:
    options: "_journal_mode=WAL&_busy_timeout=5000"
```

These options enable Write-Ahead Logging (WAL) mode and increase the busy timeout, which resolves most hostPath compatibility issues.

**Recommended for Production**: For production deployments requiring persistent storage, use MySQL or PostgreSQL instead of SQLite. These databases are better suited for containerized environments and persistent storage scenarios.

For more details, see the upstream Kite issue ([kite-org/kite#204](https://github.com/kite-org/kite/issues/204)).

## How to Change Font

By default, KubeOsprey provides three fonts: system default, `Maple Mono`, and `JetBrains Mono`.

If you want to use a different font, you need to build the project yourself.

Build KubeOsprey with make and change the font in `./ui/src/index.css`:

```css
@font-face {
  font-family: "Maple Mono";
  font-style: normal;
  font-display: swap;
  font-weight: 400;
  src:
    url(https://cdn.jsdelivr.net/fontsource/fonts/maple-mono@latest/latin-400-normal.woff2)
      format("woff2"),
    url(https://cdn.jsdelivr.net/fontsource/fonts/maple-mono@latest/latin-400-normal.woff)
      format("woff");
}

body {
  font-family: "Maple Mono", var(--font-sans);
}
```

## How Can I Contribute to KubeOsprey?

We welcome contributions! You can:

- Report bugs and feature requests on [GitHub Issues](https://github.com/fadewalk/kube-osprey/issues)
- Submit pull requests
- Improve documentation
- Share feedback and use cases

## Where Can I Get Help?

You can get support through:

- [GitHub Issues](https://github.com/fadewalk/kube-osprey/issues) for bug reports and feature requests
- [GitHub Issues](https://github.com/fadewalk/kube-osprey/issues) for questions and community support

---

**Didn't find what you're looking for?** Feel free to [open an issue](https://github.com/fadewalk/kube-osprey/issues) on GitHub.

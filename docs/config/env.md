# Environment Variables

KubeOsprey supports several environment variables by default to change the default values of some configuration items.

::: tip Renamed from `KITE_*`
Environment variables were renamed from the `KITE_` prefix to `KUBE_OSPREY_` (for example `KITE_ENCRYPT_KEY` → `KUBE_OSPREY_ENCRYPT_KEY`). The old names keep working but log a deprecation warning; update your deployment at your convenience.
:::

## Server

- **PORT**: Port on which KubeOsprey runs, default value is `8080`.

- **HOST**: Used for generating OAuth 2.0 authorization callback addresses, default will be obtained from request headers. If you find the result not as expected, you can manually configure this environment variable.

- **KUBE_OSPREY_BASE**: Base path under which KubeOsprey is served, for example `/kite`. Leading and trailing slashes are normalized automatically.

- **TRUSTED_PROXIES**: Comma-separated list of reverse proxy, ingress, or load balancer IPs/CIDRs that KubeOsprey should trust when reading `X-Forwarded-For` / `X-Real-IP` to determine the client IP. By default, KubeOsprey trusts local/private network ranges (`127.0.0.0/8`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `::1`, `fc00::/7`) so common ingress deployments can report real client IPs. Set a narrower value such as `TRUSTED_PROXIES=10.42.0.0/16,192.168.1.10` for production, or `TRUSTED_PROXIES=none` to ignore all client-supplied forwarding headers.

- **CORS_ALLOWED_ORIGINS**: Comma-separated list of origins allowed to call the API cross-origin, for example `http://localhost:5173` for the local Vite dev server. Empty by default (no CORS). Intended for development only.

- **DISABLE_GZIP**: Disable gzip response compression, default value is `true` (compression disabled). Set to `false` to enable gzip.

- **DISABLE_VERSION_CHECK**: Set to `true` to disable the new-version check, default value is `false`.

## Security

- **JWT_SECRET**: Secret key used for signing and verifying JWT
- **KUBE_OSPREY_ENCRYPT_KEY**: Secret key used for encrypting sensitive data, such as OAuth client secrets, kubeconfig, and Cluster Agent private keys. It must be non-empty, at least 32 characters long, and must not equal the built-in development default — otherwise KubeOsprey refuses to start. It must be explicitly configured for production and must remain stable while restoring an existing database.
- **KUBE_OSPREY_ALLOW_INSECURE_DEFAULTS**: Set to `true` only for local development to allow the public development encryption-key fallback. Do not set this in production.
- **ANONYMOUS_USER_ENABLED**: Enable anonymous user access, default value is `false`. When enabled, all access will no longer require authentication and will have the highest permissions by default.

## Database

- **DB_TYPE**: Database type, one of `sqlite`, `mysql`, `postgres`. Default value is `sqlite`. Invalid values cause startup failure.

- **DB_DSN**: Database connection string. The default value `dev.db` is a SQLite file relative to KubeOsprey's working directory. Required for `mysql`/`postgres`.

## Images and Terminals

- **NODE_TERMINAL_IMAGE**: Docker image used for generating Node Terminal Agent, default value is `busybox:latest`.

- **KUBECTL_TERMINAL_IMAGE**: Docker image used for the kubectl terminal, default value is `zzde/kubectl:latest`.

- **CLUSTER_AGENT_IMAGE**: Docker image used when generating the Cluster Agent manifest for Cluster Agent clusters, default value is `ghcr.io/kite-org/kite:latest`.

- **NAMESPACE**: Namespace where terminal and agent pods are created, default value is `kube-system`.

## Kubernetes Client

- **KUBE_OSPREY_CACHE_SYNC_TIMEOUT**: Timeout in seconds for the initial informer cache sync when connecting to a cluster, default value is `60`. Increase it for large or remote clusters.

- **KUBE_OSPREY_KUBE_API_QPS**: Kubernetes API client QPS limit, default value is `50`.

- **KUBE_OSPREY_KUBE_API_BURST**: Kubernetes API client burst limit, default value is `100`.

- **DISABLE_CACHE**: Set to `true` to disable the built-in client-side cache and read directly from the Kubernetes API. Useful for troubleshooting; increase `KUBE_OSPREY_CACHE_SYNC_TIMEOUT` instead when cache sync times out.

## Plugins

- **PLUGIN_DIR**: Directory for installed plugin files. Defaults to `data/plugins`, relative to KubeOsprey's working directory (`/app/data/plugins` in the official container image). The Helm Chart sets it to `/data/plugins` to share SQLite's volume. See [Plugin Storage](./chart-values#plugin-storage) for directory and persistence configuration.

- **PLUGIN_DEV_URL**: Development plugin manifest URL, for example `http://localhost:5174/plugin.json`. The browser loads it directly; it overrides an installed plugin with the same ID without changing installation records. Leave unset to use installed plugins only. Helm deployments can set it through `extraEnvs`. See [Plugin Debugging](/plugins/debugging) for details.

## MCP

- **KUBE_OSPREY_MCP_ENABLED**: Bootstrap default for the built-in MCP server (`POST /api/v1/mcp`). Defaults to `false`. This value seeds the general setting on first run; administrators can toggle MCP later in Settings → General.
- **KUBE_OSPREY_MCP_RATE_LIMIT_PER_MINUTE**: Per-user MCP tool call budget per minute. Defaults to `60`.

## Bootstrap and Legacy

- **KUBE_OSPREY_CONFIG_FILE**: Path to the configuration file. Available in KubeOsprey `v0.10.0` and later. When set, KubeOsprey loads cluster, OAuth, LDAP, RBAC, and super user settings from this file. See [Configuration File](/config/config-file) for details.
- **KUBE_OSPREY_USERNAME**: Legacy environment variable for the initial administrator username. It is only used for env-to-DB migration when `KUBE_OSPREY_CONFIG_FILE` is not set.
- **KUBE_OSPREY_PASSWORD**: Legacy environment variable for the initial administrator password. It is only used for env-to-DB migration when `KUBE_OSPREY_CONFIG_FILE` is not set.
- **KUBECONFIG**: Legacy kubeconfig environment variable used to import clusters when `KUBE_OSPREY_CONFIG_FILE` is not set.

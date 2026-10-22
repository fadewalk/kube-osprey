# Chart Values

This document describes all available configuration options for the KubeOsprey Helm Chart.

## Basic Configuration

| Parameter          | Description                                                | Default                 |
| ------------------ | ---------------------------------------------------------- | ----------------------- |
| `replicaCount`     | Number of replicas                                         | `1`                     |
| `image.repository` | Container image repository                                 | `ghcr.io/fadewalk/kube-osprey` |
| `image.pullPolicy` | Image pull policy                                          | `IfNotPresent`          |
| `image.tag`        | Image tag. If set, will override the chart's `appVersion`. | `""`                    |
| `imagePullSecrets` | Image pull secrets for private repositories                | `[]`                    |
| `nameOverride`     | Override chart name                                        | `""`                    |
| `fullnameOverride` | Override full name                                         | `""`                    |
| `debug`            | Enable debug mode                                          | `false`                 |
| `basePath`         | Base path where KubeOsprey is served. See notes below.           | `""`                    |
| `deploymentStrategy.type` | Deployment update strategy. Keep `Recreate` for SQLite on a `ReadWriteOnce` volume, so two Pods never mount the same volume during upgrades | `Recreate` |

## Authentication & Security

| Parameter              | Description                                                                              | Default                                              |
| ---------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| `anonymousUserEnabled` | Enable anonymous user access with full admin privileges. Use with caution in production. | `false`                                              |
| `jwtSecret`            | Secret key for signing JWT tokens. Auto-generated on first boot if empty.                | `""`                                                 |
| `encryptKey`           | Secret key used for encrypting sensitive data. Leave empty to generate a key for a new chart Secret or use `secret.existingSecret`. Keep it stable when restoring an existing database. | `""` |
| `host`                 | Hostname for the application                                                             | `""`                                                 |

### Secret Configuration

| Parameter                | Description                                                                        | Default |
| ------------------------ | ---------------------------------------------------------------------------------- | ------- |
| `secret.create`          | Create a Kubernetes Secret from chart values (`jwtSecret`, `encryptKey`, `db.*`)   | `true`  |
| `secret.existingSecret`  | Reference an existing Secret instead of creating one. Ignored when `create: true`. | `""`    |

When `secret.existingSecret` is set, the referenced Secret must contain all required keys:

- `JWT_SECRET`
- `KITE_ENCRYPT_KEY`
- `DB_TYPE` — supported values: `sqlite`, `postgres`, `mysql` (defaults to `sqlite` if not set)
- `DB_DSN` — not required for `sqlite`

With an existing Secret, the `jwtSecret`, `encryptKey`, `db.type`, and `db.dsn` chart values are ignored. See [Production Security Hardening](./security-hardening) for key handling guidance.

## Database Configuration

| Parameter | Description                                                              | Default  |
| --------- | ------------------------------------------------------------------------ | -------- |
| `db.type` | Database type: `sqlite`, `postgres`, `mysql`                             | `sqlite` |
| `db.dsn`  | Full DSN string for MySQL/Postgres. Required when type is mysql/postgres | `""`     |

### SQLite Configuration

| Parameter                                 | Description                                               | Default             |
| ----------------------------------------- | --------------------------------------------------------- | ------------------- |
| `db.sqlite.persistence.pvc.enabled`       | Whether to create a PVC to store the sqlite database file | `false`             |
| `db.sqlite.persistence.pvc.existingClaim` | Use existing PVC                                          | `""`                |
| `db.sqlite.persistence.pvc.storageClass`  | StorageClass for PVC (optional)                           | `""`                |
| `db.sqlite.persistence.pvc.accessModes`   | Access modes for PVC                                      | `["ReadWriteOnce"]` |
| `db.sqlite.persistence.pvc.size`          | Requested storage size for PVC                            | `1Gi`               |
| `db.sqlite.persistence.hostPath.enabled`  | Whether to use hostPath storage                           | `false`             |
| `db.sqlite.persistence.hostPath.path`     | hostPath path                                             | `/path/to/host/dir` |
| `db.sqlite.persistence.hostPath.type`     | hostPath type                                             | `DirectoryOrCreate` |
| `db.sqlite.persistence.mountPath`         | Mount path inside container                               | `/data`             |
| `db.sqlite.persistence.filename`          | SQLite filename inside mountPath                          | `kite.db`           |
| `db.sqlite.options`                       | Additional SQLite connection options, e.g. `_journal_mode=WAL&_busy_timeout=5000` | `""`        |

## MCP Server

Available in KubeOsprey `v0.17.0` and later. These values seed the runtime settings on first run; enablement can also be changed later by administrators in **Settings → General**.

| Parameter                 | Description                                                 | Default |
| ------------------------- | ----------------------------------------------------------- | ------- |
| `mcp.enabled`             | Bootstrap default for the built-in MCP server (`POST /api/v1/mcp`) | `false` |
| `mcp.rateLimitPerMinute`  | Per-user MCP tool call budget per minute                    | `60`    |

## Plugin Storage

Available in KubeOsprey `v0.16.0` and later. The default plugin directory is `/data/plugins`, which shares SQLite's `/data` volume. Persistence is disabled by default.

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `plugins.directory` | Plugin directory inside the container; sets `PLUGIN_DIR` | `/data/plugins` |
| `plugins.persistence.enabled` | Mount a separate PVC for plugin files | `false` |
| `plugins.persistence.existingClaim` | Use an existing PVC instead of creating one | `""` |
| `plugins.persistence.storageClass` | StorageClass for the new PVC; empty uses the cluster default | `""` |
| `plugins.persistence.accessModes` | Access modes for the new PVC | `["ReadWriteOnce"]` |
| `plugins.persistence.size` | Storage requested for the new PVC | `1Gi` |

::: warning
If the plugin directory is not on persistent storage, recreating the Pod loses plugin files. Plugins installed from the official catalog are downloaded again, requiring access to the download source. Manually uploaded plugins must be uploaded again.

Automatic downloads rely on installation records in the database. Persist the SQLite database or use an external database as well.
:::

With SQLite, enable database persistence and leave `plugins.persistence.enabled` set to `false`. The database and plugins share one PVC:

```yaml
deploymentStrategy:
  type: Recreate

db:
  sqlite:
    persistence:
      pvc:
        enabled: true
```

If you change `db.sqlite.persistence.mountPath`, also set `plugins.directory` to a subdirectory of that path to keep sharing the volume.

With MySQL or PostgreSQL, enable a separate plugin PVC:

```yaml
deploymentStrategy:
  type: Recreate

plugins:
  directory: /data/plugins
  persistence:
    enabled: true
    size: 1Gi
```

Set `plugins.persistence.existingClaim` to reuse a PVC in KubeOsprey's namespace. With `ReadWriteOnce` storage, use `Recreate` for a single replica to avoid volume attachment conflicts during upgrades.

## Environment Variables

| Parameter   | Description                              | Default |
| ----------- | ---------------------------------------- | ------- |
| `extraEnvs` | List of additional environment variables | `[]`    |

## Application Configuration

KubeOsprey supports loading cluster, OAuth/LDAP, and RBAC configuration from a YAML config file. When enabled, managed sections become read-only in the UI.

Available in KubeOsprey `v0.10.0` and later.

See [Configuration File](./config-file) for the full config file format, usage examples, and reference.

| Parameter                 | Description                                                                      | Default |
| ------------------------- | -------------------------------------------------------------------------------- | ------- |
| `config.enabled`          | Enable configuration file mode                                                   | `false` |
| `config.existingSecret`   | Name of an existing Secret containing a `config.yaml` key. Recommended approach. | `""`    |
| `config.superUser`        | Inline super user configuration (created on first startup only)                  | `{}`    |
| `config.clusters`         | Inline cluster configurations (when no existingSecret)                           | `[]`    |
| `config.oauth`            | Inline OAuth provider configurations                                             | `[]`    |
| `config.ldap`             | Inline LDAP configuration                                                        | `{}`    |
| `config.ai`               | Inline AI assistant configuration                                                | `{}`    |
| `config.rbac.roles`       | Inline RBAC role definitions                                                     | `[]`    |
| `config.rbac.roleMapping` | Inline RBAC role mappings                                                        | `[]`    |

## Service Account Configuration

| Parameter                    | Description                                         | Default |
| ---------------------------- | --------------------------------------------------- | ------- |
| `serviceAccount.create`      | Whether to create a service account                 | `true`  |
| `serviceAccount.automount`   | Automatically mount service account API credentials | `true`  |
| `serviceAccount.annotations` | Annotations for service account                     | `{}`    |
| `serviceAccount.name`        | Name of service account to use                      | `""`    |

## RBAC Configuration

| Parameter     | Description                      | Default           |
| ------------- | -------------------------------- | ----------------- |
| `rbac.create` | Whether to create RBAC resources | `true`            |
| `rbac.rules`  | List of RBAC rules               | See example below |

### RBAC Rules Example

```yaml
rbac:
  rules:
    - apiGroups: ["*"]
      resources: ["*"]
      verbs: ["*"]
    - nonResourceURLs: ["*"]
      verbs: ["*"]
```

## Pod Configuration

| Parameter            | Description                    | Default |
| -------------------- | ------------------------------ | ------- |
| `podAnnotations`     | Kubernetes annotations for Pod | `{}`    |
| `podLabels`          | Kubernetes labels for Pod      | `{}`    |
| `podSecurityContext` | Pod security context           | `{}`    |
| `securityContext`    | Container security context     | `{}`    |

## Service Configuration

| Parameter      | Description  | Default     |
| -------------- | ------------ | ----------- |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8080`      |

## Ingress Configuration

| Parameter             | Description                | Default           |
| --------------------- | -------------------------- | ----------------- |
| `ingress.enabled`     | Whether to enable Ingress  | `false`           |
| `ingress.className`   | Ingress class name         | `"nginx"`         |
| `ingress.annotations` | Ingress annotations        | `{}`              |
| `ingress.hosts`       | Ingress host configuration | See example below |
| `ingress.tls`         | TLS configuration          | `[]`              |

### Ingress Host Configuration Example

```yaml
ingress:
  hosts:
    - host: kube-osprey.example.com
      paths:
        - path: /
          pathType: ImplementationSpecific
```

## Gateway API Configuration

Requires Gateway API CRDs to be installed in the cluster. Enables exposing KubeOsprey through a `Gateway` and `HTTPRoute` instead of an Ingress.

| Parameter                | Description                                                              | Default           |
| ------------------------ | ------------------------------------------------------------------------ | ----------------- |
| `gateway.enabled`        | Whether to create Gateway API resources                                  | `false`           |
| `gateway.className`      | Existing `GatewayClass` name (e.g. `istio`, `cilium`, `envoy-gateway`)   | `""`              |
| `gateway.create`         | Create a new `Gateway` (`true`) or reference an existing one (`false`)   | `false`           |
| `gateway.name`           | Name of the `Gateway` (created or referenced)                            | `""`              |
| `gateway.namespace`      | Namespace of the existing `Gateway`; defaults to the release namespace   | `""`              |
| `gateway.annotations`    | Annotations for the created `Gateway`                                    | `{}`              |
| `gateway.listeners`      | Listener configuration for the created `Gateway`                         | See `values.yaml` |
| `gateway.httpRoutes`     | `HTTPRoute` configuration (supports multiple routes)                     | See `values.yaml` |

## Resource Limits

| Parameter   | Description                            | Default |
| ----------- | -------------------------------------- | ------- |
| `resources` | Container resource limits and requests | `{}` (no default requests or limits are set) |

### Resource Limits Example

```yaml
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

## Health Checks

| Parameter        | Description                   | Default           |
| ---------------- | ----------------------------- | ----------------- |
| `livenessProbe`  | Liveness probe configuration  | See example below |
| `readinessProbe` | Readiness probe configuration | See example below |

### Health Check Example

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 10
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /readyz
    port: http
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 6
```

## Storage Configuration

| Parameter      | Description                            | Default |
| -------------- | -------------------------------------- | ------- |
| `volumes`      | Additional volume configurations       | `[]`    |
| `volumeMounts` | Additional volume mount configurations | `[]`    |

## Scheduling Configuration

| Parameter      | Description               | Default |
| -------------- | ------------------------- | ------- |
| `nodeSelector` | Node selector             | `{}`    |
| `tolerations`  | Tolerations configuration | `[]`    |
| `affinity`     | Affinity configuration    | `{}`    |

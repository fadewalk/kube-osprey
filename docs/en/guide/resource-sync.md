# Resource Sync

Resource Sync is an optional engine that mirrors whitelisted Kubernetes resource types from your clusters into the KubeOsprey database. When enabled, resource lists, global search, and the topology view are served from the local database instead of live API server calls, which keeps KubeOsprey fast on large clusters.

The engine is off by default; KubeOsprey always talks to the API server directly when it is off.

## Enable Resource Sync

1. Open `Settings`.
2. Go to `General`.
3. Turn on `Resource Sync`.
4. Optionally override the resource whitelist (comma-separated plural names, e.g. `pods,deployments,services`). Leave it empty to use the default whitelist:

   `pods, deployments, statefulsets, daemonsets, replicasets, jobs, services, ingresses, configmaps, persistentvolumeclaims, nodes, namespaces, events`

5. Save the settings.

::: warning
Secrets are deliberately excluded from the default whitelist. If you add `secrets` yourself, the stored object JSON will contain the secret data in the KubeOsprey database.
:::

## What Changes When It Is On

- **Resource lists** for whitelisted types are served from the database (fast path). If the synced data is stale or the query fails, KubeOsprey falls back to the live API server automatically — behavior never breaks, it just slows back down.
- **Global search** plain-name queries use the database index across all synced clusters, making cross-cluster search fast.
- **Topology view** becomes available. It walks owner references (Pod ← ReplicaSet ← Deployment, etc.) from the synced data to draw workload relationships.
- Non-whitelisted resource types keep using the direct path, exactly as before.

## Operating Notes

- The engine keeps data fresh with Kubernetes watches (ListAndWatch with automatic reconnect); expect a full re-list after a KubeOsprey restart.
- Synced objects live in the KubeOsprey database. On single-binary SQLite deployments, watch the database size on clusters with many churn-heavy objects (for example, large numbers of Jobs or Events); narrow the whitelist if it grows too fast.
- Turning the engine off returns KubeOsprey to direct mode immediately; already-synced data stays until cleaned up.

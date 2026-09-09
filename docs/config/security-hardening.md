# Production Security Hardening

This guide is the production security baseline for Kite. It applies to Helm installations, the standalone Kubernetes manifest, and Cluster Agent deployments.

Kite is a Kubernetes control plane. It can create privileged Kubernetes clients, terminal pods, exec sessions, proxy requests, and AI-assisted resource changes. Treat a Kite installation as a cluster administration boundary.

## Production baseline

Before exposing Kite to users, verify all of the following:

- Use HTTPS for Kite, OAuth/OIDC callbacks, Cluster Agent connections, and WebSocket traffic.
- Set `KITE_ENCRYPT_KEY` to a high-entropy value and store it in an external Secret. Never use the public example value from development documentation.
- Set `JWT_SECRET` explicitly when you need stable sessions across restarts or multiple replicas.
- Set `ANONYMOUS_USER_ENABLED=false` and verify that anonymous access is disabled.
- Use a narrow `TRUSTED_PROXIES` list. Do not trust arbitrary client-supplied forwarding headers.
- Use an external PostgreSQL or MySQL database, or provision a persistent volume for SQLite.
- Pin Kite, terminal, and Cluster Agent images to a release tag or digest. Do not use `latest` in production.
- Review every `cluster-admin`, wildcard RBAC rule, privileged container, host namespace, and host path before deployment.
- Enable only the terminal and Cluster Agent capabilities that your operators need.
- Back up the database and the encryption key together. A database backup without its encryption key cannot restore encrypted fields.

## Secrets and encryption

`SecretString` fields in the database include credentials such as kubeconfig data, OAuth and LDAP secrets, Helm repository passwords, API keys, MFA secrets, and Cluster Agent private keys. They are encrypted with `KITE_ENCRYPT_KEY`.

User passwords are stored as one-way password hashes; they are not recovered with `KITE_ENCRYPT_KEY`.

Use an existing Kubernetes Secret when possible:

```yaml
secret:
  create: false
  existingSecret: kite-secrets
```

The Secret must contain the keys required by the selected deployment mode. Keep the encryption key stable while restoring an existing database. Rotating it requires a planned migration that can decrypt with the old key and re-encrypt with the new key.

Do not put real keys in `values.yaml`, Git, shell history, issue trackers, or generated manifests.

## Kubernetes permissions

Kite has application-level RBAC, while Kubernetes separately evaluates the ServiceAccount or kubeconfig used by the Kite server and Cluster Agent. The effective permission is the intersection of both systems, but an overly privileged Kubernetes identity can still make a mistake or compromised session more damaging.

Review these areas separately:

- Kite server ServiceAccount and ClusterRole;
- Cluster Agent ServiceAccount and ClusterRoleBinding;
- kubectl terminal ServiceAccount and ClusterRoleBinding;
- Node Terminal host access, privileged mode, host namespaces, and host paths;
- `imagePullSecrets` and the namespaces where terminal pods are created.

Avoid wildcard rules such as `apiGroups: ["*"]`, `resources: ["*"]`, and `verbs: ["*"]` for production unless the installation is intentionally an administrator-only break-glass system.

The generated Cluster Agent manifest currently represents a high-trust deployment. Review and reduce its permissions before using it in a shared or regulated cluster.

## Terminal and exec capabilities

Pod exec, kubectl terminal, and node terminal are not ordinary read-only features:

- Pod exec can run commands inside workloads.
- kubectl terminal can operate on the cluster using its configured ServiceAccount.
- Node terminal can expose host-level capabilities.

Use explicit administrator permissions, short sessions, automatic cleanup, and audit logging. Keep kubectl and node terminal disabled unless they are required. Test that disabled capabilities do not create ServiceAccounts, bindings, or agent pods.

## Prometheus and raw PromQL

The PromQL Explorer is a cluster-level observability tool. It should be granted only to users who are allowed to query the configured Prometheus data source. Kubernetes namespace RBAC does not automatically filter Prometheus results.

If Kite is used as a multi-tenant platform, enforce tenant isolation in the Prometheus layer or replace unrestricted PromQL with approved query templates. Do not attempt to provide isolation by blindly appending a namespace selector to arbitrary PromQL.

## Database and persistence

The default SQLite setup is suitable for evaluation only. Without a persistent volume, replacing the Pod loses the database. For production:

- Prefer PostgreSQL or MySQL for multiple replicas and operational backups.
- If using SQLite, provision a persistent volume and use a deployment strategy compatible with a single-writer volume.
- Test restore procedures, not only backup creation.
- Back up the database, encryption key, and deployment configuration as one recovery set.

## Cluster Agent checklist

- Use an HTTPS server URL with a certificate trusted by the Agent.
- Treat the one-time Agent token and generated YAML as credentials.
- Rotate or revoke the Agent token after suspected exposure.
- Verify the Agent image and digest before applying the manifest.
- Inspect the ServiceAccount, ClusterRoleBinding, Secret, and Deployment before applying.
- Ensure the reverse proxy supports WebSocket upgrade and long-lived connections.
- For multiple Kite replicas, validate session routing and shared state before production use.

## Verification commands

Render and lint the Chart:

```bash
helm lint charts/kite
helm template kite charts/kite --namespace kite-system
bash scripts/validate-chart.sh
```

Review effective Kubernetes permissions in a test cluster:

```bash
kubectl auth can-i --list \
  --as=system:serviceaccount:<namespace>:<service-account>

kubectl get clusterrolebinding <binding-name> -o yaml
kubectl get serviceaccount <service-account> -n <namespace> -o yaml
```

Run application checks:

```bash
go test ./...
pnpm --dir ui exec tsc --noEmit -p tsconfig.app.json
```

Check readiness separately from process liveness. A process can be running while its database, encryption key, or cluster configuration is unusable.

## Current limitations and roadmap

The following items require separate design work rather than a blind permission reduction:

- Cluster Agent currently uses its registered Kubernetes credentials when forwarding requests; it does not impersonate the Kite user at the Kubernetes API.
- kubectl and node terminal require stronger break-glass controls and more narrowly scoped identities.
- Encryption-key rotation needs a migration path for existing `SecretString` values.
- Raw PromQL is not namespace-rewritten by Kite. Multi-tenant Prometheus isolation must be provided by the metrics architecture or a restricted query model.
- Passkey ceremonies, Cluster Agent sessions, and other in-memory state need shared storage before a multi-replica deployment is considered fully supported.

# Security Policy

KubeOsprey holds cluster credentials (kubeconfigs, API keys, tokens) and can
modify the clusters it is connected to. Please treat vulnerability reports
accordingly.

## Supported versions

Only the latest tagged release and `main` receive security fixes.

| Version            | Supported          |
| ------------------ | ------------------ |
| latest release     | :white_check_mark: |
| older releases     | :x:                |

## Reporting a vulnerability

**Do not open a public GitHub issue for security reports.**

Contact [fadewalk](https://github.com/fadewalk) directly (GitHub profile
contact / security advisory), including:

- affected version (`vX.Y.Z`) or commit
- reproduction steps
- impact assessment, if available

You can expect an initial response within 7 days. Please allow a coordinated
disclosure window before publishing details.

## Security design notes

- Startup fails closed: the server refuses to run with the upstream default
  encryption key or keys shorter than 32 characters
  (`KUBE_OSPREY_ENCRYPT_KEY`, `KUBE_OSPREY_ALLOW_INSECURE_DEFAULTS`).
- Sensitive values at rest (kubeconfigs, API keys, MFA secrets, notification
  channel secrets) are AES-GCM encrypted via `model.SecretString`.
- Cluster-mutating operations can be routed through change plans with human
  approval (AI assistant mutations, CI/CD webhook upgrades).
- Operation audit logs record state-changing API calls, with request bodies
  masked for sensitive keys and truncated past 4KB.

## Scope decisions

- Anonymous usage telemetry has been removed entirely — nothing leaves the
  cluster.
- The kubectl terminal, node terminal and MCP server are disabled by default
  and must be enabled explicitly in settings.

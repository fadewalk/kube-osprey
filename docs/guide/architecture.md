# Architecture

KubeOsprey ships as a single Go binary that embeds the React frontend. The diagrams on this page show how a request travels through that binary, how it reaches your clusters, and how the AI assistant, metrics pipeline, and build fit around it.

Every diagram is interactive. The inline view supports zoom, pan, and theme switching; use **Open full screen** to get guided views that walk through one path at a time, component search, and the summary cards.

::: tip Reading order
Start with the [system architecture](#system-architecture) to build a mental model, then follow one call through the [request lifecycle](#resource-api-request-lifecycle). The remaining diagrams go deeper into multi-cluster access, the AI assistant, metrics, and the build, in whatever order matches your interest.
:::

## System architecture

The browser loads the React SPA, which talks to the Gin HTTP server over REST, WebSocket, and SSE. Each API request passes through `RequireAuth` and the RBAC middleware before it reaches a resource handler in `pkg/resources`, and the handler obtains a per-cluster `ClientSet` from the `ClusterManager`. Clusters are reached directly (in-cluster or kubeconfig) or through a [cluster agent](./cluster-agent) that opens a reverse tunnel from an isolated network. The AI assistant, Prometheus access, and the GORM-backed store all hang off the same server.

<DiagramFrame src="/diagrams/en/kube-osprey-architecture.html" title="KubeOsprey system architecture" :height="520" />

Authorization is decided by KubeOsprey's own role model stored in its database, not by Kubernetes RBAC. See [RBAC Configuration](/config/rbac-config) for how roles map to resources and clusters.

## Resource API request lifecycle

A list request such as `GET /api/v1/pods?namespace=_all` authenticates with the `auth_token` cookie or an API key, loads the user and role bindings through a cached lookup, and is checked against the resource verb before the handler runs. Reads are served from the informer cache first and fall back to the Kubernetes API on a miss. When a JWT expires, the frontend silently calls `POST /api/auth/refresh` and retries the original request.

<DiagramFrame src="/diagrams/en/kube-osprey-request-sequence.html" title="Resource API request lifecycle" :height="520" />

The JWT itself lives for 24 hours and the cookie for 48, which is the window in which a silent refresh can succeed. API key format and refresh behavior are documented under [Authentication](/api/authentication).

## Cluster client sync lifecycle

The `ClusterManager` reconciles every cluster record once a minute and whenever `TriggerClusterSync` is called after a change. For each cluster it builds a `ClientSet` from the configured credential source, picks the access mode, and discovers a Prometheus URL when one is available. A failed build is recoverable and is retried on the next round; deleting or disabling a cluster removes its client.

<DiagramFrame src="/diagrams/en/kube-osprey-cluster-sync.html" title="Cluster client sync lifecycle" :height="580" />

## cluster-agent registration and reverse tunnel

For clusters that the server cannot reach directly, an administrator adds the cluster in agent mode and KubeOsprey generates a manifest with a permission profile (`readOnly`, `operator`, or `custom`). The agent is the same binary run as a subcommand. On startup it registers with NaCl-box-encrypted credentials, the server verifies the registration token against the stored hash, and a `remotedialer` WebSocket becomes the reverse tunnel through which the `ClientSet` dials the cluster's API.

<DiagramFrame src="/diagrams/en/kube-osprey-cluster-agent.html" title="cluster-agent registration and reverse tunnel" :height="540" />

Setup steps and permission profiles are covered in [KubeOsprey Cluster Agent](./cluster-agent).

## AI session and tool calls

The [AI Assistant](./ai-assistant) opens an SSE stream with `POST /api/v1/ai/chat` and runs an agent loop against the configured LLM provider. Each tool call the model proposes goes through `AuthorizeToolForUser`, which maps it onto the requesting user's own RBAC permissions, so the assistant can never do more than the person using it. Mutating tools additionally pause for a confirmation card in the chat; once confirmed, the tool executes through the cluster `ClientSet`, sensitive fields are redacted from the result, and the change is written to [Resource History](./resource-history).

<DiagramFrame src="/diagrams/en/kube-osprey-ai-toolcall.html" title="AI session and tool calls" :height="500" />

## Monitoring metrics data flow

Metrics for the [Monitoring](./monitoring) views come from the cluster's Prometheus, which KubeOsprey discovers automatically or takes from [Prometheus Setup](/config/prometheus-setup). Queries are routed per cluster: in-cluster URLs go through the Kubernetes API proxy, agent-mode clusters dial through the tunnel, and everything else connects directly. `pkg/prometheus` issues `Query` and `QueryRange` calls, `pkg/metrics` exposes them under `/api/v1/prometheus/*` with RBAC filtering, and live Pod metrics fall back to `metrics-server` when Prometheus is unavailable.

<DiagramFrame src="/diagrams/en/kube-osprey-metrics-dataflow.html" title="Monitoring metrics data flow" :height="470" />

## Build and delivery pipeline

`make build` runs `pnpm build` in `ui/`, which emits the Vite bundle into `static/`; `static.go` embeds that directory with `//go:embed` and `go build` produces the `kube-osprey` binary with `CGO_ENABLED=0`, `-trimpath`, and version information injected via `ldflags`. The same binary is packaged as the container image and deployed by the Helm chart. `make dev` builds the backend, runs it in the background with `-v=5`, and starts the Vite dev server, which proxies `/api` to it with hot reload.

<DiagramFrame src="/diagrams/en/kube-osprey-build-pipeline.html" title="Build and delivery pipeline" :height="460" />

See [Getting Started](./installation) for the Helm installation path.

## Scope and maintenance

These diagrams cover the core request path, cluster access, the AI assistant, metrics, and the build. Feature modules such as the [Alert Center](./alert-center), [Log Center](./log-center), [Change Plans](./change-plans), and the [MCP Server](./mcp-server) sit on the same Gin → `ClusterManager` path and are described in their own guides.

Each diagram is generated from a spec in [`docs/diagrams/`](https://github.com/fadewalk/kube-osprey/tree/main/docs/diagrams) with [Archify](https://github.com/tt-a1i/archify). When the code structure changes, edit the spec and regenerate the HTML; the README in that directory lists the commands.

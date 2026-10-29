# Architecture diagram specs

Source specs for the interactive diagrams on the Architecture page
(`docs/guide/architecture.md` and `docs/zh/guide/architecture.md`).
Rendered with [Archify](https://github.com/tt-a1i/archify); the generated
HTML lives in `docs/public/diagrams/{en,zh}/` and is embedded through the
`DiagramFrame` component in `docs/.vitepress/theme/components/`.

Edit the JSON here, never the generated HTML.

## Layout

```
docs/diagrams/en/<name>.<type>.json   English specs (locale "en")
docs/diagrams/zh/<name>.<type>.json   Chinese specs (locale "zh-CN")
docs/public/diagrams/en/<name>.html   generated, served at /diagrams/en/<name>.html
docs/public/diagrams/zh/<name>.html   generated, served at /diagrams/zh/<name>.html
```

Both locales share the same layout values (positions, sizes, columns,
`viewBox`); only display text differs. Code identifiers, paths, and endpoints
are kept verbatim in both.

| Spec                                  | Type         | Shows                                                    |
| ------------------------------------- | ------------ | -------------------------------------------------------- |
| `kube-osprey-architecture.architecture.json` | architecture | SPA → Gin middleware → resources → ClusterManager → K8s  |
| `kube-osprey-request-sequence.sequence.json` | sequence     | Auth, RBAC, informer cache, silent JWT refresh           |
| `kube-osprey-cluster-sync.lifecycle.json`    | lifecycle    | ClusterManager reconcile loop and ClientSet states       |
| `kube-osprey-cluster-agent.workflow.json`    | workflow     | Agent manifest, NaCl registration, remotedialer tunnel   |
| `kube-osprey-ai-toolcall.sequence.json`      | sequence     | SSE chat, tool authorization, confirmation, history      |
| `kube-osprey-metrics-dataflow.dataflow.json` | dataflow     | Prometheus routing, pkg/metrics, metrics-server fallback |
| `kube-osprey-build-pipeline.workflow.json`   | workflow     | pnpm → static/ → go:embed → binary → Helm, plus make dev |

## Regenerating

The Archify CLI is distributed as a skill (`~/.zcode/skills/archify`). From
the repository root:

```bash
ARCHIFY=~/.zcode/skills/archify/bin/archify.mjs

# validate one spec (architecture type also checks referenced repo paths)
node "$ARCHIFY" validate sequence docs/diagrams/en/kube-osprey-request-sequence.sequence.json \
  --quality showcase --json
node "$ARCHIFY" validate architecture docs/diagrams/en/kube-osprey-architecture.architecture.json \
  --quality showcase --json --repo-root .

# render one spec
node "$ARCHIFY" deliver sequence docs/diagrams/en/kube-osprey-request-sequence.sequence.json \
  docs/public/diagrams/en/kube-osprey-request-sequence.html --quality showcase --json
```

`deliver` fails if the showcase quality gate does not pass, so a successful
run is also a validation. Regenerate both locales whenever a shared layout
value changes.

## Text width budgets

Showcase quality rejects labels wider than their node. These limits were
measured against the current layouts and apply to the English specs in
particular (Chinese text is roughly half the width per character):

| Diagram type                   | Node width | Label          | Sublabel   |
| ------------------------------ | ---------- | -------------- | ---------- |
| workflow (`width: 132`)        | 132 px     | ≤ 19 chars     | ≤ 27 chars |
| lifecycle (default)            | 118 px     | ≤ 17 chars     | ≤ 24 chars |
| dataflow (default)             | 112 px     | ≤ 16 chars     | fixed 7 px |
| sequence participant           | 113 px     | ≤ 16 chars     | ≤ 20 chars |
| architecture (`size` per node) | 150–170 px | fits `size[0]` | ≤ 28 chars |

Node sublabels in workflow and lifecycle diagrams shrink to fit; the gate
fails when the projected size at a 1440 px viewport drops below 6 px, which
is what the sublabel column above protects against. Connection labels in the
architecture diagram must also fit the gap between the two components they
join (`labelDy` moves a label off the line when it cannot).

## Fact baseline

Facts were verified against the `rebrand/drop-kite-identifiers` branch
(2026-10-29): middleware and handler names, `/api/v1/ai/*` and
`/api/v1/prometheus/*` routes, the `kubeosprey<ID>-<SECRET>` API key format,
JWT 24 h / cookie 48 h, the one-minute cluster reconcile,
`AuthorizeToolForUser`, permission profile names, the `kube-osprey` binary
name, and the `ghcr.io/fadewalk/kube-osprey` image. When any of these change,
update the spec and the surrounding prose in the Architecture pages together.

import { useEffect, useMemo, useState } from 'react'
import { IconChartInfographic, IconRefresh } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

import { apiClient } from '@/lib/api-client'
import {
  getCurrentCluster,
  withCurrentClusterPath,
} from '@/lib/current-cluster'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface TopologyObject {
  uid: string
  kind: string
  name: string
  namespace: string
  resource: string
  ownerUids: string[]
  createdAt: string
  placeholder?: boolean
}

interface TopologyEdge {
  from: string
  to: string
  type: 'selector' | 'config' | 'storage'
}

interface TopologyResponse {
  objects: TopologyObject[]
  edges: TopologyEdge[]
  available: boolean
}

type RelationFilter = 'all' | 'selector' | 'config' | 'storage'

const EDGE_STYLE: Record<
  TopologyEdge['type'],
  { stroke: string; dash: string }
> = {
  selector: { stroke: '#f97316', dash: '6 3' },
  config: { stroke: '#a855f7', dash: '3 3' },
  storage: { stroke: '#14b8a6', dash: '8 4' },
}

const KIND_COLOR: Record<string, string> = {
  Deployment: 'border-blue-500/50 bg-blue-500/10',
  StatefulSet: 'border-violet-500/50 bg-violet-500/10',
  DaemonSet: 'border-cyan-500/50 bg-cyan-500/10',
  ReplicaSet: 'border-zinc-400/50 bg-zinc-400/10',
  Job: 'border-amber-500/50 bg-amber-500/10',
  Pod: 'border-green-500/50 bg-green-500/10',
  Service: 'border-orange-500/50 bg-orange-500/10',
  ConfigMap: 'border-purple-500/50 bg-purple-500/10',
  Secret: 'border-rose-500/50 bg-rose-500/10',
  PersistentVolumeClaim: 'border-teal-500/50 bg-teal-500/10',
}

function resourceUrl(obj: TopologyObject): string | null {
  if (!obj.resource) {
    return null
  }
  if (obj.namespace) {
    return `/${obj.resource}/${obj.namespace}/${obj.name}`
  }
  return `/${obj.resource}/${obj.name}`
}

interface LayoutNode {
  obj: TopologyObject
  depth: number
  x: number
  y: number
}

const NODE_W = 180
const NODE_H = 44
const COL_GAP = 70
const ROW_GAP = 22

// buildLayout assigns each object a layer by its longest owner chain within
// the visible set, then stacks layers left-to-right. Placeholder reference
// nodes (unsynced Secrets etc.) are pinned to a rightmost refs column.
function buildLayout(objects: TopologyObject[]): LayoutNode[] {
  const byUid = new Map(objects.map((o) => [o.uid, o]))
  const depthCache = new Map<string, number>()

  const depthOf = (uid: string, seen: Set<string>): number => {
    if (depthCache.has(uid)) return depthCache.get(uid)!
    if (seen.has(uid)) return 0
    seen.add(uid)
    const obj = byUid.get(uid)
    if (!obj) return 0
    let parentDepth = -1
    for (const owner of obj.ownerUids) {
      if (byUid.has(owner)) {
        parentDepth = Math.max(parentDepth, depthOf(owner, seen))
      }
    }
    const depth = parentDepth + 1
    depthCache.set(uid, depth)
    return depth
  }

  const nodes: LayoutNode[] = objects.map((obj) => ({
    obj,
    depth: depthOf(obj.uid, new Set()),
    x: 0,
    y: 0,
  }))

  const maxDepth = nodes.reduce((m, n) => Math.max(m, n.depth), 0)
  // Placeholders reference pods from anywhere in the graph; park them all in
  // one dedicated rightmost column so their edges read as "points at refs".
  const refsDepth = maxDepth + 1
  const columns: LayoutNode[][] = Array.from({ length: maxDepth + 2 }, () => [])
  // Services have no owner edges: pin them to the leftmost layer.
  for (const node of nodes) {
    if (node.obj.placeholder) {
      node.depth = refsDepth
    } else if (node.obj.kind === 'Service') {
      node.depth = node.obj.ownerUids.length > 0 ? node.depth : 0
    }
    columns[Math.min(node.depth, refsDepth)].push(node)
  }

  let width = 0
  columns.forEach((column, depth) => {
    column.sort((a, b) => a.obj.name.localeCompare(b.obj.name))
    const x = depth * (NODE_W + COL_GAP)
    column.forEach((node, row) => {
      node.x = x
      node.y = row * (NODE_H + ROW_GAP)
    })
    width = Math.max(width, x + NODE_W)
  })

  return nodes
}

function TopologyGraph({
  objects,
  relations,
  relationFilter,
}: {
  objects: TopologyObject[]
  relations: TopologyEdge[]
  relationFilter: RelationFilter
}) {
  const { t } = useTranslation()
  const nodes = useMemo(() => buildLayout(objects), [objects])
  const byUid = useMemo(
    () => new Map(nodes.map((n) => [n.obj.uid, n])),
    [nodes]
  )

  const height = nodes.reduce((m, n) => Math.max(m, n.y + NODE_H), 0) + 40
  const width = nodes.reduce((m, n) => Math.max(m, n.x + NODE_W), 0) + 40

  const edges = useMemo(() => {
    const out: { from: LayoutNode; to: LayoutNode }[] = []
    for (const node of nodes) {
      for (const owner of node.obj.ownerUids) {
        const parent = byUid.get(owner)
        if (parent) {
          out.push({ from: parent, to: node })
        }
      }
    }
    return out
  }, [nodes, byUid])

  const relationEdges = useMemo(() => {
    const out: {
      from: LayoutNode
      to: LayoutNode
      type: TopologyEdge['type']
    }[] = []
    for (const rel of relations) {
      if (relationFilter !== 'all' && rel.type !== relationFilter) {
        continue
      }
      const from = byUid.get(rel.from)
      const to = byUid.get(rel.to)
      if (from && to) {
        out.push({ from, to, type: rel.type })
      }
    }
    return out
  }, [relations, relationFilter, byUid])

  if (nodes.length === 0) {
    return (
      <div className="flex items-center justify-center rounded-lg border p-10 text-sm text-muted-foreground">
        {t('topology.empty', 'No objects to display in this namespace.')}
      </div>
    )
  }

  return (
    <div className="overflow-auto rounded-lg border bg-muted/20 p-4">
      <svg width={width} height={height} className="min-w-full">
        {edges.map((edge, i) => {
          const x1 = edge.from.x + NODE_W
          const y1 = edge.from.y + NODE_H / 2
          const x2 = edge.to.x
          const y2 = edge.to.y + NODE_H / 2
          const mid = (x1 + x2) / 2
          return (
            <path
              key={i}
              d={`M ${x1} ${y1} C ${mid} ${y1}, ${mid} ${y2}, ${x2} ${y2}`}
              fill="none"
              stroke="currentColor"
              className="text-zinc-400/60"
              strokeWidth={1.5}
            />
          )
        })}
        {relationEdges.map((edge, i) => {
          const x1 = edge.from.x + NODE_W
          const y1 = edge.from.y + NODE_H / 2
          const x2 = edge.to.x
          const y2 = edge.to.y + NODE_H / 2
          const mid = (x1 + x2) / 2
          const style = EDGE_STYLE[edge.type]
          return (
            <path
              key={`rel-${i}`}
              d={`M ${x1} ${y1} C ${mid} ${y1}, ${mid} ${y2}, ${x2} ${y2}`}
              fill="none"
              stroke={style.stroke}
              strokeDasharray={style.dash}
              strokeWidth={1.5}
              opacity={0.75}
            />
          )
        })}
        {nodes.map((node) => {
          const url = resourceUrl(node.obj)
          return (
            <g key={node.obj.uid}>
              <foreignObject
                x={node.x}
                y={node.y}
                width={NODE_W}
                height={NODE_H}
              >
                {url ? (
                  <Link
                    to={url}
                    className={`flex h-full flex-col justify-center rounded-md border px-2 py-1 transition-shadow hover:shadow-md ${
                      KIND_COLOR[node.obj.kind] ??
                      'border-zinc-400/50 bg-zinc-400/10'
                    }`}
                  >
                    <span className="truncate text-xs font-medium leading-4">
                      {node.obj.name}
                    </span>
                    <span className="text-[10px] leading-3 text-muted-foreground">
                      {node.obj.kind}
                    </span>
                  </Link>
                ) : (
                  <div
                    className={`flex h-full flex-col justify-center rounded-md border border-dashed px-2 py-1 ${
                      KIND_COLOR[node.obj.kind] ??
                      'border-zinc-400/50 bg-zinc-400/10'
                    }`}
                  >
                    <span className="truncate text-xs font-medium leading-4">
                      {node.obj.name}
                    </span>
                    <span className="text-[10px] leading-3 text-muted-foreground">
                      {node.obj.kind}
                    </span>
                  </div>
                )}
              </foreignObject>
            </g>
          )
        })}
      </svg>
    </div>
  )
}

export function TopologyPage() {
  const { t } = useTranslation()
  const [namespace, setNamespace] = useState<string>('default')
  const [relationFilter, setRelationFilter] = useState<RelationFilter>('all')
  const cluster = getCurrentCluster()

  const { data: namespaces } = useQuery({
    queryKey: ['namespaces', cluster],
    queryFn: async () => {
      const resp = await apiClient.get<{
        items: { metadata: { name: string } }[]
      }>(withCurrentClusterPath('/namespaces', cluster))
      return resp.items.map((i) => i.metadata.name).sort()
    },
  })

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['topology', cluster, namespace],
    queryFn: async (): Promise<TopologyResponse> => {
      const path = withCurrentClusterPath(
        `/topology?namespace=${encodeURIComponent(namespace)}`,
        cluster
      )
      return apiClient.get<TopologyResponse>(path)
    },
    refetchInterval: 30000,
  })

  const [tick, setTick] = useState(0)
  useEffect(() => {
    const timer = setInterval(() => setTick((v) => v + 1), 60000)
    return () => clearInterval(timer)
  }, [])

  return (
    <div className="space-y-4 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <IconChartInfographic className="size-5" />
          <h1 className="text-lg font-bold">
            {t('topology.title', 'Topology')}
          </h1>
          {data && !data.available && (
            <Badge variant="outline" className="bg-amber-500/15 text-amber-600">
              {t(
                'topology.syncUnavailable',
                'Sync engine data unavailable — enable it in Settings → General'
              )}
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Select value={namespace} onValueChange={setNamespace}>
            <SelectTrigger className="w-52">
              <SelectValue
                placeholder={t('common.fields.namespace', 'Namespace')}
              />
            </SelectTrigger>
            <SelectContent>
              {(namespaces ?? ['default']).map((ns) => (
                <SelectItem key={ns} value={ns}>
                  {ns}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button variant="outline" size="icon" onClick={() => refetch()}>
            <IconRefresh className="size-4" />
          </Button>
        </div>
      </div>

      <p className="text-xs text-muted-foreground" key={tick}>
        {t(
          'topology.hint',
          'Ownership graph from the sync engine: workloads point to their ReplicaSets and pods. Click a node to open the resource.'
        )}
      </p>

      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <span className="inline-block h-0.5 w-6 bg-zinc-400" />
            {t('topology.relations.owner', 'Ownership')}
          </span>
          <span className="flex items-center gap-1">
            <span
              className="inline-block h-0.5 w-6"
              style={{
                background: `repeating-linear-gradient(90deg, ${EDGE_STYLE.selector.stroke} 0 6px, transparent 6px 9px)`,
              }}
            />
            {t('topology.relations.selector', 'Service selector')}
          </span>
          <span className="flex items-center gap-1">
            <span
              className="inline-block h-0.5 w-6"
              style={{
                background: `repeating-linear-gradient(90deg, ${EDGE_STYLE.config.stroke} 0 3px, transparent 3px 6px)`,
              }}
            />
            {t('topology.relations.config', 'Config / env reference')}
          </span>
          <span className="flex items-center gap-1">
            <span
              className="inline-block h-0.5 w-6"
              style={{
                background: `repeating-linear-gradient(90deg, ${EDGE_STYLE.storage.stroke} 0 8px, transparent 8px 12px)`,
              }}
            />
            {t('topology.relations.storage', 'Volume claim')}
          </span>
        </div>
        <Select
          value={relationFilter}
          onValueChange={(v) => setRelationFilter(v as RelationFilter)}
        >
          <SelectTrigger className="w-44">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('topology.relations.all')}</SelectItem>
            <SelectItem value="selector">
              {t('topology.relations.selector')}
            </SelectItem>
            <SelectItem value="config">
              {t('topology.relations.config')}
            </SelectItem>
            <SelectItem value="storage">
              {t('topology.relations.storage')}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center rounded-lg border p-10 text-sm text-muted-foreground">
          Loading...
        </div>
      ) : (
        data && (
          <TopologyGraph
            objects={data.objects}
            relations={data.edges ?? []}
            relationFilter={relationFilter}
          />
        )
      )}
    </div>
  )
}

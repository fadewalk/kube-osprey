import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'

import { apiClient } from '@/lib/api-client'
import { setCurrentCluster } from '@/lib/current-cluster'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

interface ClusterSummary {
  name: string
  status: string
  error?: string
  version?: string
  totalNodes: number
  readyNodes: number
  totalPods: number
  runningPods: number
  totalNamespaces: number
  totalServices: number
}

export function MultiClusterOverviewPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['multi-cluster-overview'],
    queryFn: () => apiClient.get<ClusterSummary[]>('/clusters/overview'),
    refetchInterval: 30_000,
  })

  const openCluster = (name: string) => {
    setCurrentCluster(name)
    navigate('/')
  }

  return (
    <div className="space-y-6 p-4">
      <div>
        <h1 className="text-2xl font-bold">{t('nav.multiCluster')}</h1>
      </div>

      {isLoading && (
        <div className="flex items-center justify-center py-16">
          <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-primary"></div>
        </div>
      )}

      {isError && (
        <p role="alert" className="text-destructive text-sm">
          {(error as Error)?.message}
        </p>
      )}

      {data && (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {data.map((cluster) => (
            <button
              key={cluster.name}
              onClick={() => openCluster(cluster.name)}
              className="bg-card text-card-foreground rounded-lg border p-4 text-left shadow-sm transition-colors hover:border-primary/50"
            >
              <div className="flex items-center justify-between gap-2">
                <span className="font-medium">{cluster.name}</span>
                {cluster.status === 'ok' ? (
                  <Badge
                    variant="outline"
                    className="text-muted-foreground px-1.5"
                  >
                    {cluster.version}
                  </Badge>
                ) : (
                  <Badge variant="outline" className="text-destructive px-1.5">
                    {t('multiCluster.unreachable')}
                  </Badge>
                )}
              </div>
              {cluster.status !== 'ok' && (
                <p className="text-destructive mt-2 text-xs">{cluster.error}</p>
              )}
              <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                <div className="flex justify-between gap-2">
                  <dt className="text-muted-foreground">{t('nav.nodes')}</dt>
                  <dd className="tabular-nums">
                    {cluster.readyNodes} / {cluster.totalNodes}
                  </dd>
                </div>
                <div className="flex justify-between gap-2">
                  <dt className="text-muted-foreground">{t('nav.pods')}</dt>
                  <dd className="tabular-nums">
                    {cluster.runningPods} / {cluster.totalPods}
                  </dd>
                </div>
                <div className="flex justify-between gap-2">
                  <dt className="text-muted-foreground">
                    {t('nav.namespaces')}
                  </dt>
                  <dd className="tabular-nums">{cluster.totalNamespaces}</dd>
                </div>
                <div className="flex justify-between gap-2">
                  <dt className="text-muted-foreground">{t('nav.services')}</dt>
                  <dd className="tabular-nums">{cluster.totalServices}</dd>
                </div>
              </dl>
              <div className="mt-3">
                <Button variant="outline" size="sm" className="w-full">
                  {t('multiCluster.openCluster')}
                </Button>
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

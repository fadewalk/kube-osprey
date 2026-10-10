import { apiClient } from '../api-client'
import { getCurrentCluster, withCurrentClusterPath } from '../current-cluster'

export interface WorkloadOpRequest {
  kind: 'deployments' | 'statefulsets' | 'daemonsets'
  namespace: string
  name: string
  container?: string
  image?: string
}

const post = (op: string, body: WorkloadOpRequest) =>
  apiClient.post<{ status: string }>(
    withCurrentClusterPath(`/workload-ops/${op}`, getCurrentCluster()),
    body
  )

export const updateWorkloadImage = (req: WorkloadOpRequest) =>
  post('update-image', { ...req, image: req.image ?? '' })

export const restartWorkload = (req: WorkloadOpRequest) => post('restart', req)

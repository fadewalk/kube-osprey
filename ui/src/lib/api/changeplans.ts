import { useQuery } from '@tanstack/react-query'

import { apiClient } from '../api-client'

export interface ChangePlanStep {
  id: number
  planId: number
  seq: number
  toolName: string
  description: string
  cluster: string
  namespace: string
  resource: string
  dangerLevel: 'LOW' | 'MEDIUM' | 'HIGH'
  status: string
  argsJson: string
  beforeJson: string
  afterJson: string
  dryRunSupported: boolean
  dryRunStatus: string
  dryRunResponse: string
  applyStatus: string
  applyResponse: string
  errorMessage: string
  appliedAt: string | null
}

export interface ChangePlan {
  id: number
  userId: number
  username: string
  clientName: string
  description: string
  status:
    | 'CREATED'
    | 'PENDING'
    | 'APPROVED'
    | 'APPLYING'
    | 'APPLIED'
    | 'FAILED'
    | 'REJECTED'
    | 'EXPIRED'
  dangerLevel: 'LOW' | 'MEDIUM' | 'HIGH'
  approverName: string
  approvedAt: string | null
  expiresAt: string | null
  createdAt: string
  updatedAt: string
  steps: ChangePlanStep[]
}

export interface ChangePlanListResponse {
  plans: ChangePlan[]
  canApprove: boolean
}

export const listChangePlans = async (
  status?: string
): Promise<ChangePlanListResponse> => {
  const suffix = status ? `?status=${encodeURIComponent(status)}` : ''
  return apiClient.get<ChangePlanListResponse>(`/change-plans/${suffix}`)
}

export const getChangePlan = async (id: number): Promise<ChangePlan> => {
  return apiClient.get<ChangePlan>(`/change-plans/${id}`)
}

export const approveChangePlan = async (id: number): Promise<void> => {
  await apiClient.post(`/change-plans/${id}/approve`)
}

export const rejectChangePlan = async (id: number): Promise<void> => {
  await apiClient.post(`/change-plans/${id}/reject`)
}

export const rollbackChangePlan = async (id: number): Promise<void> => {
  await apiClient.post(`/change-plans/${id}/rollback`)
}

export const usePendingChangePlanCount = (options?: {
  enabled?: boolean
  refreshInterval?: number
}) => {
  return useQuery({
    queryKey: ['change-plans', 'pending-count'],
    queryFn: async () => {
      const resp = await apiClient.get<{ count: number }>(
        '/change-plans/pending-count'
      )
      return resp.count
    },
    enabled: options?.enabled ?? true,
    refetchInterval: options?.refreshInterval ?? 30000,
  })
}

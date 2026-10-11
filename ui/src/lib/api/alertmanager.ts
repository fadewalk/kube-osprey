import { apiClient } from '../api-client'
import { withCurrentClusterPath } from '../current-cluster'

export interface AlertManagerMatcher {
  name: string
  value: string
  isRegex: boolean
  operator?: string
}

export interface AlertManagerAlert {
  labels: Record<string, string>
  annotations: Record<string, string>
  startsAt: string
  endsAt: string
  generatorURL?: string
  fingerprint?: string
  status: {
    state: string
    silencedBy?: string[]
    inhibitedBy?: string[]
  }
}

export interface AlertManagerSilence {
  id: string
  matchers: AlertManagerMatcher[]
  startsAt: string
  endsAt: string
  createdAt?: string
  createdBy?: string
  comment?: string
  status: {
    state: string
  }
}

export interface AlertManagerStatus {
  version: string
  clusterName?: string
  peers: number
}

export interface CreateSilenceRequest {
  matchers: AlertManagerMatcher[]
  startsAt?: string
  endsAt: string
  comment: string
}

export function fetchAlerts(
  signal?: AbortSignal
): Promise<AlertManagerAlert[]> {
  return apiClient.get(withCurrentClusterPath('/alertmanager/alerts'), {
    signal,
  })
}

export function fetchSilences(
  signal?: AbortSignal
): Promise<AlertManagerSilence[]> {
  return apiClient.get(withCurrentClusterPath('/alertmanager/silences'), {
    signal,
  })
}

export function fetchAlertManagerStatus(
  signal?: AbortSignal
): Promise<AlertManagerStatus> {
  return apiClient.get(withCurrentClusterPath('/alertmanager/status'), {
    signal,
  })
}

export function createSilence(
  payload: CreateSilenceRequest
): Promise<{ silenceId: string }> {
  return apiClient.post(
    withCurrentClusterPath('/alertmanager/silences'),
    payload
  )
}

export function deleteSilence(id: string): Promise<{ message: string }> {
  return apiClient.delete(
    withCurrentClusterPath(`/alertmanager/silences/${encodeURIComponent(id)}`)
  )
}

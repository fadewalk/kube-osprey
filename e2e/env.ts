import { mkdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const e2eDir = dirname(fileURLToPath(import.meta.url))

export const authFile = resolve(e2eDir, '.auth', 'admin.json')

export const adminUser = {
  username: process.env.KUBE_OSPREY_E2E_ADMIN_USERNAME || 'admin',
  name: process.env.KUBE_OSPREY_E2E_ADMIN_NAME || 'KubeOsprey Admin',
  password: process.env.KUBE_OSPREY_E2E_ADMIN_PASSWORD || 'KubeOspreyE2E!2345',
}

export const kindClusterName =
  process.env.KUBE_OSPREY_E2E_CLUSTER_NAME || 'kube-osprey-e2e'

export function ensureAuthDir() {
  mkdirSync(dirname(authFile), { recursive: true })
}

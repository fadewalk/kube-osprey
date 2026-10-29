import { useMemo } from 'react'
import { useAuth } from '@/contexts/auth-context'
import { version as sdkVersion } from '@kite-dev/plugin-sdk/package.json'

export function useHostInfo() {
  const { appVersion } = useAuth()
  // Plugins render only after bootstrap has provided the authenticated user.
  return useMemo(
    // `kiteVersion` is the HostInfo field name defined by @kite-dev/plugin-sdk.
    () => ({ kiteVersion: appVersion!, sdkVersion }),
    [appVersion]
  )
}

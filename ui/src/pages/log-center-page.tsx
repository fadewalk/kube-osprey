import { useMemo, useState } from 'react'
import { IconFilter } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { useResources, useResourcesWatch } from '@/lib/api'
import { usePageTitle } from '@/hooks/use-page-title'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { LogViewer } from '@/components/log-viewer'

export function LogCenterPage() {
  const { t } = useTranslation()
  usePageTitle(t('nav.logCenter'))

  const { data: namespaces } = useResources('namespaces', '_all')
  const namespaceItems = useMemo(() => namespaces ?? [], [namespaces])
  const [namespace, setNamespace] = useState<string>('')
  const [selectorInput, setSelectorInput] = useState('')
  const [labelSelector, setLabelSelector] = useState('')

  const { data: pods } = useResourcesWatch('pods', namespace || undefined, {
    labelSelector: labelSelector || undefined,
    enabled: !!namespace,
  })
  const podItems = pods ?? []

  const commitSelector = () => {
    const trimmed = selectorInput.trim()
    setLabelSelector(trimmed)
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-4">
      <div>
        <h1 className="text-lg font-extrabold">{t('nav.logCenter')}</h1>
        <p className="text-sm text-muted-foreground">
          {t('logCenter.description')}
        </p>
      </div>

      <Card className="py-3">
        <CardContent className="flex flex-wrap items-end gap-3 px-4">
          <div className="grid gap-1">
            <Label htmlFor="log-center-namespace">
              {t('logCenter.namespace')}
            </Label>
            <Select
              value={namespace}
              onValueChange={(value) => {
                setNamespace(value)
                setSelectorInput('')
                setLabelSelector('')
              }}
            >
              <SelectTrigger id="log-center-namespace" className="w-56">
                <SelectValue
                  placeholder={t('logCenter.namespacePlaceholder')}
                />
              </SelectTrigger>
              <SelectContent>
                {namespaceItems.map((ns) => (
                  <SelectItem
                    key={ns.metadata?.name}
                    value={ns.metadata?.name || ''}
                  >
                    {ns.metadata?.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid min-w-0 flex-1 gap-1">
            <Label htmlFor="log-center-selector">
              {t('logCenter.selectorLabel')}
            </Label>
            <div className="relative">
              <IconFilter className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                id="log-center-selector"
                className="pl-8"
                placeholder={t('logCenter.selectorHint')}
                value={selectorInput}
                disabled={!namespace}
                onChange={(e) => setSelectorInput(e.target.value)}
                onBlur={commitSelector}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    commitSelector()
                  }
                }}
              />
            </div>
          </div>
          {namespace ? (
            <span className="pb-2 text-sm text-muted-foreground">
              {t('logCenter.podCount', { count: podItems.length })}
            </span>
          ) : null}
        </CardContent>
      </Card>

      {namespace ? (
        <div className="min-h-0 flex-1">
          <LogViewer
            namespace={namespace}
            pods={podItems}
            labelSelector={labelSelector || undefined}
            defaultPodName="_all"
          />
        </div>
      ) : (
        <Card className="flex-1">
          <CardContent className="flex h-full items-center justify-center py-16 text-sm text-muted-foreground">
            {t('logCenter.selectNamespace')}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

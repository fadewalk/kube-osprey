import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { fetchPromQL, type PromQLResponse } from '@/lib/api'
import { usePageTitle } from '@/hooks/use-page-title'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

export function PrometheusExplorerPage() {
  const { t } = useTranslation()
  const [query, setQuery] = useState('up')
  const [range, setRange] = useState(true)
  const [start, setStart] = useState('')
  const [end, setEnd] = useState('')
  const [step, setStep] = useState('60')
  const [result, setResult] = useState<PromQLResponse | null>(null)
  const mutation = useMutation({
    mutationFn: () => fetchPromQL({ query, range, start, end, step }),
    onSuccess: setResult,
  })

  usePageTitle(t('nav.prometheusExplorer'))

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-lg font-extrabold">
          {t('nav.prometheusExplorer')}
        </h1>
        <p className="text-sm text-muted-foreground">
          {t('prometheusExplorer.description')}
        </p>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>{t('prometheusExplorer.query')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Textarea
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            rows={4}
          />
          <div className="flex flex-wrap items-end gap-3">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={range}
                onChange={(event) => setRange(event.target.checked)}
              />
              {t('prometheusExplorer.rangeQuery')}
            </label>
            {range ? (
              <>
                <div className="grid gap-1">
                  <Label htmlFor="prom-start">
                    {t('prometheusExplorer.start')}
                  </Label>
                  <Input
                    id="prom-start"
                    value={start}
                    onChange={(event) => setStart(event.target.value)}
                    placeholder={t('prometheusExplorer.defaults.oneHourAgo')}
                  />
                </div>
                <div className="grid gap-1">
                  <Label htmlFor="prom-end">
                    {t('prometheusExplorer.end')}
                  </Label>
                  <Input
                    id="prom-end"
                    value={end}
                    onChange={(event) => setEnd(event.target.value)}
                    placeholder={t('prometheusExplorer.defaults.now')}
                  />
                </div>
                <div className="grid gap-1">
                  <Label htmlFor="prom-step">
                    {t('prometheusExplorer.step')}
                  </Label>
                  <Input
                    id="prom-step"
                    value={step}
                    onChange={(event) => setStep(event.target.value)}
                    className="w-24"
                  />
                </div>
              </>
            ) : null}
            <Button
              onClick={() => mutation.mutate()}
              disabled={mutation.isPending || !query.trim()}
            >
              {mutation.isPending
                ? t('common.actions.loading')
                : t('prometheusExplorer.run')}
            </Button>
          </div>
          {mutation.error ? (
            <p className="text-sm text-destructive">{mutation.error.message}</p>
          ) : null}
          {result?.warnings?.length ? (
            <div className="rounded-md border border-yellow-500/40 bg-yellow-500/10 p-3 text-sm">
              {result.warnings.join(' ')}
            </div>
          ) : null}
          {result ? (
            <pre className="max-h-[32rem] overflow-auto rounded-md bg-muted p-3 text-xs">
              {JSON.stringify(result, null, 2)}
            </pre>
          ) : null}
        </CardContent>
      </Card>
    </div>
  )
}

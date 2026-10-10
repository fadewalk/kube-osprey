import { useEffect, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { updateWorkloadImage } from '@/lib/api/workload-ops'
import { translateError } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export interface WorkloadContainer {
  name: string
  image: string
}

interface WorkloadImageDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  kind: 'deployments' | 'statefulsets' | 'daemonsets'
  namespace: string
  name: string
  containers: WorkloadContainer[]
}

export function WorkloadImageDialog({
  open,
  onOpenChange,
  kind,
  namespace,
  name,
  containers,
}: WorkloadImageDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [images, setImages] = useState<Record<string, string>>({})

  useEffect(() => {
    if (open) {
      setImages(Object.fromEntries(containers.map((c) => [c.name, c.image])))
    }
  }, [open, containers])

  const changed = containers.filter((c) => images[c.name] !== c.image)

  const mutation = useMutation({
    mutationFn: async () => {
      for (const c of changed) {
        await updateWorkloadImage({
          kind,
          namespace,
          name,
          container: c.name,
          image: images[c.name],
        })
      }
    },
    onSuccess: () => {
      toast.success(t('workloadOps.imageUpdated', { count: changed.length }))
      queryClient.invalidateQueries({ queryKey: [kind] })
      onOpenChange(false)
    },
    onError: (error) => {
      toast.error(translateError(error, t))
    },
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('workloadOps.imageTitle', { name })}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          {containers.map((container) => (
            <div key={container.name} className="space-y-1">
              <Label
                htmlFor={`image-${container.name}`}
                className="font-mono text-xs"
              >
                {container.name}
              </Label>
              <Input
                id={`image-${container.name}`}
                value={images[container.name] ?? ''}
                onChange={(e) =>
                  setImages((prev) => ({
                    ...prev,
                    [container.name]: e.target.value,
                  }))
                }
                className="font-mono text-xs"
              />
            </div>
          ))}
          <p className="text-xs text-muted-foreground">
            {t(
              'workloadOps.imageHint',
              'Changing an image triggers a rollout of the workload.'
            )}
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              {t('common.actions.cancel', 'Cancel')}
            </Button>
            <Button
              onClick={() => mutation.mutate()}
              disabled={mutation.isPending || changed.length === 0}
            >
              {t('workloadOps.applyImages', { count: changed.length })}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

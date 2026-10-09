import { useTranslation } from 'react-i18next'

import { usePendingChangePlanCount } from '@/lib/api/changeplans'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

export function ChangePlanBadge() {
  const { t } = useTranslation()
  const { data: count } = usePendingChangePlanCount()
  if (!count) return null

  const label = t('changePlans.badge.pending', { count })
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className="inline-flex shrink-0 items-center rounded-full bg-amber-500/15 px-1.5 text-xs font-medium text-amber-600 dark:text-amber-400"
          role="status"
          aria-label={label}
        >
          {count}
        </span>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )
}

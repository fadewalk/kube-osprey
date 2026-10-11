import * as yaml from 'js-yaml'
import { useTranslation } from 'react-i18next'

import { HelmReleaseHistoryItem } from '@/types/api'
import { MonacoDiffEditor } from '@/lib/monaco-loader'
import {
  defineMonacoBackgroundThemes,
  useMonacoBackgroundColor,
} from '@/lib/monaco-theme'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import { useAppearance } from './appearance-provider'

interface HelmRevisionDiffDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Older revision whose values render on the left. */
  from: HelmReleaseHistoryItem | null
  /** Newer revision whose values render on the right. */
  to: HelmReleaseHistoryItem | null
}

export function HelmRevisionDiffDialog({
  open,
  onOpenChange,
  from,
  to,
}: HelmRevisionDiffDialogProps) {
  const { t } = useTranslation()
  const { actualTheme, colorTheme } = useAppearance()
  const themeMode = actualTheme === 'dark' ? 'dark' : 'light'
  const backgroundColor = useMonacoBackgroundColor(
    '--background',
    themeMode,
    colorTheme
  )

  const dumpValues = (item: HelmReleaseHistoryItem | null) =>
    yaml.dump(item?.values ?? {}, { indent: 2, sortKeys: true })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="!max-w-6xl sm:!max-w-6xl">
        <DialogHeader>
          <DialogTitle>
            {t(
              'helm.messages.revisionDiffTitle',
              'Values diff #{{from}} → #{{to}}',
              {
                from: from?.revision ?? '?',
                to: to?.revision ?? '?',
              }
            )}
          </DialogTitle>
        </DialogHeader>
        <MonacoDiffEditor
          key={`revision-diff-${colorTheme}-${actualTheme}-${backgroundColor}`}
          height="60vh"
          language="yaml"
          original={dumpValues(from)}
          modified={dumpValues(to)}
          loading={
            <div className="text-muted-foreground flex h-full items-center justify-center">
              {t('common.messages.loading', 'Loading...')}
            </div>
          }
          beforeMount={(monaco) => {
            defineMonacoBackgroundThemes(monaco, {
              darkThemeName: `custom-dark-${colorTheme}`,
              lightThemeName: `custom-vs-${colorTheme}`,
              backgroundColor,
            })
          }}
          theme={
            actualTheme === 'dark'
              ? `custom-dark-${colorTheme}`
              : `custom-vs-${colorTheme}`
          }
          options={{
            readOnly: true,
            renderSideBySide: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            automaticLayout: true,
          }}
        />
      </DialogContent>
    </Dialog>
  )
}

export default HelmRevisionDiffDialog

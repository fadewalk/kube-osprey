import { FormEvent, useState } from 'react'
import { useAuth } from '@/contexts/auth-context'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { changeCurrentUserPassword } from '@/lib/api/auth'
import { translateError } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export function ForceChangePasswordPage() {
  const { t } = useTranslation()
  const { checkAuth } = useAuth()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    if (newPassword !== confirmPassword) {
      setError(t('login.forceChangePassword.mismatch'))
      return
    }
    setSubmitting(true)
    try {
      await changeCurrentUserPassword(currentPassword, newPassword)
      toast.success(t('login.forceChangePassword.success'))
      await checkAuth()
    } catch (err) {
      setError(translateError(err, t))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="space-y-2 text-center">
          <h1 className="text-2xl font-bold">
            {t('login.forceChangePassword.title')}
          </h1>
          <p className="text-muted-foreground text-sm">
            {t('login.forceChangePassword.description')}
          </p>
        </div>
        <form className="space-y-4" onSubmit={handleSubmit}>
          <div className="space-y-2">
            <Label htmlFor="force-current-password">
              {t('login.forceChangePassword.currentPassword')}
            </Label>
            <Input
              id="force-current-password"
              type="password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
              autoComplete="current-password"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="force-new-password">
              {t('login.forceChangePassword.newPassword')}
            </Label>
            <Input
              id="force-new-password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              autoComplete="new-password"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="force-confirm-password">
              {t('login.forceChangePassword.confirmPassword')}
            </Label>
            <Input
              id="force-confirm-password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              autoComplete="new-password"
            />
          </div>
          {error && (
            <p role="alert" className="text-destructive text-sm">
              {error}
            </p>
          )}
          <Button type="submit" className="w-full" disabled={submitting}>
            {t('login.forceChangePassword.submit')}
          </Button>
        </form>
      </div>
    </div>
  )
}

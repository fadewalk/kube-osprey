import { expect, test } from '@playwright/test'

test('kubectl terminal toggle updates the header button', async ({ page }) => {
  // KubeOsprey ships with the kubectl terminal disabled, so the cycle starts
  // by enabling it.
  await page.goto('/settings?tab=general')
  await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible()

  const kubectlSection = page
    .locator('div.rounded-lg.border')
    .filter({ has: page.getByText('Kubectl', { exact: true }) })
    .first()
  const kubectlSwitch = kubectlSection.getByRole('switch').first()

  const kubectlButton = page.getByRole('button', {
    name: 'Toggle Kubectl Terminal',
  })

  await expect(kubectlSwitch).not.toBeChecked()
  await kubectlSwitch.click()
  await expect(kubectlSwitch).toBeChecked()
  await page.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByText('General settings updated')).toBeVisible()

  await page.goto('/')
  await expect(kubectlButton).toBeVisible()

  await page.goto('/settings?tab=general')
  await expect(kubectlSwitch).toBeChecked()
  await kubectlSwitch.click()
  await expect(kubectlSwitch).not.toBeChecked()
  await page.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByText('General settings updated')).toBeVisible()

  await page.goto('/')
  await expect(kubectlButton).toHaveCount(0)

  await page.goto('/settings?tab=general')
  await expect(kubectlSwitch).not.toBeChecked()
  await kubectlSwitch.click()
  await expect(kubectlSwitch).toBeChecked()
  await page.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByText('General settings updated')).toBeVisible()

  await page.goto('/')
  await expect(kubectlButton).toBeVisible()
})

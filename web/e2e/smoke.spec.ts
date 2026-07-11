import { expect, test } from '@playwright/test'

test('login and open hosts page', async ({ page }) => {
  await page.goto('/')
  await page.getByPlaceholder('DBACK_API_TOKEN').fill('dev-token')
  await page.getByRole('button', { name: 'Connect' }).click()
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible({ timeout: 15_000 })
  await page.getByRole('link', { name: 'Hosts' }).click()
  await expect(page.getByRole('heading', { name: 'Hosts' })).toBeVisible()
})

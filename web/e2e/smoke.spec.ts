import { expect, test } from '@playwright/test'

test('login and open hosts page', async ({ page }) => {
  await page.goto('/login')
  await page.getByPlaceholder('09XXXXXXXXX').fill('09120000000')
  await page.getByPlaceholder('Password').fill('testpass12')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible({ timeout: 15_000 })
  await page.getByRole('link', { name: 'Hosts' }).click()
  await expect(page.getByRole('heading', { name: 'Hosts' })).toBeVisible()
})

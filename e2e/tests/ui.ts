import { expect } from '@playwright/test'
import type { Locator, Page } from '@playwright/test'

export async function loginViaUi(page: Page, email: string, password: string) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Пароль').fill(password)
  await page.getByRole('button', { name: 'Войти' }).click()
  await expect(page.getByRole('heading', { name: 'Проекты' })).toBeVisible()
}

export function column(page: Page, name: string): Locator {
  return page.getByRole('region', { name: `Колонка ${name}` })
}

export function card(page: Page, title: string): Locator {
  return page.getByTestId('card').filter({ hasText: title })
}

/**
 * Перетаскивает карточку указателем с промежуточными шагами: dnd-kit начинает
 * перетаскивание только после смещения на несколько пикселей.
 */
export async function drag(page: Page, source: Locator, target: Locator) {
  const from = await source.boundingBox()
  const to = await target.boundingBox()
  if (!from || !to) throw new Error('element is not visible')
  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2)
  await page.mouse.down()
  await page.mouse.move(from.x + from.width / 2 + 10, from.y + from.height / 2 + 10, { steps: 5 })
  await page.mouse.move(to.x + to.width / 2, to.y + Math.min(to.height - 20, 80), { steps: 20 })
  await page.mouse.up()
}

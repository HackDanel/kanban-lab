// Снимки экрана для презентации; запускается вручную: SHOTS=1 npx playwright test zz-screenshots
import { expect, test } from '@playwright/test'
import { Client, createAccount } from './api'
import { card, loginViaUi } from './ui'

test.skip(!process.env.SHOTS, 'только по запросу')

test('снимки для презентации', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager', 'Анна Смирнова')
  const dev = await createAccount(request, 'developer', 'Иван Петров')
  const client = new Client(request, owner.token)
  const project = await client.post<{ id: string }>('/projects', {
    key: process.env.SHOTS_KEY ?? 'SHOP',
    name: 'Интернет-магазин',
  })
  await client.post(`/projects/${project.id}/members`, { userId: dev.id }).catch(() => undefined)
  const board = await client.post<{ id: string; columns: { id: string; name: string }[] }>(
    `/projects/${project.id}/boards`,
    { name: 'Спринт 12' },
  )
  const col = (name: string) => board.columns.find((c) => c.name === name)!
  const bug = await client.post<{ id: string }>(`/projects/${project.id}/labels`, { name: 'bug', color: '#d32f2f' })
  const ui = await client.post<{ id: string }>(`/projects/${project.id}/labels`, { name: 'frontend', color: '#1976d2' })
  const api = await client.post<{ id: string }>(`/projects/${project.id}/labels`, { name: 'backend', color: '#2e7d32' })
  await client.patch(`/columns/${col('In Progress').id}`, { wipLimit: 3 })
  const cards: [string, string, object][] = [
    ['To Do', 'Экспорт отчёта в CSV', { priority: 'medium', labelIds: [api.id] }],
    ['To Do', 'Тёмная тема интерфейса', { priority: 'low', labelIds: [ui.id] }],
    ['To Do', 'Восстановление пароля', { priority: 'high', assigneeId: dev.id }],
    ['In Progress', 'Перетаскивание карточек', { priority: 'high', labelIds: [ui.id], assigneeId: dev.id, dueDate: '2026-10-20' }],
    ['In Progress', 'WIP-лимиты колонок', { priority: 'critical', labelIds: [api.id], assigneeId: owner.id }],
    ['In Progress', 'Ошибка 500 при загрузке файла', { priority: 'critical', labelIds: [bug.id], assigneeId: dev.id }],
    ['Done', 'Регистрация и вход', { priority: 'high', labelIds: [api.id, ui.id] }],
    ['Done', 'Поиск по карточкам', { priority: 'medium', labelIds: [api.id] }],
  ]
  for (const [column, title, extra] of cards) {
    await client.post(`/columns/${col(column).id}/cards`, { title, ...extra })
  }
  await page.setViewportSize({ width: 1440, height: 810 })
  await loginViaUi(page, owner.email, owner.password)
  await page.goto(`/boards/${board.id}`)
  await expect(card(page, 'Поиск по карточкам')).toBeVisible()
  await page.screenshot({ path: '../docs/presentation/img/board.png' })

  await card(page, 'Перетаскивание карточек').click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Новый комментарий').fill('Проверено в Chrome, Firefox и Edge')
  await dialog.getByRole('button', { name: 'Отправить' }).click()
  await expect(dialog.getByTestId('comment')).toBeVisible()
  await page.screenshot({ path: '../docs/presentation/img/card.png' })
})

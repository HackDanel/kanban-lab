import { expect, test } from '@playwright/test'
import { createAccount, createProjectWithBoard } from './api'
import { card, column, drag, loginViaUi } from './ui'

test('TC-07: WIP-лимит не даёт переместить карточку в заполненную колонку', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager')
  const { client, board, column: col } = await createProjectWithBoard(request, owner)
  await client.patch(`/columns/${col('In Progress').id}`, { wipLimit: 1 })
  await client.post(`/columns/${col('In Progress').id}/cards`, { title: 'Уже в работе' })
  await client.post(`/columns/${col('To Do').id}/cards`, { title: 'Вторая задача' })

  await loginViaUi(page, owner.email, owner.password)
  await page.goto(`/boards/${board.id}`)
  await expect(column(page, 'In Progress').getByTestId('column-count')).toHaveText('1 / 1')

  await drag(page, card(page, 'Вторая задача'), column(page, 'In Progress'))

  await expect(page.getByText(/Превышен WIP-лимит колонки «In Progress»/)).toBeVisible()
  await expect(column(page, 'To Do').getByText('Вторая задача')).toBeVisible()
  await expect(column(page, 'In Progress').getByTestId('card')).toHaveCount(1)
})

test('TC-06, TC-24: поиск и фильтры', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager')
  const { client, project, board, column: col } = await createProjectWithBoard(request, owner)
  const label = await client.post<{ id: string }>(`/projects/${project.id}/labels`, { name: 'bug', color: '#d32f2f' })
  const todo = col('To Do').id
  await client.post(`/columns/${todo}/cards`, { title: 'Форма входа', labelIds: [label.id], priority: 'high' })
  await client.post(`/columns/${todo}/cards`, { title: 'Отчёты', description: 'выгрузка в CSV' })
  await client.post(`/columns/${todo}/cards`, { title: 'Форма регистрации', assigneeId: owner.id })

  await loginViaUi(page, owner.email, owner.password)
  await page.goto(`/boards/${board.id}`)
  await expect(page.getByTestId('card')).toHaveCount(3)

  await page.getByRole('searchbox', { name: 'Поиск' }).fill('форма')
  await expect(page.getByTestId('card')).toHaveCount(2)
  await page.getByRole('searchbox', { name: 'Поиск' }).fill('csv')
  await expect(page.getByTestId('card')).toHaveCount(1)
  await expect(card(page, 'Отчёты')).toBeVisible()

  await page.getByRole('button', { name: 'Сбросить' }).click()
  await expect(page.getByTestId('card')).toHaveCount(3)

  await page.getByRole('combobox', { name: 'Метки' }).click()
  await page.getByRole('option', { name: 'bug' }).click()
  await page.keyboard.press('Escape')
  await expect(page.getByTestId('card')).toHaveCount(1)
  await expect(card(page, 'Форма входа')).toBeVisible()

  await page.getByRole('button', { name: 'Сбросить' }).click()
  await page.getByRole('combobox', { name: 'Исполнитель' }).click()
  await page.getByRole('option', { name: owner.name }).click()
  await expect(page.getByTestId('card')).toHaveCount(1)
  await expect(card(page, 'Форма регистрации')).toBeVisible()
})

test('TC-20, TC-21: редактирование карточки и комментарии', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager')
  const { client, board, column: col } = await createProjectWithBoard(request, owner)
  await client.post(`/columns/${col('To Do').id}/cards`, { title: 'Черновик' })

  await loginViaUi(page, owner.email, owner.password)
  await page.goto(`/boards/${board.id}`)
  await card(page, 'Черновик').click()

  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Название').fill('Готовая задача')
  await dialog.getByRole('combobox', { name: 'Приоритет' }).click()
  await page.getByRole('option', { name: 'Высокий' }).click()
  await dialog.getByRole('combobox', { name: 'Исполнитель' }).click()
  await page.getByRole('option', { name: owner.name }).click()
  await dialog.getByLabel('Дедлайн').fill('2027-03-01')
  await dialog.getByRole('button', { name: 'Сохранить' }).click()
  await expect(dialog.getByText('Изменения сохранены')).toBeVisible()

  await dialog.getByLabel('Новый комментарий').fill('Проверено на стенде')
  await dialog.getByRole('button', { name: 'Отправить' }).click()
  await expect(dialog.getByTestId('comment')).toContainText('Проверено на стенде')

  await dialog.getByRole('tab', { name: 'История' }).click()
  await expect(dialog.getByTestId('activity').first()).toContainText('оставил(а) комментарий')
  await expect(dialog.getByText(/изменил\(а\) название: Черновик → Готовая задача/)).toBeVisible()
  await dialog.getByRole('button', { name: 'Закрыть' }).click()

  await expect(card(page, 'Готовая задача')).toContainText('Высокий')
  await expect(card(page, 'Готовая задача')).toContainText('2027-03-01')
})

test('TC-22: вложения загружаются и скачиваются', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager')
  const { client, board, column: col } = await createProjectWithBoard(request, owner)
  await client.post(`/columns/${col('To Do').id}/cards`, { title: 'С файлом' })

  await loginViaUi(page, owner.email, owner.password)
  await page.goto(`/boards/${board.id}`)
  await card(page, 'С файлом').click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('tab', { name: /Вложения/ }).click()
  await dialog.getByLabel('Файл вложения').setInputFiles({
    name: 'notes.txt',
    mimeType: 'text/plain',
    buffer: Buffer.from('содержимое файла'),
  })
  await expect(dialog.getByText('notes.txt')).toBeVisible()

  const download = page.waitForEvent('download')
  await dialog.getByRole('button', { name: 'Скачать notes.txt' }).click()
  const file = await download
  expect(file.suggestedFilename()).toBe('notes.txt')

  await dialog.getByRole('button', { name: 'Удалить notes.txt' }).click()
  await expect(dialog.getByText('Вложений пока нет.')).toBeVisible()
})

test('TC-25: разметка в названии не выполняется', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager')
  const { client, board, column: col } = await createProjectWithBoard(request, owner)
  const payload = '<img src=x onerror="window.__xss=1">'
  await client.post(`/columns/${col('To Do').id}/cards`, { title: payload, description: payload })

  let dialogs = 0
  page.on('dialog', async (d) => {
    dialogs++
    await d.dismiss()
  })
  await loginViaUi(page, owner.email, owner.password)
  await page.goto(`/boards/${board.id}`)
  await expect(card(page, payload)).toBeVisible()
  await card(page, payload).click()
  await expect(page.getByRole('dialog').getByLabel('Описание')).toHaveValue(payload)

  expect(await page.evaluate(() => (window as unknown as { __xss?: number }).__xss)).toBeUndefined()
  expect(dialogs).toBe(0)
})

test('TC-27: изменения видны в другой вкладке без перезагрузки', async ({ browser, request }) => {
  const owner = await createAccount(request, 'manager')
  const { board } = await createProjectWithBoard(request, owner)

  const first = await browser.newPage()
  const second = await browser.newPage()
  for (const p of [first, second]) {
    await loginViaUi(p, owner.email, owner.password)
    await p.goto(`/boards/${board.id}`)
    await expect(column(p, 'To Do')).toBeVisible()
  }

  await first.getByRole('button', { name: 'Добавить карточку в колонку To Do' }).click()
  await first.getByRole('textbox', { name: 'Название карточки' }).fill('Живая карточка')
  await first.getByRole('button', { name: 'Добавить', exact: true }).click()

  await expect(card(second, 'Живая карточка')).toBeVisible()

  await drag(first, card(first, 'Живая карточка'), column(first, 'Done'))
  await expect(column(second, 'Done').getByText('Живая карточка')).toBeVisible()
  await first.close()
  await second.close()
})

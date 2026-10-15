import { expect, test } from '@playwright/test'
import { Client, createAccount, createProjectWithBoard } from './api'
import { card, column, loginViaUi } from './ui'

// E2E-02, TC-04: права доступа разработчика.
test('разработчик работает с карточками, но не управляет проектом', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager')
  const developer = await createAccount(request, 'developer')
  const { project, board } = await createProjectWithBoard(request, owner, [developer])

  await loginViaUi(page, developer.email, developer.password)
  await expect(page.getByRole('button', { name: 'Создать проект' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: /Удалить проект/ })).toHaveCount(0)

  await page.getByRole('heading', { name: project.name }).click()
  await expect(page.getByRole('button', { name: 'Создать доску' })).toHaveCount(0)
  await expect(page.getByLabel('Добавить участника')).toHaveCount(0)

  await page.getByText('Спринт').click()
  await expect(page.getByRole('button', { name: /Действия с колонкой/ })).toHaveCount(0)
  await page.getByRole('button', { name: 'Добавить карточку в колонку To Do' }).click()
  await page.getByRole('textbox', { name: 'Название карточки' }).fill('Задача разработчика')
  await page.getByRole('button', { name: 'Добавить', exact: true }).click()
  await expect(column(page, 'To Do').getByText('Задача разработчика')).toBeVisible()

  // Сервер тоже запрещает, а не только интерфейс.
  const api = new Client(request, developer.token)
  expect((await api.raw('DELETE', `/projects/${project.id}`)).status()).toBe(403)
  expect((await api.raw('DELETE', `/boards/${board.id}`)).status()).toBe(403)
})

test('TC-16: чужой проект недоступен', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager')
  const outsider = await createAccount(request, 'developer')
  const { project, board } = await createProjectWithBoard(request, owner)

  await loginViaUi(page, outsider.email, outsider.password)
  await expect(page.getByText(project.name)).toHaveCount(0)
  await page.goto(`/boards/${board.id}`)
  await expect(page.getByText('Недостаточно прав')).toBeVisible()
  await expect(card(page, 'x')).toHaveCount(0)
})

test('TC-09: неверный пароль', async ({ page, request }) => {
  const user = await createAccount(request, 'developer')
  await page.goto('/login')
  await page.getByLabel('Email').fill(user.email)
  await page.getByLabel('Пароль').fill('wrong-password')
  await page.getByRole('button', { name: 'Войти' }).click()
  await expect(page.getByRole('alert')).toHaveText('Неверный email или пароль')
})

test('TC-11: сессия сохраняется после перезагрузки', async ({ page, request }) => {
  const user = await createAccount(request, 'developer')
  await loginViaUi(page, user.email, user.password)
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Проекты' })).toBeVisible()
})

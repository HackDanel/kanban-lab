import { expect, test } from '@playwright/test'
import { ADMIN, createAccount, createProjectWithBoard, login, setRole, unique } from './api'
import { card, column, drag, loginViaUi } from './ui'

// E2E-01: регистрация → вход → проект → доска → карточки → перемещение → выход.
test('полный сценарий работы менеджера', async ({ page, request }) => {
  const email = `${unique('e2e')}@example.com`
  const name = unique('Менеджер')

  // TC-01
  await page.goto('/register')
  await page.getByLabel('Имя').fill(name)
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Пароль').fill('password123')
  await page.getByRole('button', { name: 'Зарегистрироваться' }).click()
  await expect(page).toHaveURL(/\/login$/)
  await expect(page.getByText('Регистрация завершена')).toBeVisible()

  // Создавать проекты может менеджер; роль выдаёт администратор.
  const users = await request.get('/api/v1/users', {
    params: { q: email },
    headers: { Authorization: `Bearer ${await login(request, ADMIN.email, ADMIN.password)}` },
  })
  const { items } = (await users.json()) as { items: { id: string }[] }
  await setRole(request, items[0].id, 'manager')

  await loginViaUi(page, email, 'password123')
  await expect(page.getByText('Менеджер', { exact: true })).toBeVisible()

  // Проект
  const projectName = unique('Сайт')
  await page.getByRole('button', { name: 'Создать проект' }).click()
  await page.getByLabel('Ключ').fill('W' + Date.now().toString(36).toUpperCase().slice(-6).replace(/[^A-Z]/g, 'X'))
  await page.getByLabel('Название').fill(projectName)
  await page.getByRole('button', { name: 'Создать', exact: true }).click()
  await page.getByRole('heading', { name: projectName }).click()

  // Доска
  await page.getByLabel('Новая доска').fill('Спринт 1')
  await page.getByRole('button', { name: 'Создать доску' }).click()
  await page.getByText('Спринт 1').click()
  await expect(page.getByRole('heading', { name: 'Спринт 1' })).toBeVisible()
  await expect(column(page, 'To Do')).toBeVisible()
  await expect(column(page, 'In Progress')).toBeVisible()
  await expect(column(page, 'Done')).toBeVisible()

  // TC-02: карточки
  for (const title of ['Вёрстка главной', 'API авторизации']) {
    await page.getByRole('button', { name: 'Добавить карточку в колонку To Do' }).click()
    await page.getByRole('textbox', { name: 'Название карточки' }).fill(title)
    await page.getByRole('button', { name: 'Добавить', exact: true }).click()
    await expect(card(page, title)).toBeVisible()
  }

  // TC-03: перетаскивание и сохранение после перезагрузки
  await drag(page, card(page, 'Вёрстка главной'), column(page, 'In Progress'))
  await expect(column(page, 'In Progress').getByText('Вёрстка главной')).toBeVisible()
  await page.reload()
  await expect(column(page, 'In Progress').getByText('Вёрстка главной')).toBeVisible()

  // История фиксирует перемещение
  await card(page, 'Вёрстка главной').click()
  await page.getByRole('tab', { name: 'История' }).click()
  await expect(page.getByText(/переместил\(а\) из «To Do» в «In Progress»/)).toBeVisible()
  await page.getByRole('button', { name: 'Закрыть' }).click()

  // TC-12: выход
  await page.getByRole('button', { name: 'Выйти' }).click()
  await expect(page.getByRole('heading', { name: 'Вход' })).toBeVisible()
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Вход' })).toBeVisible()
})

test('TC-05: карточку без названия создать нельзя', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager')
  const { board } = await createProjectWithBoard(request, owner)
  await loginViaUi(page, owner.email, owner.password)
  await page.goto(`/boards/${board.id}`)

  await page.getByRole('button', { name: 'Добавить карточку в колонку To Do' }).click()
  await page.getByRole('button', { name: 'Добавить', exact: true }).click()
  await expect(page.getByText('Название обязательно')).toBeVisible()
  await expect(page.getByTestId('card')).toHaveCount(0)
})

import { expect, test } from '@playwright/test'
import { createAccount, createProjectWithBoard } from './api'
import { column, loginViaUi } from './ui'

const widths = [375, 768, 1280]

async function pageOverflow(page: import('@playwright/test').Page) {
  return page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
}

// TC-30: на телефоне, планшете и десктопе страница не прокручивается по горизонтали;
// колонки доски прокручиваются внутри своего контейнера.
for (const width of widths) {
  test(`TC-30: адаптивная вёрстка при ширине ${width}px`, async ({ page, request }) => {
    const owner = await createAccount(request, 'manager')
    const { client, board, column: col } = await createProjectWithBoard(request, owner)
    await client.post(`/columns/${col('To Do').id}/cards`, { title: 'Карточка с довольно длинным названием для проверки переноса строк' })
    await page.setViewportSize({ width, height: 800 })

    await page.goto('/login')
    expect(await pageOverflow(page)).toBeLessThanOrEqual(0)

    await loginViaUi(page, owner.email, owner.password)
    expect(await pageOverflow(page)).toBeLessThanOrEqual(0)

    await page.goto(`/boards/${board.id}`)
    await expect(column(page, 'To Do')).toBeVisible()
    expect(await pageOverflow(page)).toBeLessThanOrEqual(0)
    await expect(page.getByRole('button', { name: 'Добавить карточку в колонку To Do' })).toBeVisible()
    await page.screenshot({ path: `test-results/usability-board-${width}.png`, fullPage: true })
  })
}

test('форма входа работает с клавиатуры', async ({ page, request }) => {
  const user = await createAccount(request, 'developer')
  await page.goto('/login')
  await page.keyboard.type(user.email) // поле email получает фокус автоматически
  await page.keyboard.press('Tab')
  await page.keyboard.type(user.password)
  await page.keyboard.press('Enter')
  await expect(page.getByRole('heading', { name: 'Проекты' })).toBeVisible()
})

test('карточку можно открыть с клавиатуры', async ({ page, request }) => {
  const owner = await createAccount(request, 'manager')
  const { client, board, column: col } = await createProjectWithBoard(request, owner)
  await client.post(`/columns/${col('To Do').id}/cards`, { title: 'Клавиатура' })
  await loginViaUi(page, owner.email, owner.password)
  await page.goto(`/boards/${board.id}`)
  await page.getByTestId('card').focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
})

import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router'
import { describe, expect, it } from 'vitest'
import { makeUser } from '../test/fixtures'
import { anonymous, renderWithProviders } from '../test/render'
import { api, server } from '../test/server'
import { LoginPage } from './LoginPage'
import { RegisterPage } from './RegisterPage'

const routes = (
  <Routes>
    <Route path="/register" element={<RegisterPage />} />
    <Route path="/login" element={<LoginPage />} />
  </Routes>
)

async function fillAndSubmit(name: string, email: string, password: string) {
  const user = userEvent.setup()
  if (name) await user.type(screen.getByLabelText(/Имя/), name)
  if (email) await user.type(screen.getByLabelText(/Email/), email)
  if (password) await user.type(screen.getByLabelText(/Пароль/), password)
  await user.click(screen.getByRole('button', { name: 'Зарегистрироваться' }))
}

const start = () => renderWithProviders(routes, { route: '/register', preloadedState: anonymous })

describe('RegisterPage', () => {
  it('TC-01: регистрирует пользователя и перенаправляет на /login', async () => {
    let body: unknown
    server.use(
      http.post(api('/auth/register'), async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(makeUser(), { status: 201 })
      }),
    )
    start()

    await fillAndSubmit('Иван', 'ivan@example.com', 'password1')

    expect(await screen.findByRole('heading', { name: 'Вход' })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('Регистрация завершена')
    expect(body).toEqual({ name: 'Иван', email: 'ivan@example.com', password: 'password1' })
  })

  it('проверяет поля до отправки на сервер', async () => {
    start()

    await fillAndSubmit('', 'not-an-email', 'short')

    expect(screen.getByText('Имя обязательно')).toBeInTheDocument()
    expect(screen.getByText('Укажите корректный email')).toBeInTheDocument()
    expect(screen.getByText(/не короче 8 символов/)).toBeInTheDocument()
  })

  it('сообщает, что email уже занят', async () => {
    server.use(
      http.post(api('/auth/register'), () =>
        HttpResponse.json({ code: 'CONFLICT', message: 'email taken' }, { status: 409 }),
      ),
    )
    start()

    await fillAndSubmit('Иван', 'ivan@example.com', 'password1')

    expect(
      await screen.findByText('Пользователь с таким email уже зарегистрирован'),
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Регистрация' })).toBeInTheDocument()
  })

  it('показывает ошибки полей, пришедшие от сервера', async () => {
    server.use(
      http.post(api('/auth/register'), () =>
        HttpResponse.json(
          {
            code: 'VALIDATION_ERROR',
            message: 'Некорректные данные',
            details: [{ field: 'password', message: 'Слишком простой пароль' }],
          },
          { status: 400 },
        ),
      ),
    )
    start()

    await fillAndSubmit('Иван', 'ivan@example.com', 'password1')

    expect(await screen.findByText('Слишком простой пароль')).toBeInTheDocument()
  })

  it('показывает общую ошибку при сбое сервера', async () => {
    server.use(http.post(api('/auth/register'), () => HttpResponse.error()))
    start()

    await fillAndSubmit('Иван', 'ivan@example.com', 'password1')

    expect(await screen.findByRole('alert')).toHaveTextContent('Не удалось выполнить запрос')
  })
})

import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router'
import { describe, expect, it } from 'vitest'
import { selectCurrentUser } from '../features/auth/authSlice'
import { makeAuth, makeUser } from '../test/fixtures'
import { anonymous, renderWithProviders } from '../test/render'
import { api, server } from '../test/server'
import { LoginPage } from './LoginPage'

const routes = (
  <Routes>
    <Route path="/login" element={<LoginPage />} />
    <Route path="/" element={<p>Главная</p>} />
    <Route path="/boards/1" element={<p>Доска</p>} />
  </Routes>
)

async function fillAndSubmit(email: string, password: string) {
  const user = userEvent.setup()
  await user.type(screen.getByLabelText(/Email/), email)
  await user.type(screen.getByLabelText(/Пароль/), password)
  await user.click(screen.getByRole('button', { name: 'Войти' }))
}

describe('LoginPage', () => {
  it('входит и переходит на главную', async () => {
    let body: unknown
    server.use(
      http.post(api('/auth/login'), async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(makeAuth(makeUser('manager')))
      }),
    )
    const { store } = renderWithProviders(routes, { route: '/login', preloadedState: anonymous })

    await fillAndSubmit('  manager@example.com ', 'password1')

    expect(await screen.findByText('Главная')).toBeInTheDocument()
    expect(body).toEqual({ email: 'manager@example.com', password: 'password1' })
    expect(selectCurrentUser(store.getState())?.role).toBe('manager')
  })

  it('после входа возвращает на страницу, с которой пришли', async () => {
    server.use(http.post(api('/auth/login'), () => HttpResponse.json(makeAuth())))
    renderWithProviders(routes, {
      route: { pathname: '/login', state: { from: '/boards/1' } },
      preloadedState: anonymous,
    })

    await fillAndSubmit('developer@example.com', 'password1')

    expect(await screen.findByText('Доска')).toBeInTheDocument()
  })

  it('показывает ошибку при неверных учётных данных', async () => {
    server.use(
      http.post(api('/auth/login'), () =>
        HttpResponse.json(
          { code: 'UNAUTHORIZED', message: 'invalid credentials' },
          { status: 401 },
        ),
      ),
    )
    renderWithProviders(routes, { route: '/login', preloadedState: anonymous })

    await fillAndSubmit('developer@example.com', 'wrong-password')

    expect(await screen.findByRole('alert')).toHaveTextContent('Неверный email или пароль')
    expect(screen.queryByText('Главная')).not.toBeInTheDocument()
  })

  it('показывает сообщение сервера при других ошибках', async () => {
    server.use(
      http.post(api('/auth/login'), () =>
        HttpResponse.json(
          { code: 'RATE_LIMITED', message: 'Слишком много попыток' },
          { status: 429 },
        ),
      ),
    )
    renderWithProviders(routes, { route: '/login', preloadedState: anonymous })

    await fillAndSubmit('developer@example.com', 'password1')

    expect(await screen.findByRole('alert')).toHaveTextContent('Слишком много попыток')
  })

  it('не даёт отправить пустую форму', () => {
    renderWithProviders(routes, { route: '/login', preloadedState: anonymous })
    expect(screen.getByRole('button', { name: 'Войти' })).toBeDisabled()
  })

  it('сообщает об успешной регистрации', () => {
    renderWithProviders(routes, {
      route: { pathname: '/login', state: { registered: true } },
      preloadedState: anonymous,
    })
    expect(screen.getByRole('alert')).toHaveTextContent('Регистрация завершена')
  })
})

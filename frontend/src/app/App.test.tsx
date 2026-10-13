import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { selectAuthStatus } from '../features/auth/authSlice'
import { makeAuth, makeUser } from '../test/fixtures'
import { anonymous, renderWithProviders, signedIn } from '../test/render'
import { api, server } from '../test/server'
import App from './App'

describe('App', () => {
  it('без refresh-cookie сразу отправляет на вход, не обращаясь к серверу', async () => {
    renderWithProviders(<App />)

    expect(await screen.findByRole('heading', { name: 'Вход' })).toBeInTheDocument()
  })

  it('восстанавливает сессию по refresh-cookie', async () => {
    document.cookie = 'csrf_token=csrf-1'
    server.use(
      http.post(api('/auth/refresh'), () => HttpResponse.json(makeAuth(makeUser('admin')))),
      http.get(api('/projects'), () => HttpResponse.json([])),
    )
    renderWithProviders(<App />)

    expect(screen.getByLabelText('Загрузка')).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: 'Проекты' })).toBeInTheDocument()
    expect(screen.getByText('Администратор')).toBeInTheDocument()
  })

  it('отправляет на вход, если refresh-токен недействителен', async () => {
    document.cookie = 'csrf_token=csrf-1'
    server.use(
      http.post(api('/auth/refresh'), () =>
        HttpResponse.json({ code: 'UNAUTHORIZED', message: 'expired' }, { status: 401 }),
      ),
    )
    renderWithProviders(<App />)

    expect(await screen.findByRole('heading', { name: 'Вход' })).toBeInTheDocument()
  })

  it('не пускает анонимного пользователя на защищённые страницы', () => {
    renderWithProviders(<App />, { route: '/', preloadedState: anonymous })

    expect(screen.getByRole('heading', { name: 'Вход' })).toBeInTheDocument()
  })

  it('перенаправляет неизвестный адрес на главную', async () => {
    server.use(http.get(api('/projects'), () => HttpResponse.json([])))
    renderWithProviders(<App />, { route: '/no-such-page', preloadedState: signedIn(makeUser()) })

    expect(await screen.findByRole('heading', { name: 'Проекты' })).toBeInTheDocument()
  })

  it('выходит из системы', async () => {
    document.cookie = 'csrf_token=csrf-1'
    let csrf: string | null = null
    server.use(
      http.get(api('/projects'), () => HttpResponse.json([])),
      http.post(api('/auth/logout'), ({ request }) => {
        csrf = request.headers.get('X-CSRF-Token')
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const { store } = renderWithProviders(<App />, { preloadedState: signedIn(makeUser()) })

    await userEvent.setup().click(await screen.findByRole('button', { name: 'Выйти' }))

    expect(await screen.findByRole('heading', { name: 'Вход' })).toBeInTheDocument()
    expect(csrf).toBe('csrf-1')
    expect(selectAuthStatus(store.getState())).toBe('anonymous')
  })

  it('завершает локальную сессию, даже если сервер не ответил на выход', async () => {
    server.use(
      http.get(api('/projects'), () => HttpResponse.json([])),
      http.post(api('/auth/logout'), () => HttpResponse.error()),
    )
    renderWithProviders(<App />, { preloadedState: signedIn(makeUser()) })

    await userEvent.setup().click(await screen.findByRole('button', { name: 'Выйти' }))

    expect(await screen.findByRole('heading', { name: 'Вход' })).toBeInTheDocument()
  })
})

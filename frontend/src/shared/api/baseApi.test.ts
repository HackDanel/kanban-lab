import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { makeStore } from '../../app/store'
import { authApi } from '../../features/auth/authApi'
import { selectAccessToken, selectAuthStatus } from '../../features/auth/authSlice'
import { projectsApi } from '../../features/projects/projectsApi'
import { makeAuth, makeProject, makeUser } from '../../test/fixtures'
import { signedIn } from '../../test/render'
import { api, server } from '../../test/server'

const unauthorized = () =>
  HttpResponse.json({ code: 'UNAUTHORIZED', message: 'Токен недействителен' }, { status: 401 })

describe('baseApi: обновление токена', () => {
  it('передаёт access-токен в заголовке Authorization', async () => {
    let header: string | null = null
    server.use(
      http.get(api('/projects'), ({ request }) => {
        header = request.headers.get('Authorization')
        return HttpResponse.json([])
      }),
    )
    const store = makeStore(signedIn(makeUser(), 'access-1'))

    await store.dispatch(projectsApi.endpoints.listProjects.initiate())

    expect(header).toBe('Bearer access-1')
  })

  it('при 401 обновляет токен и повторяет запрос с новым токеном', async () => {
    document.cookie = 'csrf_token=csrf-1'
    const seen: (string | null)[] = []
    let csrf: string | null = null
    server.use(
      http.get(api('/projects'), ({ request }) => {
        const header = request.headers.get('Authorization')
        seen.push(header)
        return header === 'Bearer access-2' ? HttpResponse.json([makeProject()]) : unauthorized()
      }),
      http.post(api('/auth/refresh'), ({ request }) => {
        csrf = request.headers.get('X-CSRF-Token')
        return HttpResponse.json(makeAuth(makeUser(), 'access-2'))
      }),
    )
    const store = makeStore(signedIn(makeUser(), 'access-1'))

    const result = await store.dispatch(projectsApi.endpoints.listProjects.initiate())

    expect(result.data).toEqual([makeProject()])
    expect(seen).toEqual(['Bearer access-1', 'Bearer access-2'])
    expect(csrf).toBe('csrf-1')
    expect(selectAccessToken(store.getState())).toBe('access-2')
  })

  it('завершает сессию, если обновить токен не удалось', async () => {
    server.use(
      http.get(api('/projects'), unauthorized),
      http.post(api('/auth/refresh'), unauthorized),
    )
    const store = makeStore(signedIn(makeUser()))

    const result = await store.dispatch(projectsApi.endpoints.listProjects.initiate())

    expect(result.error).toMatchObject({ status: 401 })
    expect(selectAuthStatus(store.getState())).toBe('anonymous')
    expect(selectAccessToken(store.getState())).toBeNull()
  })

  it('выполняет одно обновление на несколько параллельных 401', async () => {
    let refreshes = 0
    server.use(
      http.delete(api('/projects/:id'), ({ request }) =>
        request.headers.get('Authorization') === 'Bearer access-2'
          ? new HttpResponse(null, { status: 204 })
          : unauthorized(),
      ),
      http.post(api('/auth/refresh'), () => {
        refreshes += 1
        return HttpResponse.json(makeAuth(makeUser(), 'access-2'))
      }),
    )
    const store = makeStore(signedIn(makeUser(), 'access-1'))

    const results = await Promise.all(
      ['a', 'b', 'c'].map((id) => store.dispatch(projectsApi.endpoints.deleteProject.initiate(id))),
    )

    expect(results.every((r) => !('error' in r) || r.error === undefined)).toBe(true)
    expect(refreshes).toBe(1)
  })

  it('не пытается обновить токен при 401 на эндпоинтах /auth', async () => {
    let refreshes = 0
    server.use(
      http.post(api('/auth/login'), unauthorized),
      http.post(api('/auth/refresh'), () => {
        refreshes += 1
        return unauthorized()
      }),
    )
    const store = makeStore()

    const result = await store.dispatch(
      authApi.endpoints.login.initiate({ email: 'a@example.com', password: 'wrong' }),
    )

    expect(result.error).toMatchObject({ status: 401 })
    expect(refreshes).toBe(0)
    expect(selectAuthStatus(store.getState())).toBe('unknown')
  })
})

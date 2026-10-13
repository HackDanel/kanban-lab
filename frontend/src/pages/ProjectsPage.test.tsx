import { screen, waitFor, waitForElementToBeRemoved, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import type { Project } from '../shared/api/types'
import { makeProject, makeUser } from '../test/fixtures'
import { renderWithProviders, signedIn } from '../test/render'
import { api, server } from '../test/server'
import { ProjectsPage } from './ProjectsPage'

const manager = makeUser('manager')
const own = makeProject({ ownerId: manager.id })
const foreign = makeProject({
  id: '10000000-0000-4000-8000-000000000002',
  key: 'OPS',
  name: 'Operations',
  ownerId: 'someone-else',
})

function serveProjects(initial: Project[]) {
  const projects = [...initial]
  server.use(
    http.get(api('/projects'), () => HttpResponse.json(projects)),
    http.post(api('/projects'), async ({ request }) => {
      const input = (await request.json()) as { key: string; name: string; description: string }
      if (projects.some((p) => p.key === input.key)) {
        return HttpResponse.json({ code: 'CONFLICT', message: 'key taken' }, { status: 409 })
      }
      const created = makeProject({ ...input, id: `id-${input.key}`, ownerId: manager.id })
      projects.push(created)
      return HttpResponse.json(created, { status: 201 })
    }),
    http.delete(api('/projects/:id'), ({ params }) => {
      const index = projects.findIndex((p) => p.id === params.id)
      if (index < 0) {
        return HttpResponse.json({ code: 'NOT_FOUND', message: 'нет проекта' }, { status: 404 })
      }
      projects.splice(index, 1)
      return new HttpResponse(null, { status: 204 })
    }),
  )
  return projects
}

describe('ProjectsPage', () => {
  it('показывает список проектов', async () => {
    serveProjects([own, foreign])
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(manager) })

    expect(await screen.findByRole('heading', { name: 'Kanban' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Operations' })).toBeInTheDocument()
  })

  it('TC-04: разработчику недоступны создание и удаление проекта', async () => {
    serveProjects([own])
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(makeUser('developer')) })

    await screen.findByRole('heading', { name: 'Kanban' })

    expect(screen.queryByRole('button', { name: 'Создать проект' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Удалить проект/ })).not.toBeInTheDocument()
  })

  it('менеджер может удалить только свой проект', async () => {
    serveProjects([own, foreign])
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(manager) })

    await screen.findByRole('heading', { name: 'Kanban' })

    expect(screen.getByRole('button', { name: 'Удалить проект Kanban' })).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Удалить проект Operations' }),
    ).not.toBeInTheDocument()
  })

  it('администратор может удалить любой проект', async () => {
    serveProjects([own, foreign])
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(makeUser('admin')) })

    await screen.findByRole('heading', { name: 'Kanban' })

    expect(screen.getAllByRole('button', { name: /Удалить проект/ })).toHaveLength(2)
  })

  it('создаёт проект и показывает его в списке', async () => {
    const projects = serveProjects([])
    const user = userEvent.setup()
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(manager) })
    expect(await screen.findByText('Проектов пока нет. Создайте первый.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Создать проект' }))
    const dialog = screen.getByRole('dialog')
    await user.type(within(dialog).getByLabelText(/Ключ/), 'web')
    await user.type(within(dialog).getByLabelText(/Название/), '  Сайт ')
    await user.type(within(dialog).getByLabelText(/Описание/), 'Новый сайт')
    await user.click(within(dialog).getByRole('button', { name: 'Создать' }))

    expect(await screen.findByRole('heading', { name: 'Сайт' })).toBeInTheDocument()
    expect(projects).toHaveLength(1)
    expect(projects[0]).toMatchObject({ key: 'WEB', name: 'Сайт', description: 'Новый сайт' })
  })

  it('проверяет форму проекта до отправки', async () => {
    const projects = serveProjects([])
    const user = userEvent.setup()
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(manager) })

    await user.click(await screen.findByRole('button', { name: 'Создать проект' }))
    await user.click(screen.getByRole('button', { name: 'Создать' }))

    expect(screen.getByText('Название обязательно')).toBeInTheDocument()
    expect(screen.getByText(/латинских заглавных букв/)).toBeInTheDocument()
    expect(projects).toHaveLength(0)
  })

  it('сообщает о занятом ключе проекта', async () => {
    serveProjects([own])
    const user = userEvent.setup()
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(manager) })

    await user.click(await screen.findByRole('button', { name: 'Создать проект' }))
    await user.type(screen.getByLabelText(/Ключ/), 'KAN')
    await user.type(screen.getByLabelText(/Название/), 'Дубль')
    await user.click(screen.getByRole('button', { name: 'Создать' }))

    expect(await screen.findByText('Проект с таким ключом уже существует')).toBeInTheDocument()
  })

  it('удаляет проект после подтверждения', async () => {
    const projects = serveProjects([own])
    const user = userEvent.setup()
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(manager) })

    await user.click(await screen.findByRole('button', { name: 'Удалить проект Kanban' }))
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveTextContent('Проект «Kanban» будет удалён')
    await user.click(within(dialog).getByRole('button', { name: 'Удалить' }))

    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: 'Kanban' })).not.toBeInTheDocument(),
    )
    expect(projects).toHaveLength(0)
  })

  it('не удаляет проект при отмене', async () => {
    const projects = serveProjects([own])
    const user = userEvent.setup()
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(manager) })

    await user.click(await screen.findByRole('button', { name: 'Удалить проект Kanban' }))
    await user.click(screen.getByRole('button', { name: 'Отмена' }))

    await waitForElementToBeRemoved(() => screen.queryByRole('dialog'))
    expect(screen.getByRole('heading', { name: 'Kanban' })).toBeInTheDocument()
    expect(projects).toHaveLength(1)
  })

  it('показывает ошибку загрузки', async () => {
    server.use(
      http.get(api('/projects'), () =>
        HttpResponse.json({ code: 'INTERNAL', message: 'Сервис недоступен' }, { status: 500 }),
      ),
    )
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(manager) })

    expect(await screen.findByRole('alert')).toHaveTextContent('Сервис недоступен')
  })

  it('объясняет разработчику пустой список', async () => {
    serveProjects([])
    renderWithProviders(<ProjectsPage />, { preloadedState: signedIn(makeUser('developer')) })

    expect(await screen.findByText('Вас пока не добавили ни в один проект.')).toBeInTheDocument()
  })
})

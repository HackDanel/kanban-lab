import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it } from 'vitest'
import { fakeApi } from '../test/fakeApi'
import type { FakeState } from '../test/fakeApi'
import { renderWithProviders, signedIn } from '../test/render'
import { AdminUsersPage } from './AdminUsersPage'
import { ProjectPage } from './ProjectPage'
import { makeUser } from '../test/fixtures'

let state: FakeState

function renderProject(role: 'manager' | 'developer') {
  const user = state.members[role === 'manager' ? 0 : 1]
  return renderWithProviders(
    <Routes>
      <Route path="/projects/:projectId" element={<ProjectPage />} />
      <Route path="/boards/:boardId" element={<p>Доска открыта</p>} />
    </Routes>,
    { route: `/projects/${state.project.id}`, preloadedState: signedIn(user) },
  )
}

const section = (name: string) => screen.getByRole('region', { name, hidden: true })

beforeEach(() => {
  state = fakeApi()
})

describe('ProjectPage', () => {
  it('показывает доски, участников и метки', async () => {
    renderProject('developer')
    expect(await screen.findByRole('heading', { name: 'Kanban' })).toBeInTheDocument()
    expect(await within(section('Доски')).findByText('Sprint')).toBeInTheDocument()
    expect(await within(section('Участники')).findByText(/\(владелец\)/)).toBeInTheDocument()
    expect(await within(section('Метки')).findByText('bug')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Создать доску' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Добавить участника')).not.toBeInTheDocument()
  })

  it('открывает доску по клику', async () => {
    const user = userEvent.setup()
    renderProject('developer')
    await user.click(await screen.findByText('Sprint'))
    expect(await screen.findByText('Доска открыта')).toBeInTheDocument()
  })

  it('менеджер создаёт и удаляет доску', async () => {
    const user = userEvent.setup()
    renderProject('manager')
    await screen.findByText('Sprint')

    await user.click(screen.getByRole('button', { name: 'Создать доску' }))
    expect(screen.getByText('Название обязательно')).toBeInTheDocument()
    await user.type(screen.getByLabelText('Новая доска'), 'Релиз')
    await user.click(screen.getByRole('button', { name: 'Создать доску' }))
    await waitFor(() =>
      expect(state.requests).toContainEqual(
        expect.objectContaining({ method: 'POST', body: { name: 'Релиз' } }),
      ),
    )

    await user.click(screen.getByRole('button', { name: 'Удалить доску Sprint' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Удалить' }))
    await waitFor(() =>
      expect(state.requests).toContainEqual(
        expect.objectContaining({ method: 'DELETE', path: '/boards/board-1' }),
      ),
    )
  })

  it('владелец добавляет и исключает участников', async () => {
    const user = userEvent.setup()
    renderProject('manager')
    await screen.findByText(/\(владелец\)/)
    expect(
      screen.queryByRole('button', { name: `Исключить ${state.members[0].name}` }),
    ).not.toBeInTheDocument()

    await user.type(screen.getByLabelText('Добавить участника'), 'Outs')
    await user.click(await screen.findByRole('option', { name: /Outsider/ }))
    expect(await within(section('Участники')).findByText('Outsider')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Исключить Outsider' }))
    await waitFor(() =>
      expect(within(section('Участники')).queryByText('Outsider')).not.toBeInTheDocument(),
    )
  })

  it('менеджер создаёт и удаляет метки', async () => {
    const user = userEvent.setup()
    renderProject('manager')
    await screen.findByText('bug')

    await user.click(within(section('Метки')).getByRole('button', { name: 'Добавить' }))
    expect(within(section('Метки')).getByText('Название обязательно')).toBeInTheDocument()

    await user.type(screen.getByLabelText('Новая метка'), 'bug')
    await user.click(within(section('Метки')).getByRole('button', { name: 'Добавить' }))
    expect(await screen.findByText('Метка уже есть')).toBeInTheDocument()

    await user.clear(screen.getByLabelText('Новая метка'))
    await user.type(screen.getByLabelText('Новая метка'), 'ui')
    await user.click(within(section('Метки')).getByRole('button', { name: 'Добавить' }))
    expect(await within(section('Метки')).findByText('ui')).toBeInTheDocument()

    const bug = within(section('Метки')).getByText('bug').closest('.MuiChip-root')!
    await user.click(within(bug as HTMLElement).getByTestId('CancelIcon'))
    await waitFor(() => expect(within(section('Метки')).queryByText('bug')).not.toBeInTheDocument())
  })

  it('показывает ошибку загрузки проекта', async () => {
    const { server, api } = await import('../test/server')
    const { http, HttpResponse } = await import('msw')
    server.use(
      http.get(api('/projects/:id'), () =>
        HttpResponse.json(
          { code: 'NOT_FOUND', message: 'Проект: объект не найден' },
          { status: 404 },
        ),
      ),
    )
    renderProject('manager')
    expect(await screen.findByText('Проект: объект не найден')).toBeInTheDocument()
  })
})

describe('AdminUsersPage', () => {
  it('администратор меняет роль пользователя', async () => {
    const admin = makeUser('admin', { id: 'u-admin', name: 'Root' })
    state.users.push(admin)
    const user = userEvent.setup()
    renderWithProviders(
      <Routes>
        <Route path="/admin/users" element={<AdminUsersPage />} />
      </Routes>,
      { route: '/admin/users', preloadedState: signedIn(admin) },
    )
    const dev = state.members[1]
    await user.click(await screen.findByRole('combobox', { name: `Роль ${dev.name}` }))
    await user.click(screen.getByRole('option', { name: 'Менеджер' }))
    await waitFor(() =>
      expect(state.requests).toContainEqual(
        expect.objectContaining({ method: 'PUT', body: { role: 'manager' } }),
      ),
    )
    expect(screen.getByRole('combobox', { name: 'Роль Root' })).toHaveAttribute(
      'aria-disabled',
      'true',
    )
  })

  it('не пускает не-администратора', () => {
    renderWithProviders(
      <Routes>
        <Route path="/admin/users" element={<AdminUsersPage />} />
        <Route path="/" element={<p>Главная</p>} />
      </Routes>,
      { route: '/admin/users', preloadedState: signedIn(state.members[0]) },
    )
    expect(screen.getByText('Главная')).toBeInTheDocument()
  })
})

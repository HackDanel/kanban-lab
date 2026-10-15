import { act, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fakeApi } from '../test/fakeApi'
import type { FakeState } from '../test/fakeApi'
import { FakeSocket } from '../test/fakeSocket'
import { makeUser } from '../test/fixtures'
import { renderWithProviders, signedIn } from '../test/render'
import { BoardPage } from './BoardPage'

const routes = (
  <Routes>
    <Route path="/boards/:boardId" element={<BoardPage />} />
    <Route path="/projects/:projectId" element={<p>Страница проекта</p>} />
  </Routes>
)

let state: FakeState

function renderBoard(role: 'manager' | 'developer' | 'admin' = 'manager') {
  const user =
    role === 'manager'
      ? state.members[0]
      : role === 'developer'
        ? state.members[1]
        : makeUser('admin')
  return renderWithProviders(routes, { route: '/boards/board-1', preloadedState: signedIn(user) })
}

// hidden: пока открыт диалог, остальная страница скрыта от вспомогательных технологий
const column = (name: string) =>
  screen.getByRole('region', { name: `Колонка ${name}`, hidden: true })

beforeEach(() => {
  FakeSocket.install()
  state = fakeApi()
})

describe('BoardPage', () => {
  it('показывает колонки с карточками, счётчики и WIP-лимит', async () => {
    renderBoard()

    expect(await screen.findByText('Login form')).toBeInTheDocument()
    expect(within(column('To Do')).getAllByTestId('card')).toHaveLength(2)
    expect(within(column('In Progress')).getByText('Setup CI')).toBeInTheDocument()
    expect(within(column('In Progress')).getByTestId('column-count')).toHaveTextContent('1 / 1')
    expect(await within(column('To Do')).findByText('bug')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Kanban' })).toHaveAttribute(
      'href',
      '/projects/' + state.project.id,
    )
  })

  it('TC-02: создаёт карточку в колонке', async () => {
    const user = userEvent.setup()
    renderBoard('developer')

    await user.click(
      await screen.findByRole('button', { name: 'Добавить карточку в колонку Done' }),
    )
    await user.type(
      screen.getByRole('textbox', { name: 'Название карточки' }),
      'Новая задача{Enter}',
    )

    expect(await within(column('Done')).findByText('Новая задача')).toBeInTheDocument()
    expect(state.requests).toContainEqual(
      expect.objectContaining({
        method: 'POST',
        path: '/columns/done/cards',
        body: { title: 'Новая задача' },
      }),
    )
  })

  it('TC-05: не создаёт карточку с пустым названием', async () => {
    const user = userEvent.setup()
    renderBoard()

    await user.click(
      await screen.findByRole('button', { name: 'Добавить карточку в колонку Done' }),
    )
    await user.click(screen.getByRole('button', { name: 'Добавить' }))

    expect(screen.getByText('Название обязательно')).toBeInTheDocument()
    expect(state.requests.some((r) => r.method === 'POST' && r.path.endsWith('/cards'))).toBe(false)
  })

  it('отменяет добавление по Escape и кнопке', async () => {
    const user = userEvent.setup()
    renderBoard()

    await user.click(
      await screen.findByRole('button', { name: 'Добавить карточку в колонку Done' }),
    )
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('textbox', { name: 'Название карточки' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Добавить карточку в колонку Done' }))
    await user.click(screen.getByRole('button', { name: 'Отмена' }))
    expect(screen.queryByRole('textbox', { name: 'Название карточки' })).not.toBeInTheDocument()
  })

  it('не даёт добавить карточку в заполненную колонку', async () => {
    renderBoard()
    expect(
      await screen.findByRole('button', { name: 'Добавить карточку в колонку In Progress' }),
    ).toBeDisabled()
  })

  it('TC-06: ищет карточки по тексту и сбрасывает фильтры', async () => {
    const user = userEvent.setup()
    renderBoard()
    await screen.findByText('Login form')

    await user.type(screen.getByRole('searchbox', { name: 'Поиск' }), 'login')

    await waitFor(() => expect(screen.queryByText('Reports')).not.toBeInTheDocument())
    expect(screen.getByText('Login form')).toBeInTheDocument()
    expect(screen.getByText('Найдено карточек: 1')).toBeInTheDocument()
    expect(state.requests.some((r) => r.search?.includes('q=login'))).toBe(true)

    await user.click(screen.getByRole('button', { name: 'Сбросить' }))
    expect(await screen.findByText('Reports')).toBeInTheDocument()
  })

  it('фильтрует по исполнителю и приоритету', async () => {
    const user = userEvent.setup()
    renderBoard()
    await screen.findByText('Login form')

    await user.click(screen.getByRole('combobox', { name: 'Исполнитель' }))
    await user.click(screen.getByRole('option', { name: state.members[1].name }))
    await waitFor(() => expect(screen.queryByText('Login form')).not.toBeInTheDocument())
    expect(screen.getByText('Reports')).toBeInTheDocument()

    await user.click(screen.getByRole('combobox', { name: 'Приоритет' }))
    await user.click(screen.getByRole('option', { name: 'Высокий' }))
    await waitFor(() => expect(screen.queryByText('Reports')).not.toBeInTheDocument())
    expect(state.requests.some((r) => r.search?.includes('priority=high'))).toBe(true)
  })

  it('TC-07: перемещение в заполненную колонку откатывается с предупреждением', async () => {
    const user = userEvent.setup()
    renderBoard('developer')

    await user.click(await screen.findByRole('button', { name: /Карточка KAN-1: Login form/ }))
    const dialog = screen.getByRole('dialog')
    await user.click(within(dialog).getByRole('combobox', { name: 'Колонка' }))
    await user.click(screen.getByRole('option', { name: 'In Progress' }))

    expect(await screen.findByText(/Превышен WIP-лимит колонки «In Progress»/)).toBeInTheDocument()
    await waitFor(() => expect(within(column('To Do')).getByText('Login form')).toBeInTheDocument())
  })

  it('TC-03: перемещает карточку в другую колонку из диалога', async () => {
    const user = userEvent.setup()
    renderBoard('developer')

    await user.click(await screen.findByRole('button', { name: /Карточка KAN-2: Reports/ }))
    await user.click(within(screen.getByRole('dialog')).getByRole('combobox', { name: 'Колонка' }))
    await user.click(screen.getByRole('option', { name: 'Done' }))

    await waitFor(() => expect(within(column('Done')).getByText('Reports')).toBeInTheDocument())
    expect(state.requests).toContainEqual(
      expect.objectContaining({
        path: '/cards/c2/move',
        body: { columnId: 'done', afterCardId: null },
      }),
    )
  })

  it('разработчику недоступны действия с колонками', async () => {
    renderBoard('developer')
    await screen.findByText('Login form')
    expect(screen.queryByRole('button', { name: /Действия с колонкой/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Колонка' })).not.toBeInTheDocument()
  })

  it('менеджер создаёт колонку с WIP-лимитом', async () => {
    const user = userEvent.setup()
    renderBoard()

    await user.click(await screen.findByRole('button', { name: 'Колонка' }))
    const dialog = screen.getByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(within(dialog).getByText('Название обязательно')).toBeInTheDocument()

    await user.type(within(dialog).getByLabelText(/Название/), 'Review')
    await user.type(within(dialog).getByLabelText('WIP-лимит'), '0')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(within(dialog).getByText(/Целое число от 1 до 1000/)).toBeInTheDocument()

    await user.clear(within(dialog).getByLabelText('WIP-лимит'))
    await user.type(within(dialog).getByLabelText('WIP-лимит'), '3')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))

    expect(await screen.findByRole('region', { name: 'Колонка Review' })).toBeInTheDocument()
    expect(state.requests).toContainEqual(
      expect.objectContaining({
        path: '/boards/board-1/columns',
        body: { name: 'Review', wipLimit: 3 },
      }),
    )
  })

  it('менеджер настраивает, сдвигает и удаляет колонки', async () => {
    const user = userEvent.setup()
    renderBoard()

    await user.click(await screen.findByRole('button', { name: 'Действия с колонкой Done' }))
    await user.click(screen.getByRole('menuitem', { name: 'Настроить' }))
    const dialog = screen.getByRole('dialog')
    await user.clear(within(dialog).getByLabelText('WIP-лимит'))
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(state.requests).toContainEqual(
      expect.objectContaining({
        method: 'PATCH',
        path: '/columns/done',
        body: { name: 'Done', wipLimit: null },
      }),
    )

    await user.click(screen.getByRole('button', { name: 'Действия с колонкой Done' }))
    expect(screen.getByRole('menuitem', { name: 'Сдвинуть вправо' })).toHaveAttribute(
      'aria-disabled',
      'true',
    )
    await user.click(screen.getByRole('menuitem', { name: 'Сдвинуть влево' }))
    expect(state.requests).toContainEqual(
      expect.objectContaining({ path: '/columns/done/move', body: { afterColumnId: 'todo' } }),
    )

    await user.click(screen.getByRole('button', { name: 'Действия с колонкой To Do' }))
    await user.click(screen.getByRole('menuitem', { name: 'Удалить' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Удалить' }))
    expect(await screen.findByText('В колонке есть карточки')).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())

    await user.click(screen.getByRole('button', { name: 'Действия с колонкой Done' }))
    await user.click(screen.getByRole('menuitem', { name: 'Удалить' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Удалить' }))
    await waitFor(() =>
      expect(screen.queryByRole('region', { name: 'Колонка Done' })).not.toBeInTheDocument(),
    )
  })

  it('TC-27: перезагружает карточки по событию WebSocket и уходит с удалённой доски', async () => {
    renderBoard()
    await screen.findByText('Login form')
    const socket = FakeSocket.last()
    expect(socket.url).toContain('boardId=board-1')
    expect(socket.url).toContain('token=access-1')

    state.cards.push({ ...state.cards[0], id: 'c9', number: 9, title: 'Из другой вкладки' })
    act(() => socket.emit({ type: 'card.created', boardId: 'board-1', payload: { cardId: 'c9' } }))
    expect(await screen.findByText('Из другой вкладки')).toBeInTheDocument()

    act(() => socket.emit('not json'))
    act(() => socket.emit({ type: 'board.deleted', boardId: 'board-1' }))
    expect(await screen.findByText('Страница проекта')).toBeInTheDocument()
  })

  it('переподключается после обрыва соединения', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    try {
      renderBoard()
      await screen.findByText('Login form')
      const first = FakeSocket.last()
      act(() => first.onopen?.())
      act(() => first.onclose?.())
      await act(() => vi.advanceTimersByTimeAsync(1100))
      expect(FakeSocket.instances.length).toBeGreaterThan(1)
    } finally {
      vi.useRealTimers()
    }
  })

  it('показывает ошибку, если доска недоступна', async () => {
    const { server, api } = await import('../test/server')
    const { http, HttpResponse } = await import('msw')
    server.use(
      http.get(api('/boards/:id'), () =>
        HttpResponse.json({ code: 'FORBIDDEN', message: 'Недостаточно прав' }, { status: 403 }),
      ),
    )
    renderBoard()
    expect(await screen.findByText('Недостаточно прав')).toBeInTheDocument()
  })
})

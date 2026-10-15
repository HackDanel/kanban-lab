import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { BoardPage } from '../../pages/BoardPage'
import { fakeApi } from '../../test/fakeApi'
import type { FakeState } from '../../test/fakeApi'
import { FakeSocket } from '../../test/fakeSocket'
import { renderWithProviders, signedIn } from '../../test/render'

let state: FakeState

async function openCard(title: string, role: 'manager' | 'developer' = 'manager') {
  const user = userEvent.setup()
  renderWithProviders(
    <Routes>
      <Route path="/boards/:boardId" element={<BoardPage />} />
    </Routes>,
    {
      route: '/boards/board-1',
      preloadedState: signedIn(state.members[role === 'manager' ? 0 : 1]),
    },
  )
  await user.click(
    await screen.findByRole('button', { name: new RegExp(`Карточка KAN-\\d+: ${title}`) }),
  )
  return { user, dialog: screen.getByRole('dialog') }
}

beforeEach(() => {
  FakeSocket.install()
  state = fakeApi()
})

describe('CardDialog', () => {
  it('TC-20: сохраняет только изменённые поля', async () => {
    const { user, dialog } = await openCard('Login form')
    expect(within(dialog).getByText('KAN-1')).toBeInTheDocument()

    const title = within(dialog).getByLabelText(/Название/)
    await user.clear(title)
    await user.type(title, 'Login page')
    await user.type(within(dialog).getByLabelText('Описание'), 'Форма входа')
    await user.click(within(dialog).getByRole('combobox', { name: 'Исполнитель' }))
    await user.click(screen.getByRole('option', { name: state.members[1].name }))
    await user.click(within(dialog).getByRole('combobox', { name: 'Приоритет' }))
    await user.click(screen.getByRole('option', { name: 'Критический' }))
    await user.type(within(dialog).getByLabelText('Дедлайн'), '2026-12-31')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))

    expect(await within(dialog).findByText('Изменения сохранены')).toBeInTheDocument()
    const patch = state.requests.find((r) => r.method === 'PATCH' && r.path === '/cards/c1')
    expect(patch?.body).toEqual({
      title: 'Login page',
      description: 'Форма входа',
      priority: 'critical',
      assigneeId: state.members[1].id,
      dueDate: '2026-12-31',
    })
  })

  it('TC-05: не сохраняет пустое название', async () => {
    const { user, dialog } = await openCard('Login form')
    await user.clear(within(dialog).getByLabelText(/Название/))
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(within(dialog).getByText('Название обязательно')).toBeInTheDocument()
    expect(state.requests.some((r) => r.method === 'PATCH')).toBe(false)
  })

  it('без изменений не отправляет запрос', async () => {
    const { user, dialog } = await openCard('Login form')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(within(dialog).getByText('Изменения сохранены')).toBeInTheDocument()
    expect(state.requests.some((r) => r.method === 'PATCH')).toBe(false)
  })

  it('убирает метку', async () => {
    const { user, dialog } = await openCard('Login form')
    await user.click(within(dialog).getByRole('combobox', { name: 'Метки' }))
    await user.click(screen.getByRole('option', { name: 'bug' }))
    await user.keyboard('{Escape}')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    await waitFor(() =>
      expect(state.requests.find((r) => r.method === 'PATCH')?.body).toEqual({ labelIds: [] }),
    )
  })

  it('TC-21: добавляет, изменяет и удаляет комментарий', async () => {
    const { user, dialog } = await openCard('Login form')
    expect(within(dialog).getByText('Комментариев пока нет.')).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: 'Отправить' }))
    expect(within(dialog).getByText('Комментарий не может быть пустым')).toBeInTheDocument()

    await user.type(within(dialog).getByLabelText('Новый комментарий'), 'Первый')
    await user.click(within(dialog).getByRole('button', { name: 'Отправить' }))
    expect(await within(dialog).findByText('Первый')).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: 'Изменить' }))
    const editor = within(dialog).getByRole('textbox', { name: 'Текст комментария' })
    await user.clear(editor)
    await user.click(within(dialog).getAllByRole('button', { name: 'Сохранить' })[0])
    expect(within(dialog).getByText('Комментарий не может быть пустым')).toBeInTheDocument()
    await user.type(editor, 'Исправленный')
    await user.click(within(dialog).getAllByRole('button', { name: 'Сохранить' })[0])
    expect(await within(dialog).findByText('Исправленный')).toBeInTheDocument()
    expect(within(dialog).getByText(/\(изменён\)/)).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: 'Удалить' }))
    expect(await within(dialog).findByText('Комментариев пока нет.')).toBeInTheDocument()
  })

  it('чужой комментарий нельзя изменить, а разработчик не может его удалить', async () => {
    state.comments.push({
      id: 'm1',
      cardId: 'c1',
      authorId: state.members[0].id,
      body: 'Комментарий менеджера',
      createdAt: '2026-01-02T10:00:00Z',
      updatedAt: '2026-01-02T10:00:00Z',
    })
    const { dialog } = await openCard('Login form', 'developer')
    expect(await within(dialog).findByText('Комментарий менеджера')).toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: 'Изменить' })).not.toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: 'Удалить' })).not.toBeInTheDocument()
  })

  it('TC-22: загружает, скачивает и удаляет вложение', async () => {
    const createObjectURL = vi.fn(() => 'blob:x')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL, revokeObjectURL }))
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const { user, dialog } = await openCard('Login form')
    await user.click(within(dialog).getByRole('tab', { name: /Вложения/ }))
    expect(within(dialog).getByText('Вложений пока нет.')).toBeInTheDocument()

    const big = new File([new Uint8Array(11 * 1024 * 1024)], 'big.bin')
    await user.upload(within(dialog).getByLabelText('Файл вложения'), big)
    expect(within(dialog).getByText('Файл больше 10 МБ')).toBeInTheDocument()

    await user.upload(
      within(dialog).getByLabelText('Файл вложения'),
      new File(['%PDF'], 'report.pdf', { type: 'application/pdf' }),
    )
    expect(await within(dialog).findByText('report.pdf')).toBeInTheDocument()
    expect(within(dialog).getByText('2.0 КБ')).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: 'Скачать report.pdf' }))
    await waitFor(() => expect(click).toHaveBeenCalled())
    expect(createObjectURL).toHaveBeenCalled()

    await user.click(within(dialog).getByRole('button', { name: 'Удалить report.pdf' }))
    expect(await within(dialog).findByText('Вложений пока нет.')).toBeInTheDocument()
    click.mockRestore()
  })

  it('показывает историю изменений', async () => {
    state.activity = [
      {
        id: 'h2',
        cardId: 'c1',
        actorId: state.members[1].id,
        action: 'moved',
        oldValue: 'To Do',
        newValue: 'Done',
        createdAt: '2026-01-02T10:00:00Z',
      },
      {
        id: 'h1',
        cardId: 'c1',
        actorId: state.members[0].id,
        action: 'created',
        createdAt: '2026-01-01T10:00:00Z',
      },
    ]
    const { user, dialog } = await openCard('Login form')
    await user.click(within(dialog).getByRole('tab', { name: 'История' }))
    const entries = await within(dialog).findAllByTestId('activity')
    expect(entries[0]).toHaveTextContent(
      `${state.members[1].name} переместил(а) из «To Do» в «Done»`,
    )
    expect(entries[1]).toHaveTextContent('создал(а) карточку')
  })

  it('удаляет карточку после подтверждения', async () => {
    const { user, dialog } = await openCard('Reports')
    await user.click(within(dialog).getByRole('button', { name: 'Удалить карточку' }))
    const confirm = screen.getAllByRole('dialog').at(-1)!
    await user.click(within(confirm).getByRole('button', { name: 'Удалить' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.queryByText('Reports')).not.toBeInTheDocument()
  })
})

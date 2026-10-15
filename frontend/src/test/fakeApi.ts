import { http, HttpResponse } from 'msw'
import type {
  Activity,
  Attachment,
  BoardDetail,
  Card,
  Column,
  Comment,
  Label,
  Project,
  User,
} from '../shared/api/types'
import { makeCard, makeColumn, makeProject, makeUser } from './fixtures'
import { api, server } from './server'

/** Состояние фейкового бэкенда для тестов страниц проекта и доски. */
export interface FakeState {
  project: Project
  board: BoardDetail
  members: User[]
  users: User[]
  labels: Label[]
  cards: Card[]
  comments: Comment[]
  attachments: Attachment[]
  activity: Activity[]
  requests: { method: string; path: string; body?: unknown; search?: string }[]
}

const error = (status: number, code: string, message: string) =>
  HttpResponse.json({ code, message }, { status })

export function fakeApi(overrides: Partial<FakeState> = {}): FakeState {
  const manager = makeUser('manager')
  const developer = makeUser('developer')
  const project = makeProject({ ownerId: manager.id })
  const columns: Column[] = [
    makeColumn({ id: 'todo', name: 'To Do', position: 1024 }),
    makeColumn({ id: 'doing', name: 'In Progress', position: 2048, wipLimit: 1 }),
    makeColumn({ id: 'done', name: 'Done', position: 3072 }),
  ]
  const state: FakeState = {
    project,
    board: {
      id: 'board-1',
      projectId: project.id,
      name: 'Sprint',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
      columns,
    },
    members: [manager, developer],
    users: [
      manager,
      developer,
      makeUser('admin', { id: 'u-other', name: 'Outsider', email: 'out@example.com' }),
    ],
    labels: [{ id: 'label-bug', projectId: project.id, name: 'bug', color: '#ff0000' }],
    cards: [
      makeCard({
        id: 'c1',
        number: 1,
        title: 'Login form',
        columnId: 'todo',
        position: 1024,
        labelIds: ['label-bug'],
      }),
      makeCard({
        id: 'c2',
        number: 2,
        title: 'Reports',
        columnId: 'todo',
        position: 2048,
        assigneeId: developer.id,
      }),
      makeCard({ id: 'c3', number: 3, title: 'Setup CI', columnId: 'doing', position: 1024 }),
    ],
    comments: [],
    attachments: [],
    activity: [],
    requests: [],
    ...overrides,
  }

  const counts = () => {
    for (const c of state.board.columns)
      c.cardCount = state.cards.filter((x) => x.columnId === c.id).length
  }
  counts()

  const log = async (request: Request, path: string) => {
    let body: unknown
    const multipart = request.headers.get('Content-Type')?.startsWith('multipart/')
    if (request.method !== 'GET' && request.method !== 'DELETE' && !multipart) {
      const text = await request.clone().text()
      body = text && request.headers.get('Content-Type')?.includes('json') ? JSON.parse(text) : text
    }
    state.requests.push({ method: request.method, path, body, search: new URL(request.url).search })
  }

  server.use(
    http.all(api('/*'), async ({ request }) => {
      await log(request, new URL(request.url).pathname.replace('/api/v1', ''))
      return undefined
    }),
    http.get(api('/projects/:id'), () => HttpResponse.json(state.project)),
    http.get(api('/projects/:id/members'), () => HttpResponse.json(state.members)),
    http.post(api('/projects/:id/members'), async ({ request }) => {
      const { userId } = (await request.json()) as { userId: string }
      const user = state.users.find((u) => u.id === userId)
      if (!user) return error(404, 'NOT_FOUND', 'Пользователь: объект не найден')
      if (!state.members.some((m) => m.id === userId)) state.members.push(user)
      return new HttpResponse(null, { status: 204 })
    }),
    http.delete(api('/projects/:id/members/:userId'), ({ params }) => {
      state.members = state.members.filter((m) => m.id !== params.userId)
      return new HttpResponse(null, { status: 204 })
    }),
    http.get(api('/projects/:id/labels'), () => HttpResponse.json(state.labels)),
    http.post(api('/projects/:id/labels'), async ({ request }) => {
      const input = (await request.json()) as { name: string; color: string }
      if (state.labels.some((l) => l.name === input.name)) {
        return HttpResponse.json(
          {
            code: 'VALIDATION_ERROR',
            message: 'x',
            details: [{ field: 'name', message: 'Метка уже есть' }],
          },
          { status: 400 },
        )
      }
      const label = { id: `label-${input.name}`, projectId: state.project.id, ...input }
      state.labels.push(label)
      return HttpResponse.json(label, { status: 201 })
    }),
    http.delete(api('/labels/:id'), ({ params }) => {
      state.labels = state.labels.filter((l) => l.id !== params.id)
      return new HttpResponse(null, { status: 204 })
    }),
    http.get(api('/projects/:id/boards'), () => HttpResponse.json([state.board])),
    http.post(api('/projects/:id/boards'), async ({ request }) => {
      const { name } = (await request.json()) as { name: string }
      return HttpResponse.json({ ...state.board, id: 'board-new', name }, { status: 201 })
    }),
    http.delete(api('/boards/:id'), () => new HttpResponse(null, { status: 204 })),
    http.get(api('/users'), ({ request }) => {
      const q = new URL(request.url).searchParams.get('q')?.toLowerCase() ?? ''
      const items = state.users.filter(
        (u) => u.name.toLowerCase().includes(q) || u.email.includes(q),
      )
      return HttpResponse.json({ items, total: items.length })
    }),
    http.put(api('/users/:id/role'), async ({ params, request }) => {
      const { role } = (await request.json()) as { role: User['role'] }
      const user = state.users.find((u) => u.id === params.id)!
      user.role = role
      return HttpResponse.json(user)
    }),

    http.get(api('/boards/:id'), () => {
      counts()
      return HttpResponse.json(state.board)
    }),
    http.post(api('/boards/:id/columns'), async ({ request }) => {
      const input = (await request.json()) as { name: string; wipLimit: number | null }
      const column = makeColumn({
        id: `col-${input.name}`,
        name: input.name,
        wipLimit: input.wipLimit,
        position: 9999,
      })
      state.board.columns.push(column)
      return HttpResponse.json(column, { status: 201 })
    }),
    http.patch(api('/columns/:id'), async ({ params, request }) => {
      const patch = (await request.json()) as Partial<Column>
      const column = state.board.columns.find((c) => c.id === params.id)!
      Object.assign(column, patch)
      return HttpResponse.json(column)
    }),
    http.post(api('/columns/:id/move'), ({ params }) => {
      const column = state.board.columns.find((c) => c.id === params.id)!
      return HttpResponse.json(column)
    }),
    http.delete(api('/columns/:id'), ({ params }) => {
      if (state.cards.some((c) => c.columnId === params.id)) {
        return error(409, 'COLUMN_NOT_EMPTY', 'В колонке есть карточки')
      }
      state.board.columns = state.board.columns.filter((c) => c.id !== params.id)
      return new HttpResponse(null, { status: 204 })
    }),
    http.get(api('/boards/:id/cards'), ({ request }) => {
      const params = new URL(request.url).searchParams
      const q = params.get('q')?.toLowerCase()
      const items = state.cards.filter(
        (c) =>
          (!q || c.title.toLowerCase().includes(q) || c.description.toLowerCase().includes(q)) &&
          (!params.get('assigneeId') || c.assigneeId === params.get('assigneeId')) &&
          (!params.get('priority') || c.priority === params.get('priority')) &&
          params.getAll('labelId').every((id) => c.labelIds.includes(id)),
      )
      return HttpResponse.json({ items, total: items.length })
    }),
    http.post(api('/columns/:id/cards'), async ({ params, request }) => {
      const input = (await request.json()) as { title: string }
      const column = state.board.columns.find((c) => c.id === params.id)!
      if (
        column.wipLimit != null &&
        state.cards.filter((c) => c.columnId === column.id).length >= column.wipLimit
      ) {
        return error(409, 'WIP_LIMIT_EXCEEDED', 'Превышен WIP-лимит')
      }
      const card = makeCard({
        id: `c${state.cards.length + 1}`,
        number: state.cards.length + 1,
        title: input.title,
        columnId: column.id,
        position: 99999,
      })
      state.cards.push(card)
      return HttpResponse.json(card, { status: 201 })
    }),
    http.get(api('/cards/:id'), ({ params }) =>
      HttpResponse.json(state.cards.find((c) => c.id === params.id)),
    ),
    http.patch(api('/cards/:id'), async ({ params, request }) => {
      const patch = (await request.json()) as Partial<Card>
      const card = state.cards.find((c) => c.id === params.id)!
      Object.assign(card, patch)
      return HttpResponse.json(card)
    }),
    http.post(api('/cards/:id/move'), async ({ params, request }) => {
      const { columnId } = (await request.json()) as { columnId: string }
      const card = state.cards.find((c) => c.id === params.id)!
      const column = state.board.columns.find((c) => c.id === columnId)!
      const count = state.cards.filter((c) => c.columnId === columnId && c.id !== card.id).length
      if (card.columnId !== columnId && column.wipLimit != null && count >= column.wipLimit) {
        return error(409, 'WIP_LIMIT_EXCEEDED', `Превышен WIP-лимит колонки «${column.name}»`)
      }
      card.columnId = columnId
      card.position = 99999
      return HttpResponse.json(card)
    }),
    http.delete(api('/cards/:id'), ({ params }) => {
      state.cards = state.cards.filter((c) => c.id !== params.id)
      return new HttpResponse(null, { status: 204 })
    }),
    http.get(api('/cards/:id/activity'), () => HttpResponse.json(state.activity)),
    http.get(api('/cards/:id/comments'), () => HttpResponse.json(state.comments)),
    http.post(api('/cards/:id/comments'), async ({ params, request }) => {
      const { body } = (await request.json()) as { body: string }
      const comment: Comment = {
        id: `m${state.comments.length + 1}`,
        cardId: String(params.id),
        authorId: state.members[0].id,
        body,
        createdAt: '2026-01-02T10:00:00Z',
        updatedAt: '2026-01-02T10:00:00Z',
      }
      state.comments.push(comment)
      return HttpResponse.json(comment, { status: 201 })
    }),
    http.patch(api('/comments/:id'), async ({ params, request }) => {
      const { body } = (await request.json()) as { body: string }
      const comment = state.comments.find((c) => c.id === params.id)!
      Object.assign(comment, { body, updatedAt: '2026-01-03T10:00:00Z' })
      return HttpResponse.json(comment)
    }),
    http.delete(api('/comments/:id'), ({ params }) => {
      state.comments = state.comments.filter((c) => c.id !== params.id)
      return new HttpResponse(null, { status: 204 })
    }),
    http.get(api('/cards/:id/attachments'), () => HttpResponse.json(state.attachments)),
    http.post(api('/cards/:id/attachments'), ({ params }) => {
      const attachment: Attachment = {
        id: `a${state.attachments.length + 1}`,
        cardId: String(params.id),
        uploaderId: state.members[0].id,
        filename: 'report.pdf',
        contentType: 'application/pdf',
        size: 2048,
        createdAt: '2026-01-02T10:00:00Z',
      }
      state.attachments.push(attachment)
      return HttpResponse.json(attachment, { status: 201 })
    }),
    http.delete(api('/attachments/:id'), ({ params }) => {
      state.attachments = state.attachments.filter((a) => a.id !== params.id)
      return new HttpResponse(null, { status: 204 })
    }),
    http.get(
      api('/attachments/:id/download'),
      () => new HttpResponse('file-bytes', { status: 200 }),
    ),
  )
  return state
}

import { baseApi } from '../../shared/api/baseApi'
import type {
  Activity,
  Attachment,
  BoardDetail,
  Card,
  CardInput,
  CardPage,
  CardPatch,
  Column,
  ColumnInput,
  ColumnPatch,
  Comment,
  Priority,
} from '../../shared/api/types'

export interface CardFilters {
  q?: string
  assigneeId?: string
  labelId?: string[]
  priority?: Priority
}

export interface ListCardsArgs extends CardFilters {
  boardId: string
}

/** Сколько карточек доски загружается за раз. */
export const BOARD_PAGE_SIZE = 500

function cardParams({ boardId: _boardId, ...filters }: ListCardsArgs) {
  const params = new URLSearchParams({ limit: String(BOARD_PAGE_SIZE) })
  if (filters.q?.trim()) params.set('q', filters.q.trim())
  if (filters.assigneeId) params.set('assigneeId', filters.assigneeId)
  if (filters.priority) params.set('priority', filters.priority)
  for (const id of filters.labelId ?? []) params.append('labelId', id)
  return params
}

export interface MoveCardArgs {
  card: Card
  columnId: string
  afterCardId: string | null
  /** Аргументы закэшированного списка, в котором карточку переставить сразу. */
  listArgs: ListCardsArgs
  /** Позиция для оптимистичного обновления. */
  position: number
}

export const boardApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    getBoard: build.query<BoardDetail, string>({
      query: (id) => `/boards/${id}`,
      providesTags: (_r, _e, id) => [{ type: 'Board', id }],
    }),
    renameBoard: build.mutation<void, { boardId: string; name: string }>({
      query: ({ boardId, name }) => ({
        url: `/boards/${boardId}`,
        method: 'PATCH',
        body: { name },
      }),
      invalidatesTags: (_r, _e, { boardId }) => [{ type: 'Board', id: boardId }],
    }),
    createColumn: build.mutation<Column, { boardId: string; input: ColumnInput }>({
      query: ({ boardId, input }) => ({
        url: `/boards/${boardId}/columns`,
        method: 'POST',
        body: input,
      }),
      invalidatesTags: (_r, _e, { boardId }) => [{ type: 'Board', id: boardId }],
    }),
    updateColumn: build.mutation<Column, { boardId: string; columnId: string; patch: ColumnPatch }>(
      {
        query: ({ columnId, patch }) => ({
          url: `/columns/${columnId}`,
          method: 'PATCH',
          body: patch,
        }),
        invalidatesTags: (_r, _e, { boardId }) => [{ type: 'Board', id: boardId }],
      },
    ),
    moveColumn: build.mutation<
      Column,
      { boardId: string; columnId: string; afterColumnId: string | null }
    >({
      query: ({ columnId, afterColumnId }) => ({
        url: `/columns/${columnId}/move`,
        method: 'POST',
        body: { afterColumnId },
      }),
      invalidatesTags: (_r, _e, { boardId }) => [{ type: 'Board', id: boardId }],
    }),
    deleteColumn: build.mutation<void, { boardId: string; columnId: string }>({
      query: ({ columnId }) => ({ url: `/columns/${columnId}`, method: 'DELETE' }),
      invalidatesTags: (_r, _e, { boardId }) => [{ type: 'Board', id: boardId }],
    }),

    listCards: build.query<CardPage, ListCardsArgs>({
      query: (args) => `/boards/${args.boardId}/cards?${cardParams(args).toString()}`,
      providesTags: (_r, _e, { boardId }) => [{ type: 'Cards', id: boardId }],
    }),
    getCard: build.query<Card, string>({
      query: (id) => `/cards/${id}`,
      providesTags: (_r, _e, id) => [{ type: 'Cards', id: `card-${id}` }],
    }),
    createCard: build.mutation<Card, { boardId: string; columnId: string; input: CardInput }>({
      query: ({ columnId, input }) => ({
        url: `/columns/${columnId}/cards`,
        method: 'POST',
        body: input,
      }),
      invalidatesTags: (_r, _e, { boardId }) => [
        { type: 'Cards', id: boardId },
        { type: 'Board', id: boardId },
      ],
    }),
    updateCard: build.mutation<Card, { card: Card; patch: CardPatch }>({
      query: ({ card, patch }) => ({ url: `/cards/${card.id}`, method: 'PATCH', body: patch }),
      invalidatesTags: (_r, _e, { card }) => [
        { type: 'Cards', id: card.boardId },
        { type: 'Cards', id: `card-${card.id}` },
        { type: 'Activity', id: card.id },
      ],
    }),
    moveCard: build.mutation<Card, MoveCardArgs>({
      query: ({ card, columnId, afterCardId }) => ({
        url: `/cards/${card.id}/move`,
        method: 'POST',
        body: { columnId, afterCardId },
      }),
      // Карточка переезжает сразу; при ошибке (например, WIP-лимит) возвращается на место.
      async onQueryStarted({ card, columnId, position, listArgs }, { dispatch, queryFulfilled }) {
        const patch = dispatch(
          boardApi.util.updateQueryData('listCards', listArgs, (draft) => {
            const target = draft.items.find((c) => c.id === card.id)
            if (target) {
              target.columnId = columnId
              target.position = position
            }
          }),
        )
        try {
          await queryFulfilled
        } catch {
          patch.undo()
        }
      },
      invalidatesTags: (_r, _e, { card }) => [
        { type: 'Cards', id: card.boardId },
        { type: 'Cards', id: `card-${card.id}` },
        { type: 'Board', id: card.boardId },
        { type: 'Activity', id: card.id },
      ],
    }),
    deleteCard: build.mutation<void, Card>({
      query: (card) => ({ url: `/cards/${card.id}`, method: 'DELETE' }),
      invalidatesTags: (_r, _e, card) => [
        { type: 'Cards', id: card.boardId },
        { type: 'Board', id: card.boardId },
      ],
    }),

    listActivity: build.query<Activity[], string>({
      query: (cardId) => `/cards/${cardId}/activity?limit=100`,
      providesTags: (_r, _e, cardId) => [{ type: 'Activity', id: cardId }],
    }),
    listComments: build.query<Comment[], string>({
      query: (cardId) => `/cards/${cardId}/comments`,
      providesTags: (_r, _e, cardId) => [{ type: 'Comment', id: cardId }],
    }),
    createComment: build.mutation<Comment, { card: Card; body: string }>({
      query: ({ card, body }) => ({
        url: `/cards/${card.id}/comments`,
        method: 'POST',
        body: { body },
      }),
      invalidatesTags: (_r, _e, { card }) => [
        { type: 'Comment', id: card.id },
        { type: 'Cards', id: card.boardId },
        { type: 'Activity', id: card.id },
      ],
    }),
    updateComment: build.mutation<Comment, { cardId: string; commentId: string; body: string }>({
      query: ({ commentId, body }) => ({
        url: `/comments/${commentId}`,
        method: 'PATCH',
        body: { body },
      }),
      invalidatesTags: (_r, _e, { cardId }) => [{ type: 'Comment', id: cardId }],
    }),
    deleteComment: build.mutation<void, { card: Card; commentId: string }>({
      query: ({ commentId }) => ({ url: `/comments/${commentId}`, method: 'DELETE' }),
      invalidatesTags: (_r, _e, { card }) => [
        { type: 'Comment', id: card.id },
        { type: 'Cards', id: card.boardId },
      ],
    }),
    listAttachments: build.query<Attachment[], string>({
      query: (cardId) => `/cards/${cardId}/attachments`,
      providesTags: (_r, _e, cardId) => [{ type: 'Attachment', id: cardId }],
    }),
    uploadAttachment: build.mutation<Attachment, { card: Card; file: File }>({
      query: ({ card, file }) => {
        const body = new FormData()
        body.append('file', file)
        return { url: `/cards/${card.id}/attachments`, method: 'POST', body }
      },
      invalidatesTags: (_r, _e, { card }) => [
        { type: 'Attachment', id: card.id },
        { type: 'Cards', id: card.boardId },
        { type: 'Activity', id: card.id },
      ],
    }),
    deleteAttachment: build.mutation<void, { card: Card; attachmentId: string }>({
      query: ({ attachmentId }) => ({ url: `/attachments/${attachmentId}`, method: 'DELETE' }),
      invalidatesTags: (_r, _e, { card }) => [
        { type: 'Attachment', id: card.id },
        { type: 'Cards', id: card.boardId },
        { type: 'Activity', id: card.id },
      ],
    }),
    downloadAttachment: build.mutation<Blob, string>({
      query: (attachmentId) => ({
        url: `/attachments/${attachmentId}/download`,
        responseHandler: (response) => response.blob(),
      }),
    }),
  }),
})

export const {
  useGetBoardQuery,
  useRenameBoardMutation,
  useCreateColumnMutation,
  useUpdateColumnMutation,
  useMoveColumnMutation,
  useDeleteColumnMutation,
  useListCardsQuery,
  useGetCardQuery,
  useCreateCardMutation,
  useUpdateCardMutation,
  useMoveCardMutation,
  useDeleteCardMutation,
  useListActivityQuery,
  useListCommentsQuery,
  useCreateCommentMutation,
  useUpdateCommentMutation,
  useDeleteCommentMutation,
  useListAttachmentsQuery,
  useUploadAttachmentMutation,
  useDeleteAttachmentMutation,
  useDownloadAttachmentMutation,
} = boardApi

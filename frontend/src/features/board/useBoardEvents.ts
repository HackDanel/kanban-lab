import { useEffect } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { baseApi } from '../../shared/api/baseApi'
import { selectAccessToken } from '../auth/authSlice'

interface BoardEvent {
  type: string
  boardId: string
  payload?: { cardId?: string }
}

const MAX_RETRY_DELAY = 30_000

export function boardSocketUrl(boardId: string, token: string, location: Location): string {
  const url = new URL('/api/v1/ws', location.href)
  url.protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  url.searchParams.set('boardId', boardId)
  url.searchParams.set('token', token)
  return url.toString()
}

/**
 * Слушает события доски по WebSocket и помечает устаревшими затронутые данные,
 * чтобы RTK Query перезапросил их. После обрыва переподключается с растущей паузой.
 */
export function useBoardEvents(boardId: string, onBoardDeleted?: () => void) {
  const token = useAppSelector(selectAccessToken)
  const dispatch = useAppDispatch()

  useEffect(() => {
    if (!token) return
    let socket: WebSocket | null = null
    let timer: ReturnType<typeof setTimeout> | undefined
    let attempt = 0
    let stopped = false

    const handle = (event: BoardEvent) => {
      if (event.type === 'board.deleted') {
        onBoardDeleted?.()
        return
      }
      const tags: {
        type: 'Cards' | 'Board' | 'Activity' | 'Comment' | 'Attachment'
        id: string
      }[] = [
        { type: 'Cards', id: boardId },
        { type: 'Board', id: boardId },
      ]
      const cardId = event.payload?.cardId
      if (cardId) {
        tags.push(
          { type: 'Cards', id: `card-${cardId}` },
          { type: 'Activity', id: cardId },
          { type: 'Comment', id: cardId },
          { type: 'Attachment', id: cardId },
        )
      }
      dispatch(baseApi.util.invalidateTags(tags))
    }

    const connect = () => {
      socket = new WebSocket(boardSocketUrl(boardId, token, window.location))
      socket.onopen = () => {
        attempt = 0
      }
      socket.onmessage = (message) => {
        try {
          handle(JSON.parse(String(message.data)) as BoardEvent)
        } catch {
          // некорректное сообщение пропускаем
        }
      }
      socket.onclose = () => {
        if (stopped) return
        const delay = Math.min(MAX_RETRY_DELAY, 1000 * 2 ** attempt)
        attempt += 1
        timer = setTimeout(connect, delay)
      }
    }
    connect()

    return () => {
      stopped = true
      clearTimeout(timer)
      socket?.close()
    }
  }, [boardId, token, dispatch, onBoardDeleted])
}

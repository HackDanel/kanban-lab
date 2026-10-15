import { useState } from 'react'
import type { FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useAppSelector } from '../../app/hooks'
import { getErrorMessage } from '../../shared/api/errors'
import type { Card, Comment, User } from '../../shared/api/types'
import { canManage } from '../../shared/lib/permissions'
import { selectCurrentUser } from '../auth/authSlice'
import {
  useCreateCommentMutation,
  useDeleteCommentMutation,
  useListCommentsQuery,
  useUpdateCommentMutation,
} from './boardApi'
import { formatDateTime } from './format'

function CommentItem({ card, comment, author }: { card: Card; comment: Comment; author?: User }) {
  const me = useAppSelector(selectCurrentUser)
  const [updateComment, update] = useUpdateCommentMutation()
  const [deleteComment, deletion] = useDeleteCommentMutation()
  const [editing, setEditing] = useState(false)
  const [body, setBody] = useState(comment.body)
  const [error, setError] = useState<string | null>(null)
  const own = me?.id === comment.authorId

  const save = async () => {
    if (body.trim() === '') {
      setError('Комментарий не может быть пустым')
      return
    }
    try {
      await updateComment({ cardId: card.id, commentId: comment.id, body }).unwrap()
      setEditing(false)
      setError(null)
    } catch (err) {
      setError(getErrorMessage(err))
    }
  }

  return (
    <ListItem disableGutters sx={{ display: 'block' }} data-testid="comment">
      <Typography variant="caption" color="text.secondary">
        {author?.name ?? 'Бывший участник'} · {formatDateTime(comment.createdAt)}
        {comment.updatedAt !== comment.createdAt && ' (изменён)'}
      </Typography>
      {editing ? (
        <Stack spacing={1} sx={{ mt: 0.5 }}>
          <TextField
            size="small"
            multiline
            value={body}
            onChange={(e) => setBody(e.target.value)}
            error={Boolean(error)}
            helperText={error}
            slotProps={{ htmlInput: { 'aria-label': 'Текст комментария' } }}
          />
          <Stack direction="row" spacing={1}>
            <Button
              size="small"
              variant="contained"
              disabled={update.isLoading}
              onClick={() => void save()}
            >
              Сохранить
            </Button>
            <Button size="small" onClick={() => setEditing(false)}>
              Отмена
            </Button>
          </Stack>
        </Stack>
      ) : (
        <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
          {comment.body}
        </Typography>
      )}
      {!editing && (
        <Stack direction="row" spacing={1}>
          {own && (
            <Button size="small" onClick={() => setEditing(true)}>
              Изменить
            </Button>
          )}
          {(own || canManage(me)) && (
            <Button
              size="small"
              color="error"
              disabled={deletion.isLoading}
              onClick={() => void deleteComment({ card, commentId: comment.id })}
            >
              Удалить
            </Button>
          )}
        </Stack>
      )}
    </ListItem>
  )
}

export function CommentsPanel({ card, members }: { card: Card; members: User[] }) {
  const { data: comments = [], error: loadError } = useListCommentsQuery(card.id)
  const [createComment, creation] = useCreateCommentMutation()
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)
  const byId = new Map(members.map((m) => [m.id, m]))

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    if (body.trim() === '') {
      setError('Комментарий не может быть пустым')
      return
    }
    try {
      await createComment({ card, body }).unwrap()
      setBody('')
      setError(null)
    } catch (err) {
      setError(getErrorMessage(err))
    }
  }

  return (
    <Stack spacing={2}>
      {loadError && <Alert severity="error">{getErrorMessage(loadError)}</Alert>}
      {comments.length === 0 ? (
        <Typography color="text.secondary">Комментариев пока нет.</Typography>
      ) : (
        <List dense>
          {comments.map((c) => (
            <CommentItem key={c.id} card={card} comment={c} author={byId.get(c.authorId)} />
          ))}
        </List>
      )}
      <Stack component="form" spacing={1} onSubmit={onSubmit} noValidate>
        <TextField
          size="small"
          multiline
          minRows={2}
          label="Новый комментарий"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          error={Boolean(error)}
          helperText={error}
        />
        <Button
          type="submit"
          variant="outlined"
          disabled={creation.isLoading}
          sx={{ alignSelf: 'flex-start' }}
        >
          Отправить
        </Button>
      </Stack>
    </Stack>
  )
}

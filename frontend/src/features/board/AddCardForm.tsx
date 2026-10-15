import { useState } from 'react'
import type { FormEvent } from 'react'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import { getErrorMessage, getFieldErrors } from '../../shared/api/errors'
import { useCreateCardMutation } from './boardApi'

interface AddCardFormProps {
  boardId: string
  columnId: string
  onDone: () => void
  onError: (message: string) => void
}

export function AddCardForm({ boardId, columnId, onDone, onError }: AddCardFormProps) {
  const [createCard, { isLoading }] = useCreateCardMutation()
  const [title, setTitle] = useState('')
  const [error, setError] = useState<string | null>(null)

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    if (title.trim() === '') {
      setError('Название обязательно')
      return
    }
    try {
      await createCard({ boardId, columnId, input: { title: title.trim() } }).unwrap()
      setTitle('')
      setError(null)
      onDone()
    } catch (err) {
      const field = getFieldErrors(err).title
      if (field) setError(field)
      else onError(getErrorMessage(err))
    }
  }

  return (
    <Stack component="form" spacing={1} onSubmit={onSubmit} noValidate>
      <TextField
        size="small"
        autoFocus
        multiline
        placeholder="Что нужно сделать?"
        slotProps={{ htmlInput: { 'aria-label': 'Название карточки', maxLength: 200 } }}
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault()
            e.currentTarget.closest('form')?.requestSubmit()
          }
          if (e.key === 'Escape') onDone()
        }}
        error={Boolean(error)}
        helperText={error}
      />
      <Stack direction="row" spacing={1}>
        <Button type="submit" size="small" variant="contained" disabled={isLoading}>
          Добавить
        </Button>
        <Button size="small" onClick={onDone}>
          Отмена
        </Button>
      </Stack>
    </Stack>
  )
}

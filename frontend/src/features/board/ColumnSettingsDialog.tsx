import { useState } from 'react'
import type { FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogTitle from '@mui/material/DialogTitle'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import { getErrorMessage } from '../../shared/api/errors'
import type { Column } from '../../shared/api/types'
import { useCreateColumnMutation, useUpdateColumnMutation } from './boardApi'
import { parseWipLimit } from './format'

interface ColumnSettingsDialogProps {
  boardId: string
  /** Колонка для редактирования; без неё диалог создаёт новую. */
  column?: Column
  onClose: () => void
}

export function ColumnSettingsDialog({ boardId, column, onClose }: ColumnSettingsDialogProps) {
  const [createColumn, creating] = useCreateColumnMutation()
  const [updateColumn, updating] = useUpdateColumnMutation()
  const [name, setName] = useState(column?.name ?? '')
  const [wip, setWip] = useState(column?.wipLimit != null ? String(column.wipLimit) : '')
  const [errors, setErrors] = useState<{ name?: string; wip?: string; form?: string }>({})

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    const limit = parseWipLimit(wip)
    const found = {
      name: name.trim() === '' ? 'Название обязательно' : undefined,
      wip: limit.error,
    }
    setErrors(found)
    if (found.name || found.wip) return
    try {
      if (column) {
        await updateColumn({
          boardId,
          columnId: column.id,
          patch: { name: name.trim(), wipLimit: limit.value },
        }).unwrap()
      } else {
        await createColumn({
          boardId,
          input: { name: name.trim(), wipLimit: limit.value },
        }).unwrap()
      }
      onClose()
    } catch (err) {
      setErrors({ form: getErrorMessage(err) })
    }
  }

  return (
    <Dialog open onClose={onClose} fullWidth maxWidth="xs">
      <form onSubmit={onSubmit} noValidate>
        <DialogTitle>{column ? 'Настройки колонки' : 'Новая колонка'}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            {errors.form && <Alert severity="error">{errors.form}</Alert>}
            <TextField
              label="Название"
              required
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              error={Boolean(errors.name)}
              helperText={errors.name}
            />
            <TextField
              label="WIP-лимит"
              type="number"
              value={wip}
              onChange={(e) => setWip(e.target.value)}
              error={Boolean(errors.wip)}
              helperText={errors.wip ?? 'Максимум карточек в колонке; пусто — без ограничения'}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>Отмена</Button>
          <Button
            type="submit"
            variant="contained"
            disabled={creating.isLoading || updating.isLoading}
          >
            Сохранить
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}

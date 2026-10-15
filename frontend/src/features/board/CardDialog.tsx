import { useState } from 'react'
import type { FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogTitle from '@mui/material/DialogTitle'
import MenuItem from '@mui/material/MenuItem'
import Stack from '@mui/material/Stack'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import TextField from '@mui/material/TextField'
import { getErrorMessage, getFieldErrors } from '../../shared/api/errors'
import type { Card, Column, Label, Priority, User } from '../../shared/api/types'
import { ConfirmDialog } from '../../shared/ui/ConfirmDialog'
import { ActivityList } from './ActivityList'
import { AttachmentsPanel } from './AttachmentsPanel'
import { useDeleteCardMutation, useUpdateCardMutation } from './boardApi'
import { CommentsPanel } from './CommentsPanel'
import { diffCard, PRIORITY_LABELS, toForm } from './format'
import type { CardForm } from './format'

interface CardDialogProps {
  card: Card
  projectKey: string
  columns: Column[]
  members: User[]
  labels: Label[]
  onMove: (card: Card, columnId: string) => void
  onClose: () => void
}

export function CardDialog({
  card,
  projectKey,
  columns,
  members,
  labels,
  onMove,
  onClose,
}: CardDialogProps) {
  const [updateCard, update] = useUpdateCardMutation()
  const [deleteCard, deletion] = useDeleteCardMutation()
  const [form, setForm] = useState(() => toForm(card))
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [tab, setTab] = useState(0)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const set = <K extends keyof CardForm>(key: K, value: CardForm[K]) => {
    setSaved(false)
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setFormError(null)
    if (form.title.trim() === '') {
      setErrors({ title: 'Название обязательно' })
      return
    }
    setErrors({})
    const patch = diffCard(card, form)
    if (Object.keys(patch).length === 0) {
      setSaved(true)
      return
    }
    try {
      await updateCard({ card, patch }).unwrap()
      setSaved(true)
    } catch (err) {
      const fields = getFieldErrors(err)
      if (Object.keys(fields).length > 0) setErrors(fields)
      else setFormError(getErrorMessage(err))
    }
  }

  const onDelete = async () => {
    try {
      await deleteCard(card).unwrap()
      onClose()
    } catch (err) {
      setConfirmDelete(false)
      setFormError(getErrorMessage(err))
    }
  }

  return (
    <Dialog open onClose={onClose} fullWidth maxWidth="md" scroll="paper">
      <DialogTitle>
        {projectKey}-{card.number}
      </DialogTitle>
      <DialogContent dividers>
        <Stack component="form" id="card-form" spacing={2} onSubmit={onSubmit} noValidate>
          {formError && <Alert severity="error">{formError}</Alert>}
          {saved && !update.isLoading && <Alert severity="success">Изменения сохранены</Alert>}
          <TextField
            label="Название"
            required
            value={form.title}
            onChange={(e) => set('title', e.target.value)}
            error={Boolean(errors.title)}
            helperText={errors.title}
            slotProps={{ htmlInput: { maxLength: 200 } }}
          />
          <TextField
            label="Описание"
            multiline
            minRows={3}
            value={form.description}
            onChange={(e) => set('description', e.target.value)}
            error={Boolean(errors.description)}
            helperText={errors.description}
          />
          <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' } }}>
            <TextField
              select
              label="Колонка"
              value={card.columnId}
              onChange={(e) => onMove(card, e.target.value)}
            >
              {columns.map((c) => (
                <MenuItem key={c.id} value={c.id}>
                  {c.name}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              select
              label="Приоритет"
              value={form.priority}
              onChange={(e) => set('priority', e.target.value as Priority)}
            >
              {(Object.keys(PRIORITY_LABELS) as Priority[]).map((p) => (
                <MenuItem key={p} value={p}>
                  {PRIORITY_LABELS[p]}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              select
              label="Исполнитель"
              value={form.assigneeId}
              onChange={(e) => set('assigneeId', e.target.value)}
              error={Boolean(errors.assigneeId)}
              helperText={errors.assigneeId}
            >
              <MenuItem value="">Не назначен</MenuItem>
              {members.map((m) => (
                <MenuItem key={m.id} value={m.id}>
                  {m.name}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Дедлайн"
              type="date"
              value={form.dueDate}
              onChange={(e) => set('dueDate', e.target.value)}
              slotProps={{ inputLabel: { shrink: true } }}
            />
          </Box>
          <TextField
            select
            label="Метки"
            value={form.labelIds}
            onChange={(e) => set('labelIds', e.target.value as unknown as string[])}
            slotProps={{ select: { multiple: true } }}
            helperText={labels.length === 0 ? 'В проекте пока нет меток' : undefined}
          >
            {labels.map((l) => (
              <MenuItem key={l.id} value={l.id}>
                {l.name}
              </MenuItem>
            ))}
          </TextField>
        </Stack>

        <Tabs value={tab} onChange={(_e, v: number) => setTab(v)} sx={{ mt: 3, mb: 2 }}>
          <Tab label={`Комментарии (${card.commentCount})`} />
          <Tab label={`Вложения (${card.attachmentCount})`} />
          <Tab label="История" />
        </Tabs>
        {tab === 0 && <CommentsPanel card={card} members={members} />}
        {tab === 1 && <AttachmentsPanel card={card} />}
        {tab === 2 && <ActivityList cardId={card.id} members={members} columns={columns} />}
      </DialogContent>
      <DialogActions>
        <Button color="error" onClick={() => setConfirmDelete(true)}>
          Удалить карточку
        </Button>
        <Box sx={{ flexGrow: 1 }} />
        <Button onClick={onClose}>Закрыть</Button>
        <Button type="submit" form="card-form" variant="contained" disabled={update.isLoading}>
          Сохранить
        </Button>
      </DialogActions>
      <ConfirmDialog
        open={confirmDelete}
        title="Удалить карточку?"
        text={`Карточка «${card.title}» будет удалена вместе с комментариями и вложениями.`}
        confirmLabel="Удалить"
        busy={deletion.isLoading}
        onConfirm={() => void onDelete()}
        onCancel={() => setConfirmDelete(false)}
      />
    </Dialog>
  )
}

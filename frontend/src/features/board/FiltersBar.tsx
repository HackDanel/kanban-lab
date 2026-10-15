import { useEffect, useState } from 'react'
import Button from '@mui/material/Button'
import MenuItem from '@mui/material/MenuItem'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import type { Label, Priority, User } from '../../shared/api/types'
import type { CardFilters } from './boardApi'
import { hasFilters, PRIORITY_LABELS } from './format'

interface FiltersBarProps {
  value: CardFilters
  onChange: (filters: CardFilters) => void
  members: User[]
  labels: Label[]
}

const SEARCH_DELAY = 300

export function FiltersBar({ value, onChange, members, labels }: FiltersBarProps) {
  const [query, setQuery] = useState(value.q ?? '')

  // Поиск отправляется после паузы в наборе, а не на каждую букву.
  useEffect(() => {
    if (query === (value.q ?? '')) return
    const timer = setTimeout(() => onChange({ ...value, q: query }), SEARCH_DELAY)
    return () => clearTimeout(timer)
  }, [query, value, onChange])

  return (
    <Stack
      component="search"
      direction={{ xs: 'column', md: 'row' }}
      spacing={1}
      sx={{ mb: 2, alignItems: { md: 'center' } }}
    >
      <TextField
        size="small"
        type="search"
        label="Поиск"
        placeholder="Название или описание"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        sx={{ minWidth: 220 }}
      />
      <TextField
        select
        size="small"
        label="Исполнитель"
        value={value.assigneeId ?? ''}
        onChange={(e) => onChange({ ...value, assigneeId: e.target.value || undefined })}
        sx={{ minWidth: 180 }}
      >
        <MenuItem value="">Все</MenuItem>
        {members.map((m) => (
          <MenuItem key={m.id} value={m.id}>
            {m.name}
          </MenuItem>
        ))}
      </TextField>
      <TextField
        select
        size="small"
        label="Приоритет"
        value={value.priority ?? ''}
        onChange={(e) =>
          onChange({ ...value, priority: (e.target.value || undefined) as Priority | undefined })
        }
        sx={{ minWidth: 160 }}
      >
        <MenuItem value="">Любой</MenuItem>
        {(Object.keys(PRIORITY_LABELS) as Priority[]).map((p) => (
          <MenuItem key={p} value={p}>
            {PRIORITY_LABELS[p]}
          </MenuItem>
        ))}
      </TextField>
      <TextField
        select
        size="small"
        label="Метки"
        value={value.labelId ?? []}
        onChange={(e) => {
          const v = e.target.value as unknown as string[]
          onChange({ ...value, labelId: v.length > 0 ? v : undefined })
        }}
        slotProps={{ select: { multiple: true } }}
        sx={{ minWidth: 160 }}
      >
        {labels.map((l) => (
          <MenuItem key={l.id} value={l.id}>
            {l.name}
          </MenuItem>
        ))}
      </TextField>
      {hasFilters(value) && (
        <Button
          onClick={() => {
            setQuery('')
            onChange({})
          }}
        >
          Сбросить
        </Button>
      )}
    </Stack>
  )
}

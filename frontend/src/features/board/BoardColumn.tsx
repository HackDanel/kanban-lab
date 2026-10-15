import { useState } from 'react'
import { useDroppable } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import IconButton from '@mui/material/IconButton'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import Add from '@mui/icons-material/Add'
import MoreVert from '@mui/icons-material/MoreVert'
import type { Card, Column, Label, User } from '../../shared/api/types'
import { AddCardForm } from './AddCardForm'
import { CardTile } from './CardTile'
import { isColumnFull } from './ordering'

interface BoardColumnProps {
  boardId: string
  column: Column
  cards: Card[]
  projectKey: string
  labels: Map<string, Label>
  members: Map<string, User>
  canManage: boolean
  isFirst: boolean
  isLast: boolean
  onOpenCard: (card: Card) => void
  onEdit: (column: Column) => void
  onDelete: (column: Column) => void
  onShift: (column: Column, direction: -1 | 1) => void
  onError: (message: string) => void
}

export function BoardColumn(props: BoardColumnProps) {
  const { boardId, column, cards, canManage, onError } = props
  const { setNodeRef, isOver } = useDroppable({ id: column.id })
  const [adding, setAdding] = useState(false)
  const [menu, setMenu] = useState<HTMLElement | null>(null)
  const full = isColumnFull(column, column.cardCount)

  const act = (fn: () => void) => () => {
    setMenu(null)
    fn()
  }

  return (
    <Paper
      component="section"
      aria-label={`Колонка ${column.name}`}
      data-testid="column"
      data-column-id={column.id}
      variant="outlined"
      sx={{
        width: 300,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        maxHeight: '100%',
        bgcolor: isOver ? 'action.hover' : 'grey.50',
      }}
    >
      <Stack direction="row" sx={{ alignItems: 'center', px: 1.5, pt: 1, gap: 1 }}>
        <Typography
          component="h2"
          variant="subtitle2"
          sx={{ flexGrow: 1, textTransform: 'uppercase' }}
        >
          {column.name}
        </Typography>
        <Typography
          variant="caption"
          color={full ? 'error' : 'text.secondary'}
          title={column.wipLimit != null ? 'Карточек / WIP-лимит' : 'Карточек'}
          data-testid="column-count"
        >
          {column.cardCount}
          {column.wipLimit != null && ` / ${column.wipLimit}`}
        </Typography>
        {canManage && (
          <>
            <IconButton
              size="small"
              aria-label={`Действия с колонкой ${column.name}`}
              onClick={(e) => setMenu(e.currentTarget)}
            >
              <MoreVert fontSize="small" />
            </IconButton>
            <Menu anchorEl={menu} open={menu !== null} onClose={() => setMenu(null)}>
              <MenuItem onClick={act(() => props.onEdit(column))}>Настроить</MenuItem>
              <MenuItem disabled={props.isFirst} onClick={act(() => props.onShift(column, -1))}>
                Сдвинуть влево
              </MenuItem>
              <MenuItem disabled={props.isLast} onClick={act(() => props.onShift(column, 1))}>
                Сдвинуть вправо
              </MenuItem>
              <MenuItem onClick={act(() => props.onDelete(column))} sx={{ color: 'error.main' }}>
                Удалить
              </MenuItem>
            </Menu>
          </>
        )}
      </Stack>

      <SortableContext items={cards.map((c) => c.id)} strategy={verticalListSortingStrategy}>
        <Stack
          ref={setNodeRef}
          spacing={1}
          sx={{ p: 1, overflowY: 'auto', minHeight: 60, flexGrow: 1 }}
        >
          {cards.map((card) => (
            <CardTile
              key={card.id}
              card={card}
              projectKey={props.projectKey}
              labels={props.labels}
              members={props.members}
              onOpen={props.onOpenCard}
            />
          ))}
        </Stack>
      </SortableContext>

      <Box sx={{ p: 1, pt: 0 }}>
        {adding ? (
          <AddCardForm
            boardId={boardId}
            columnId={column.id}
            onDone={() => setAdding(false)}
            onError={onError}
          />
        ) : (
          <Button
            fullWidth
            size="small"
            startIcon={<Add />}
            onClick={() => setAdding(true)}
            disabled={full}
            title={full ? 'Колонка заполнена до WIP-лимита' : undefined}
            aria-label={`Добавить карточку в колонку ${column.name}`}
            sx={{ justifyContent: 'flex-start' }}
          >
            Добавить карточку
          </Button>
        )}
      </Box>
    </Paper>
  )
}

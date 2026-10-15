import { useCallback, useMemo, useState } from 'react'
import { Link as RouterLink, useNavigate, useParams } from 'react-router'
import {
  closestCorners,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import type { DragEndEvent } from '@dnd-kit/core'
import { sortableKeyboardCoordinates } from '@dnd-kit/sortable'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Breadcrumbs from '@mui/material/Breadcrumbs'
import Button from '@mui/material/Button'
import CircularProgress from '@mui/material/CircularProgress'
import Link from '@mui/material/Link'
import Snackbar from '@mui/material/Snackbar'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import Add from '@mui/icons-material/Add'
import { useAppSelector } from '../app/hooks'
import { selectCurrentUser } from '../features/auth/authSlice'
import { BoardColumn } from '../features/board/BoardColumn'
import {
  BOARD_PAGE_SIZE,
  useDeleteColumnMutation,
  useGetBoardQuery,
  useListCardsQuery,
  useMoveCardMutation,
  useMoveColumnMutation,
} from '../features/board/boardApi'
import type { CardFilters, ListCardsArgs } from '../features/board/boardApi'
import { CardDialog } from '../features/board/CardDialog'
import { ColumnSettingsDialog } from '../features/board/ColumnSettingsDialog'
import { FiltersBar } from '../features/board/FiltersBar'
import { hasFilters } from '../features/board/format'
import {
  computeMove,
  groupByColumn,
  positionBetween,
  sortColumns,
} from '../features/board/ordering'
import { useBoardEvents } from '../features/board/useBoardEvents'
import {
  useGetProjectQuery,
  useListLabelsQuery,
  useListMembersQuery,
} from '../features/projects/projectApi'
import { getErrorCode, getErrorMessage } from '../shared/api/errors'
import type { Card, Column } from '../shared/api/types'
import { canManage } from '../shared/lib/permissions'
import { ConfirmDialog } from '../shared/ui/ConfirmDialog'

export function BoardPage() {
  const { boardId = '' } = useParams()
  const navigate = useNavigate()
  const user = useAppSelector(selectCurrentUser)
  const [filters, setFilters] = useState<CardFilters>({})
  const [openCardId, setOpenCardId] = useState<string | null>(null)
  const [editingColumn, setEditingColumn] = useState<Column | 'new' | null>(null)
  const [deletingColumn, setDeletingColumn] = useState<Column | null>(null)
  const [message, setMessage] = useState<{ text: string; severity: 'error' | 'warning' } | null>(
    null,
  )

  const board = useGetBoardQuery(boardId)
  const projectId = board.data?.projectId ?? ''
  const skip = projectId === ''
  const { data: project } = useGetProjectQuery(projectId, { skip })
  const { data: members = [] } = useListMembersQuery(projectId, { skip })
  const { data: labels = [] } = useListLabelsQuery(projectId, { skip })
  const listArgs: ListCardsArgs = useMemo(() => ({ boardId, ...filters }), [boardId, filters])
  const cards = useListCardsQuery(listArgs)
  const [moveCard] = useMoveCardMutation()
  const [moveColumn] = useMoveColumnMutation()
  const [deleteColumn, columnDeletion] = useDeleteColumnMutation()

  const onBoardDeleted = useCallback(() => {
    void navigate(projectId ? `/projects/${projectId}` : '/', { replace: true })
  }, [navigate, projectId])
  useBoardEvents(boardId, onBoardDeleted)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const columns = useMemo(() => sortColumns(board.data?.columns ?? []), [board.data])
  const groups = useMemo(
    () => groupByColumn(cards.data?.items ?? [], columns),
    [cards.data, columns],
  )
  const labelMap = useMemo(() => new Map(labels.map((l) => [l.id, l])), [labels])
  const memberMap = useMemo(() => new Map(members.map((m) => [m.id, m])), [members])
  const openCard = cards.data?.items.find((c) => c.id === openCardId) ?? null
  const manage = canManage(user)

  const showError = (err: unknown) =>
    setMessage({
      text: getErrorMessage(err),
      severity: getErrorCode(err) === 'WIP_LIMIT_EXCEEDED' ? 'warning' : 'error',
    })

  const move = async (
    card: Card,
    columnId: string,
    afterCardId: string | null,
    position: number,
  ) => {
    try {
      await moveCard({ card, columnId, afterCardId, position, listArgs }).unwrap()
    } catch (err) {
      showError(err)
    }
  }

  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over) return
    const card = cards.data?.items.find((c) => c.id === active.id)
    const target = computeMove(groups, String(active.id), String(over.id))
    if (card && target) void move(card, target.columnId, target.afterCardId, target.position)
  }

  // Перемещение из диалога карточки: в конец выбранной колонки.
  const moveToColumn = (card: Card, columnId: string) => {
    if (columnId === card.columnId) return
    const list = groups[columnId] ?? []
    const last = list[list.length - 1]
    void move(card, columnId, last?.id ?? null, positionBetween(last))
  }

  const shiftColumn = async (column: Column, direction: -1 | 1) => {
    const index = columns.findIndex((c) => c.id === column.id)
    // Влево: после колонки, стоящей через одну слева; вправо: после правой соседки.
    const after = direction === -1 ? (columns[index - 2] ?? null) : columns[index + 1]
    try {
      await moveColumn({ boardId, columnId: column.id, afterColumnId: after?.id ?? null }).unwrap()
    } catch (err) {
      showError(err)
    }
  }

  const confirmDeleteColumn = async () => {
    if (!deletingColumn) return
    try {
      await deleteColumn({ boardId, columnId: deletingColumn.id }).unwrap()
    } catch (err) {
      showError(err)
    } finally {
      setDeletingColumn(null)
    }
  }

  if (board.isLoading) return <CircularProgress aria-label="Загрузка доски" />
  if (board.error || !board.data) {
    return <Alert severity="error">{getErrorMessage(board.error)}</Alert>
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)' }}>
      <Breadcrumbs sx={{ mb: 1 }}>
        <Link component={RouterLink} to="/" underline="hover" color="inherit">
          Проекты
        </Link>
        {project && (
          <Link
            component={RouterLink}
            to={`/projects/${project.id}`}
            underline="hover"
            color="inherit"
          >
            {project.name}
          </Link>
        )}
        <Typography color="text.primary">{board.data.name}</Typography>
      </Breadcrumbs>
      <Stack direction="row" sx={{ mb: 2, alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography component="h1" variant="h5">
          {board.data.name}
        </Typography>
        {manage && (
          <Button startIcon={<Add />} onClick={() => setEditingColumn('new')}>
            Колонка
          </Button>
        )}
      </Stack>

      <FiltersBar value={filters} onChange={setFilters} members={members} labels={labels} />
      {cards.error && <Alert severity="error">{getErrorMessage(cards.error)}</Alert>}
      {cards.data && cards.data.total > BOARD_PAGE_SIZE && (
        <Alert severity="info" sx={{ mb: 1 }}>
          Показаны первые {BOARD_PAGE_SIZE} из {cards.data.total} карточек. Уточните фильтры.
        </Alert>
      )}
      {cards.data && hasFilters(filters) && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }} role="status">
          Найдено карточек: {cards.data.total}
        </Typography>
      )}

      <DndContext sensors={sensors} collisionDetection={closestCorners} onDragEnd={onDragEnd}>
        <Stack
          direction="row"
          spacing={2}
          sx={{ flexGrow: 1, overflowX: 'auto', pb: 1, alignItems: 'flex-start' }}
        >
          {columns.map((column, i) => (
            <BoardColumn
              key={column.id}
              boardId={boardId}
              column={column}
              cards={groups[column.id] ?? []}
              projectKey={project?.key ?? ''}
              labels={labelMap}
              members={memberMap}
              canManage={manage}
              isFirst={i === 0}
              isLast={i === columns.length - 1}
              onOpenCard={(card) => setOpenCardId(card.id)}
              onEdit={setEditingColumn}
              onDelete={setDeletingColumn}
              onShift={(c, d) => void shiftColumn(c, d)}
              onError={(text) => setMessage({ text, severity: 'error' })}
            />
          ))}
        </Stack>
      </DndContext>

      {openCard && (
        <CardDialog
          key={openCard.id}
          card={openCard}
          projectKey={project?.key ?? ''}
          columns={columns}
          members={members}
          labels={labels}
          onMove={moveToColumn}
          onClose={() => setOpenCardId(null)}
        />
      )}
      {editingColumn && (
        <ColumnSettingsDialog
          boardId={boardId}
          column={editingColumn === 'new' ? undefined : editingColumn}
          onClose={() => setEditingColumn(null)}
        />
      )}
      <ConfirmDialog
        open={deletingColumn !== null}
        title="Удалить колонку?"
        text={`Колонка «${deletingColumn?.name ?? ''}» будет удалена. Удалить можно только пустую колонку.`}
        confirmLabel="Удалить"
        busy={columnDeletion.isLoading}
        onConfirm={() => void confirmDeleteColumn()}
        onCancel={() => setDeletingColumn(null)}
      />
      <Snackbar
        open={message !== null}
        autoHideDuration={6000}
        onClose={() => setMessage(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert
          severity={message?.severity ?? 'error'}
          onClose={() => setMessage(null)}
          variant="filled"
        >
          {message?.text}
        </Alert>
      </Snackbar>
    </Box>
  )
}

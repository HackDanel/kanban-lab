import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link as RouterLink, useParams } from 'react-router'
import Alert from '@mui/material/Alert'
import Autocomplete from '@mui/material/Autocomplete'
import Box from '@mui/material/Box'
import Breadcrumbs from '@mui/material/Breadcrumbs'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardActionArea from '@mui/material/CardActionArea'
import CardContent from '@mui/material/CardContent'
import CircularProgress from '@mui/material/CircularProgress'
import IconButton from '@mui/material/IconButton'
import Link from '@mui/material/Link'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemText from '@mui/material/ListItemText'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import DeleteOutlined from '@mui/icons-material/DeleteOutlined'
import { useAppSelector } from '../app/hooks'
import { selectCurrentUser } from '../features/auth/authSlice'
import { LabelChip } from '../features/board/labels'
import {
  useAddMemberMutation,
  useCreateBoardMutation,
  useCreateLabelMutation,
  useDeleteBoardMutation,
  useDeleteLabelMutation,
  useGetProjectQuery,
  useListBoardsQuery,
  useListLabelsQuery,
  useListMembersQuery,
  useRemoveMemberMutation,
  useSearchUsersQuery,
} from '../features/projects/projectApi'
import { getErrorMessage, getFieldErrors } from '../shared/api/errors'
import type { Board, Project, User } from '../shared/api/types'
import { canEditProject, canManage, ROLE_LABELS } from '../shared/lib/permissions'
import { ConfirmDialog } from '../shared/ui/ConfirmDialog'

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Paper component="section" variant="outlined" sx={{ p: 2 }} aria-label={title}>
      <Typography component="h2" variant="h6" sx={{ mb: 2 }}>
        {title}
      </Typography>
      {children}
    </Paper>
  )
}

function BoardsSection({ project, manage }: { project: Project; manage: boolean }) {
  const { data: boards = [] } = useListBoardsQuery(project.id)
  const [createBoard, creation] = useCreateBoardMutation()
  const [deleteBoard, deletion] = useDeleteBoardMutation()
  const [name, setName] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [toDelete, setToDelete] = useState<Board | null>(null)

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    if (name.trim() === '') {
      setError('Название обязательно')
      return
    }
    try {
      await createBoard({ projectId: project.id, name: name.trim() }).unwrap()
      setName('')
      setError(null)
    } catch (err) {
      setError(getErrorMessage(err))
    }
  }

  return (
    <Section title="Доски">
      {boards.length === 0 && <Typography color="text.secondary">Досок пока нет.</Typography>}
      <Box
        sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' } }}
      >
        {boards.map((b) => (
          <Card key={b.id} variant="outlined" sx={{ display: 'flex', alignItems: 'center' }}>
            <CardActionArea component={RouterLink} to={`/boards/${b.id}`}>
              <CardContent>
                <Typography variant="subtitle1">{b.name}</Typography>
              </CardContent>
            </CardActionArea>
            {manage && (
              <IconButton
                aria-label={`Удалить доску ${b.name}`}
                onClick={() => setToDelete(b)}
                sx={{ mr: 1 }}
              >
                <DeleteOutlined />
              </IconButton>
            )}
          </Card>
        ))}
      </Box>
      {manage && (
        <Stack
          component="form"
          direction="row"
          spacing={1}
          sx={{ mt: 2 }}
          onSubmit={onSubmit}
          noValidate
        >
          <TextField
            size="small"
            label="Новая доска"
            value={name}
            onChange={(e) => setName(e.target.value)}
            error={Boolean(error)}
            helperText={error}
          />
          <Button
            type="submit"
            variant="contained"
            disabled={creation.isLoading}
            sx={{ alignSelf: 'flex-start' }}
          >
            Создать доску
          </Button>
        </Stack>
      )}
      <ConfirmDialog
        open={toDelete !== null}
        title="Удалить доску?"
        text={`Доска «${toDelete?.name ?? ''}» будет удалена вместе с карточками.`}
        confirmLabel="Удалить"
        busy={deletion.isLoading}
        onConfirm={() => {
          if (toDelete)
            void deleteBoard({ projectId: project.id, boardId: toDelete.id }).finally(() =>
              setToDelete(null),
            )
        }}
        onCancel={() => setToDelete(null)}
      />
    </Section>
  )
}

function MembersSection({ project, owner }: { project: Project; owner: boolean }) {
  const { data: members = [] } = useListMembersQuery(project.id)
  const [query, setQuery] = useState('')
  const { data: found } = useSearchUsersQuery(query, { skip: !owner || query.trim().length < 2 })
  const [addMember] = useAddMemberMutation()
  const [removeMember] = useRemoveMemberMutation()
  const [error, setError] = useState<string | null>(null)
  const memberIds = new Set(members.map((m) => m.id))
  const candidates = (found?.items ?? []).filter((u) => !memberIds.has(u.id))

  const run = async (action: Promise<unknown>) => {
    setError(null)
    try {
      await action
    } catch (err) {
      setError(getErrorMessage(err))
    }
  }

  return (
    <Section title="Участники">
      {error && (
        <Alert severity="error" sx={{ mb: 1 }}>
          {error}
        </Alert>
      )}
      <List dense>
        {members.map((m) => (
          <ListItem
            key={m.id}
            disableGutters
            secondaryAction={
              owner &&
              m.id !== project.ownerId && (
                <IconButton
                  aria-label={`Исключить ${m.name}`}
                  onClick={() =>
                    void run(removeMember({ projectId: project.id, userId: m.id }).unwrap())
                  }
                >
                  <DeleteOutlined />
                </IconButton>
              )
            }
          >
            <ListItemText
              primary={m.id === project.ownerId ? `${m.name} (владелец)` : m.name}
              secondary={`${m.email} · ${ROLE_LABELS[m.role]}`}
            />
          </ListItem>
        ))}
      </List>
      {owner && (
        <Autocomplete<User>
          options={candidates}
          getOptionLabel={(u) => `${u.name} (${u.email})`}
          filterOptions={(x) => x}
          inputValue={query}
          onInputChange={(_e, v) => setQuery(v)}
          onChange={(_e, user) => {
            if (user)
              void run(addMember({ projectId: project.id, userId: user.id }).unwrap()).then(() =>
                setQuery(''),
              )
          }}
          value={null}
          noOptionsText={query.trim().length < 2 ? 'Введите имя или email' : 'Никого не найдено'}
          renderInput={(params) => (
            <TextField {...params} size="small" label="Добавить участника" />
          )}
          sx={{ mt: 1, maxWidth: 400 }}
        />
      )}
    </Section>
  )
}

const DEFAULT_COLOR = '#0c66e4'

function LabelsSection({ project, manage }: { project: Project; manage: boolean }) {
  const { data: labels = [] } = useListLabelsQuery(project.id)
  const [createLabel, creation] = useCreateLabelMutation()
  const [deleteLabel] = useDeleteLabelMutation()
  const [name, setName] = useState('')
  const [color, setColor] = useState(DEFAULT_COLOR)
  const [error, setError] = useState<string | null>(null)

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    if (name.trim() === '') {
      setError('Название обязательно')
      return
    }
    try {
      await createLabel({ projectId: project.id, input: { name: name.trim(), color } }).unwrap()
      setName('')
      setError(null)
    } catch (err) {
      setError(getFieldErrors(err).name ?? getErrorMessage(err))
    }
  }

  return (
    <Section title="Метки">
      {labels.length === 0 && <Typography color="text.secondary">Меток пока нет.</Typography>}
      <Stack direction="row" sx={{ gap: 1, flexWrap: 'wrap' }}>
        {labels.map((l) => (
          <LabelChip
            key={l.id}
            label={l}
            onDelete={
              manage ? () => void deleteLabel({ projectId: project.id, labelId: l.id }) : undefined
            }
          />
        ))}
      </Stack>
      {manage && (
        <Stack
          component="form"
          direction="row"
          spacing={1}
          sx={{ mt: 2 }}
          onSubmit={onSubmit}
          noValidate
        >
          <TextField
            size="small"
            label="Новая метка"
            value={name}
            onChange={(e) => setName(e.target.value)}
            error={Boolean(error)}
            helperText={error}
          />
          <TextField
            size="small"
            type="color"
            label="Цвет"
            value={color}
            onChange={(e) => setColor(e.target.value)}
            sx={{ width: 80 }}
          />
          <Button
            type="submit"
            variant="outlined"
            disabled={creation.isLoading}
            sx={{ alignSelf: 'flex-start' }}
          >
            Добавить
          </Button>
        </Stack>
      )}
    </Section>
  )
}

export function ProjectPage() {
  const { projectId = '' } = useParams()
  const user = useAppSelector(selectCurrentUser)
  const { data: project, isLoading, error } = useGetProjectQuery(projectId)

  if (isLoading) return <CircularProgress aria-label="Загрузка проекта" />
  if (error || !project) return <Alert severity="error">{getErrorMessage(error)}</Alert>

  return (
    <>
      <Breadcrumbs sx={{ mb: 1 }}>
        <Link component={RouterLink} to="/" underline="hover" color="inherit">
          Проекты
        </Link>
        <Typography color="text.primary">{project.name}</Typography>
      </Breadcrumbs>
      <Typography variant="overline" color="text.secondary">
        {project.key}
      </Typography>
      <Typography component="h1" variant="h4">
        {project.name}
      </Typography>
      {project.description && (
        <Typography color="text.secondary" sx={{ mt: 1 }}>
          {project.description}
        </Typography>
      )}
      <Stack spacing={2} sx={{ mt: 3 }}>
        <BoardsSection project={project} manage={canManage(user)} />
        <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' } }}>
          <MembersSection project={project} owner={canEditProject(user, project)} />
          <LabelsSection project={project} manage={canManage(user)} />
        </Box>
      </Stack>
    </>
  )
}

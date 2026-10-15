import { useState } from 'react'
import { Link as RouterLink } from 'react-router'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardActionArea from '@mui/material/CardActionArea'
import CardActions from '@mui/material/CardActions'
import CardContent from '@mui/material/CardContent'
import CircularProgress from '@mui/material/CircularProgress'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useAppSelector } from '../app/hooks'
import { selectCurrentUser } from '../features/auth/authSlice'
import { CreateProjectDialog } from '../features/projects/CreateProjectDialog'
import { useDeleteProjectMutation, useListProjectsQuery } from '../features/projects/projectsApi'
import { getErrorMessage } from '../shared/api/errors'
import type { Project } from '../shared/api/types'
import { canEditProject, canManage } from '../shared/lib/permissions'
import { ConfirmDialog } from '../shared/ui/ConfirmDialog'

export function ProjectsPage() {
  const user = useAppSelector(selectCurrentUser)
  const { data: projects, isLoading, error } = useListProjectsQuery()
  const [deleteProject, deletion] = useDeleteProjectMutation()
  const [creating, setCreating] = useState(false)
  const [toDelete, setToDelete] = useState<Project | null>(null)

  const confirmDelete = async () => {
    if (toDelete === null) return
    try {
      await deleteProject(toDelete.id).unwrap()
    } finally {
      setToDelete(null)
    }
  }

  return (
    <>
      <Stack direction="row" sx={{ mb: 3, alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography component="h1" variant="h4">
          Проекты
        </Typography>
        {canManage(user) && (
          <Button variant="contained" onClick={() => setCreating(true)}>
            Создать проект
          </Button>
        )}
      </Stack>

      {isLoading && <CircularProgress aria-label="Загрузка проектов" />}
      {error && <Alert severity="error">{getErrorMessage(error)}</Alert>}
      {deletion.error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {getErrorMessage(deletion.error)}
        </Alert>
      )}
      {projects?.length === 0 && (
        <Typography color="text.secondary">
          {canManage(user)
            ? 'Проектов пока нет. Создайте первый.'
            : 'Вас пока не добавили ни в один проект.'}
        </Typography>
      )}

      <Box
        component="ul"
        sx={{
          display: 'grid',
          gap: 2,
          gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(3, 1fr)' },
          listStyle: 'none',
          p: 0,
          m: 0,
        }}
      >
        {projects?.map((project) => (
          <Card key={project.id} component="li" variant="outlined">
            <CardActionArea component={RouterLink} to={`/projects/${project.id}`}>
              <CardContent>
                <Typography variant="overline" color="text.secondary">
                  {project.key}
                </Typography>
                <Typography component="h2" variant="h6">
                  {project.name}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {project.description}
                </Typography>
              </CardContent>
            </CardActionArea>
            {canEditProject(user, project) && (
              <CardActions>
                <Button
                  size="small"
                  color="error"
                  aria-label={`Удалить проект ${project.name}`}
                  onClick={() => setToDelete(project)}
                >
                  Удалить
                </Button>
              </CardActions>
            )}
          </Card>
        ))}
      </Box>

      <CreateProjectDialog open={creating} onClose={() => setCreating(false)} />
      <ConfirmDialog
        open={toDelete !== null}
        title="Удалить проект?"
        text={`Проект «${toDelete?.name ?? ''}» будет удалён вместе с досками и карточками.`}
        confirmLabel="Удалить"
        busy={deletion.isLoading}
        onConfirm={() => void confirmDelete()}
        onCancel={() => setToDelete(null)}
      />
    </>
  )
}

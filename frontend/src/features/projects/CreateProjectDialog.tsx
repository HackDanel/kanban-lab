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
import { getErrorCode, getErrorMessage, getFieldErrors } from '../../shared/api/errors'
import { useCreateProjectMutation } from './projectsApi'
import { validateProject } from './validation'
import type { ProjectErrors } from './validation'

const EMPTY = { key: '', name: '', description: '' }

export function CreateProjectDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [createProject, { isLoading }] = useCreateProjectMutation()
  const [values, setValues] = useState(EMPTY)
  const [errors, setErrors] = useState<ProjectErrors>({})
  const [formError, setFormError] = useState<string | null>(null)

  const close = () => {
    setValues(EMPTY)
    setErrors({})
    setFormError(null)
    onClose()
  }

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setFormError(null)
    const input = { ...values, name: values.name.trim() }
    const found = validateProject(input)
    setErrors(found)
    if (Object.keys(found).length > 0) return

    try {
      await createProject(input).unwrap()
      close()
    } catch (error) {
      if (getErrorCode(error) === 'CONFLICT') {
        setErrors({ key: 'Проект с таким ключом уже существует' })
        return
      }
      const fields = getFieldErrors(error)
      if (Object.keys(fields).length > 0) setErrors(fields)
      else setFormError(getErrorMessage(error))
    }
  }

  return (
    <Dialog open={open} onClose={close} fullWidth maxWidth="sm">
      <form onSubmit={onSubmit} noValidate>
        <DialogTitle>Новый проект</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            {formError && <Alert severity="error">{formError}</Alert>}
            <TextField
              label="Ключ"
              required
              autoFocus
              value={values.key}
              onChange={(e) => setValues({ ...values, key: e.target.value.toUpperCase() })}
              error={Boolean(errors.key)}
              helperText={errors.key ?? 'Например, KAN'}
            />
            <TextField
              label="Название"
              required
              value={values.name}
              onChange={(e) => setValues({ ...values, name: e.target.value })}
              error={Boolean(errors.name)}
              helperText={errors.name}
            />
            <TextField
              label="Описание"
              multiline
              minRows={3}
              value={values.description}
              onChange={(e) => setValues({ ...values, description: e.target.value })}
              error={Boolean(errors.description)}
              helperText={errors.description}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={close}>Отмена</Button>
          <Button type="submit" variant="contained" disabled={isLoading}>
            Создать
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}

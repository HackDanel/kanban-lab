import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link as RouterLink, useNavigate } from 'react-router'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Link from '@mui/material/Link'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import { useRegisterMutation } from '../features/auth/authApi'
import { validateRegistration } from '../features/auth/validation'
import type { RegisterErrors } from '../features/auth/validation'
import { getErrorCode, getErrorMessage, getFieldErrors } from '../shared/api/errors'
import { AuthCard } from '../shared/ui/AuthCard'

export function RegisterPage() {
  const navigate = useNavigate()
  const [register, { isLoading }] = useRegisterMutation()
  const [values, setValues] = useState({ name: '', email: '', password: '' })
  const [errors, setErrors] = useState<RegisterErrors>({})
  const [formError, setFormError] = useState<string | null>(null)

  const field = (name: keyof typeof values) => ({
    value: values[name],
    onChange: (e: { target: { value: string } }) =>
      setValues((prev) => ({ ...prev, [name]: e.target.value })),
    error: Boolean(errors[name]),
    helperText: errors[name],
  })

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setFormError(null)
    const found = validateRegistration(values)
    setErrors(found)
    if (Object.keys(found).length > 0) return

    try {
      await register({
        name: values.name.trim(),
        email: values.email.trim(),
        password: values.password,
      }).unwrap()
      void navigate('/login', { state: { registered: true } })
    } catch (error) {
      if (getErrorCode(error) === 'CONFLICT') {
        setErrors({ email: 'Пользователь с таким email уже зарегистрирован' })
        return
      }
      const fields = getFieldErrors(error)
      if (Object.keys(fields).length > 0) setErrors(fields)
      else setFormError(getErrorMessage(error))
    }
  }

  return (
    <AuthCard title="Регистрация">
      <Stack component="form" spacing={2} onSubmit={onSubmit} noValidate>
        {formError && <Alert severity="error">{formError}</Alert>}
        <TextField label="Имя" autoComplete="name" required autoFocus {...field('name')} />
        <TextField label="Email" type="email" autoComplete="email" required {...field('email')} />
        <TextField
          label="Пароль"
          type="password"
          autoComplete="new-password"
          required
          {...field('password')}
        />
        <Button type="submit" variant="contained" disabled={isLoading}>
          Зарегистрироваться
        </Button>
        <Link component={RouterLink} to="/login" variant="body2">
          Уже есть аккаунт? Войти
        </Link>
      </Stack>
    </AuthCard>
  )
}

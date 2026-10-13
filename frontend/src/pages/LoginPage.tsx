import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link as RouterLink, Navigate, useLocation } from 'react-router'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Link from '@mui/material/Link'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import { useAppSelector } from '../app/hooks'
import { useLoginMutation } from '../features/auth/authApi'
import { selectAuthStatus } from '../features/auth/authSlice'
import { getErrorCode, getErrorMessage } from '../shared/api/errors'
import { AuthCard } from '../shared/ui/AuthCard'

interface LoginLocationState {
  from?: string
  registered?: boolean
}

export function LoginPage() {
  const status = useAppSelector(selectAuthStatus)
  const state = useLocation().state as LoginLocationState | null
  const [login, { isLoading, error }] = useLoginMutation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  if (status === 'authenticated') {
    return <Navigate to={state?.from ?? '/'} replace />
  }

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    void login({ email: email.trim(), password })
  }

  return (
    <AuthCard title="Вход">
      <Stack component="form" spacing={2} onSubmit={onSubmit} noValidate>
        {state?.registered && !error && (
          <Alert severity="success">Регистрация завершена. Войдите в систему.</Alert>
        )}
        {error && (
          <Alert severity="error">
            {getErrorCode(error) === 'UNAUTHORIZED'
              ? 'Неверный email или пароль'
              : getErrorMessage(error)}
          </Alert>
        )}
        <TextField
          label="Email"
          type="email"
          autoComplete="email"
          required
          autoFocus
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <TextField
          label="Пароль"
          type="password"
          autoComplete="current-password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <Button type="submit" variant="contained" disabled={isLoading || !email || !password}>
          Войти
        </Button>
        <Link component={RouterLink} to="/register" variant="body2">
          Нет аккаунта? Зарегистрироваться
        </Link>
      </Stack>
    </AuthCard>
  )
}

import { useEffect } from 'react'
import type { ReactNode } from 'react'
import Box from '@mui/material/Box'
import CircularProgress from '@mui/material/CircularProgress'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { CSRF_COOKIE } from '../../shared/api/baseApi'
import { getCookie } from '../../shared/lib/cookies'
import { useRestoreSessionMutation } from './authApi'
import { loggedOut, selectAuthStatus } from './authSlice'

/** Пытается восстановить сессию по refresh-cookie до первого показа приложения. */
export function SessionGate({ children }: { children: ReactNode }) {
  const status = useAppSelector(selectAuthStatus)
  const dispatch = useAppDispatch()
  const [restoreSession] = useRestoreSessionMutation()

  useEffect(() => {
    if (status !== 'unknown') return
    if (getCookie(CSRF_COOKIE) === null) {
      dispatch(loggedOut())
      return
    }
    const request = restoreSession()
    return () => request.abort()
  }, [status, dispatch, restoreSession])

  if (status === 'unknown') {
    return (
      <Box sx={{ display: 'grid', placeItems: 'center', minHeight: '100vh' }}>
        <CircularProgress aria-label="Загрузка" />
      </Box>
    )
  }
  return children
}

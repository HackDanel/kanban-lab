import { Navigate, Outlet, useLocation } from 'react-router'
import { useAppSelector } from '../../app/hooks'
import { selectAuthStatus } from './authSlice'

export function RequireAuth() {
  const status = useAppSelector(selectAuthStatus)
  const location = useLocation()

  if (status !== 'authenticated') {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  return <Outlet />
}

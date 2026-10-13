import { Navigate, Route, Routes } from 'react-router'
import { RequireAuth } from '../features/auth/RequireAuth'
import { SessionGate } from '../features/auth/SessionGate'
import { LoginPage } from '../pages/LoginPage'
import { ProjectsPage } from '../pages/ProjectsPage'
import { RegisterPage } from '../pages/RegisterPage'
import { AppLayout } from './AppLayout'

export default function App() {
  return (
    <SessionGate>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route element={<RequireAuth />}>
          <Route element={<AppLayout />}>
            <Route index element={<ProjectsPage />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </SessionGate>
  )
}

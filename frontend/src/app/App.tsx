import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router'
import CircularProgress from '@mui/material/CircularProgress'
import { RequireAuth } from '../features/auth/RequireAuth'
import { SessionGate } from '../features/auth/SessionGate'
import { LoginPage } from '../pages/LoginPage'
import { ProjectsPage } from '../pages/ProjectsPage'
import { RegisterPage } from '../pages/RegisterPage'
import { AppLayout } from './AppLayout'

// Доска с drag-and-drop и страницы проекта загружаются отдельными чанками.
const BoardPage = lazy(() => import('../pages/BoardPage').then((m) => ({ default: m.BoardPage })))
const ProjectPage = lazy(() =>
  import('../pages/ProjectPage').then((m) => ({ default: m.ProjectPage })),
)
const AdminUsersPage = lazy(() =>
  import('../pages/AdminUsersPage').then((m) => ({ default: m.AdminUsersPage })),
)

export default function App() {
  return (
    <SessionGate>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route element={<RequireAuth />}>
          <Route
            element={
              <Suspense fallback={<CircularProgress aria-label="Загрузка" />}>
                <AppLayout />
              </Suspense>
            }
          >
            <Route index element={<ProjectsPage />} />
            <Route path="projects/:projectId" element={<ProjectPage />} />
            <Route path="boards/:boardId" element={<BoardPage />} />
            <Route path="admin/users" element={<AdminUsersPage />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </SessionGate>
  )
}

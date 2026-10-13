import { Link as RouterLink, Outlet } from 'react-router'
import AppBar from '@mui/material/AppBar'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Container from '@mui/material/Container'
import Link from '@mui/material/Link'
import Toolbar from '@mui/material/Toolbar'
import Typography from '@mui/material/Typography'
import { useLogoutMutation } from '../features/auth/authApi'
import { selectCurrentUser } from '../features/auth/authSlice'
import { ROLE_LABELS } from '../shared/lib/permissions'
import { useAppSelector } from './hooks'

export function AppLayout() {
  const user = useAppSelector(selectCurrentUser)
  const [logout, { isLoading }] = useLogoutMutation()

  return (
    <>
      <AppBar position="sticky" elevation={0}>
        <Toolbar sx={{ gap: 2 }}>
          <Link
            component={RouterLink}
            to="/"
            color="inherit"
            underline="none"
            variant="h6"
            sx={{ flexGrow: 1 }}
          >
            Kanban
          </Link>
          {user && (
            <>
              <Typography variant="body2" sx={{ display: { xs: 'none', sm: 'block' } }}>
                {user.name}
              </Typography>
              <Chip
                size="small"
                label={ROLE_LABELS[user.role]}
                sx={{ color: 'inherit', bgcolor: 'rgba(255,255,255,0.2)' }}
              />
            </>
          )}
          <Button color="inherit" disabled={isLoading} onClick={() => void logout()}>
            Выйти
          </Button>
        </Toolbar>
      </AppBar>
      <Container component="main" maxWidth="lg" sx={{ py: 3 }}>
        <Outlet />
      </Container>
    </>
  )
}

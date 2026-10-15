import { useState } from 'react'
import { Navigate } from 'react-router'
import Alert from '@mui/material/Alert'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useAppSelector } from '../app/hooks'
import { selectCurrentUser } from '../features/auth/authSlice'
import { useListAllUsersQuery, useSetUserRoleMutation } from '../features/projects/projectApi'
import { getErrorMessage } from '../shared/api/errors'
import type { Role } from '../shared/api/types'
import { isAdmin, ROLE_LABELS } from '../shared/lib/permissions'

export function AdminUsersPage() {
  const me = useAppSelector(selectCurrentUser)
  const [query, setQuery] = useState('')
  const { data, error } = useListAllUsersQuery({ q: query, offset: 0 }, { skip: !isAdmin(me) })
  const [setRole, roleChange] = useSetUserRoleMutation()

  if (!isAdmin(me)) return <Navigate to="/" replace />

  return (
    <>
      <Typography component="h1" variant="h4" sx={{ mb: 2 }}>
        Пользователи
      </Typography>
      <TextField
        size="small"
        type="search"
        label="Поиск"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        sx={{ mb: 2 }}
      />
      {(error || roleChange.error) && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {getErrorMessage(error ?? roleChange.error)}
        </Alert>
      )}
      <Paper variant="outlined" sx={{ overflowX: 'auto' }}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Имя</TableCell>
              <TableCell>Email</TableCell>
              <TableCell>Роль</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {data?.items.map((u) => (
              <TableRow key={u.id}>
                <TableCell>{u.name}</TableCell>
                <TableCell>{u.email}</TableCell>
                <TableCell>
                  <TextField
                    select
                    size="small"
                    value={u.role}
                    disabled={u.id === me?.id}
                    onChange={(e) => void setRole({ userId: u.id, role: e.target.value as Role })}
                    slotProps={{ htmlInput: { 'aria-label': `Роль ${u.name}` } }}
                  >
                    {(Object.keys(ROLE_LABELS) as Role[]).map((r) => (
                      <MenuItem key={r} value={r}>
                        {ROLE_LABELS[r]}
                      </MenuItem>
                    ))}
                  </TextField>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Paper>
      {data && data.total > data.items.length && (
        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
          Показано {data.items.length} из {data.total}. Уточните поиск.
        </Typography>
      )}
    </>
  )
}

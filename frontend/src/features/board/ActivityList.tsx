import Alert from '@mui/material/Alert'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemText from '@mui/material/ListItemText'
import Typography from '@mui/material/Typography'
import { getErrorMessage } from '../../shared/api/errors'
import type { Column, User } from '../../shared/api/types'
import { useListActivityQuery } from './boardApi'
import { describeActivity, formatDateTime } from './format'

export function ActivityList({
  cardId,
  members,
}: {
  cardId: string
  members: User[]
  columns: Column[]
}) {
  const { data: entries = [], error } = useListActivityQuery(cardId)
  const byId = new Map(members.map((m) => [m.id, m]))

  if (error) return <Alert severity="error">{getErrorMessage(error)}</Alert>
  if (entries.length === 0) return <Typography color="text.secondary">История пуста.</Typography>
  return (
    <List dense>
      {entries.map((entry) => (
        <ListItem key={entry.id} disableGutters data-testid="activity">
          <ListItemText
            primary={`${byId.get(entry.actorId)?.name ?? 'Бывший участник'} ${describeActivity(entry, byId)}`}
            secondary={formatDateTime(entry.createdAt)}
          />
        </ListItem>
      ))}
    </List>
  )
}

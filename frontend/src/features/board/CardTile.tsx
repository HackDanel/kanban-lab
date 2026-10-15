import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import Avatar from '@mui/material/Avatar'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import ChatBubbleOutline from '@mui/icons-material/ChatBubbleOutlined'
import AttachFile from '@mui/icons-material/AttachFile'
import type { Card, Label, User } from '../../shared/api/types'
import { initials, isOverdue, PRIORITY_COLORS, PRIORITY_LABELS } from './format'
import { LabelChip } from './labels'

interface CardTileProps {
  card: Card
  projectKey: string
  labels: Map<string, Label>
  members: Map<string, User>
  onOpen: (card: Card) => void
}

export function CardTile({ card, projectKey, labels, members, onOpen }: CardTileProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: card.id,
  })
  const assignee = card.assigneeId ? members.get(card.assigneeId) : undefined

  return (
    <Paper
      ref={setNodeRef}
      variant="outlined"
      data-testid="card"
      data-card-id={card.id}
      {...attributes}
      {...listeners}
      aria-label={`Карточка ${projectKey}-${card.number}: ${card.title}`}
      onClick={() => onOpen(card)}
      onKeyDown={(event) => {
        listeners?.onKeyDown?.(event)
        if (event.key === 'Enter' && !event.defaultPrevented) onOpen(card)
      }}
      sx={{
        p: 1.5,
        cursor: 'grab',
        opacity: isDragging ? 0.5 : 1,
        transform: CSS.Transform.toString(transform),
        transition,
        '&:hover': { borderColor: 'primary.main' },
      }}
    >
      <Typography variant="body2" sx={{ fontWeight: 500, wordBreak: 'break-word' }}>
        {card.title}
      </Typography>
      {card.labelIds.length > 0 && (
        <Stack direction="row" sx={{ mt: 1, gap: 0.5, flexWrap: 'wrap' }}>
          {card.labelIds.map((id) => {
            const label = labels.get(id)
            return label ? <LabelChip key={id} label={label} /> : null
          })}
        </Stack>
      )}
      <Stack direction="row" sx={{ mt: 1, alignItems: 'center', gap: 1 }}>
        <Typography variant="caption" color="text.secondary">
          {projectKey}-{card.number}
        </Typography>
        <Chip
          size="small"
          variant="outlined"
          color={PRIORITY_COLORS[card.priority]}
          label={PRIORITY_LABELS[card.priority]}
        />
        {card.dueDate && (
          <Typography
            variant="caption"
            color={isOverdue(card.dueDate) ? 'error' : 'text.secondary'}
            title="Дедлайн"
          >
            {card.dueDate}
          </Typography>
        )}
        <Box sx={{ flexGrow: 1 }} />
        {card.commentCount > 0 && (
          <Stack direction="row" sx={{ alignItems: 'center' }} title="Комментарии">
            <ChatBubbleOutline sx={{ fontSize: 14 }} />
            <Typography variant="caption">{card.commentCount}</Typography>
          </Stack>
        )}
        {card.attachmentCount > 0 && (
          <Stack direction="row" sx={{ alignItems: 'center' }} title="Вложения">
            <AttachFile sx={{ fontSize: 14 }} />
            <Typography variant="caption">{card.attachmentCount}</Typography>
          </Stack>
        )}
        {assignee && (
          <Avatar sx={{ width: 24, height: 24, fontSize: 12 }} title={assignee.name}>
            {initials(assignee.name)}
          </Avatar>
        )}
      </Stack>
    </Paper>
  )
}

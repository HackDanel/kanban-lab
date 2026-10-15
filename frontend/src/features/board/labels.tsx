import Chip from '@mui/material/Chip'
import type { Label } from '../../shared/api/types'
import { contrastText } from './format'

export function LabelChip({ label, onDelete }: { label: Label; onDelete?: () => void }) {
  return (
    <Chip
      size="small"
      label={label.name}
      onDelete={onDelete}
      sx={{ bgcolor: label.color, color: contrastText(label.color) }}
    />
  )
}

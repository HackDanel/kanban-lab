import { useRef, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import IconButton from '@mui/material/IconButton'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemText from '@mui/material/ListItemText'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import DeleteOutlined from '@mui/icons-material/DeleteOutlined'
import FileDownloadOutlined from '@mui/icons-material/FileDownloadOutlined'
import { useAppSelector } from '../../app/hooks'
import { getErrorMessage } from '../../shared/api/errors'
import type { Attachment, Card } from '../../shared/api/types'
import { canManage } from '../../shared/lib/permissions'
import { selectCurrentUser } from '../auth/authSlice'
import {
  useDeleteAttachmentMutation,
  useDownloadAttachmentMutation,
  useListAttachmentsQuery,
  useUploadAttachmentMutation,
} from './boardApi'
import { formatSize, MAX_UPLOAD_BYTES } from './format'

/** Сохраняет blob как файл: ссылку на файл нельзя открыть напрямую, нужен заголовок Authorization. */
function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

export function AttachmentsPanel({ card }: { card: Card }) {
  const me = useAppSelector(selectCurrentUser)
  const { data: attachments = [] } = useListAttachmentsQuery(card.id)
  const [upload, uploading] = useUploadAttachmentMutation()
  const [remove] = useDeleteAttachmentMutation()
  const [download] = useDownloadAttachmentMutation()
  const [error, setError] = useState<string | null>(null)
  const input = useRef<HTMLInputElement>(null)

  const onFile = async (file: File | undefined) => {
    if (!file) return
    setError(null)
    if (file.size > MAX_UPLOAD_BYTES) {
      setError('Файл больше 10 МБ')
      return
    }
    try {
      await upload({ card, file }).unwrap()
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      if (input.current) input.current.value = ''
    }
  }

  const onDownload = async (a: Attachment) => {
    try {
      saveBlob(await download(a.id).unwrap(), a.filename)
    } catch (err) {
      setError(getErrorMessage(err))
    }
  }

  const onDelete = async (a: Attachment) => {
    try {
      await remove({ card, attachmentId: a.id }).unwrap()
    } catch (err) {
      setError(getErrorMessage(err))
    }
  }

  return (
    <Stack spacing={2}>
      {error && <Alert severity="error">{error}</Alert>}
      {attachments.length === 0 ? (
        <Typography color="text.secondary">Вложений пока нет.</Typography>
      ) : (
        <List dense>
          {attachments.map((a) => (
            <ListItem
              key={a.id}
              disableGutters
              secondaryAction={
                <>
                  <IconButton
                    aria-label={`Скачать ${a.filename}`}
                    onClick={() => void onDownload(a)}
                  >
                    <FileDownloadOutlined />
                  </IconButton>
                  {(a.uploaderId === me?.id || canManage(me)) && (
                    <IconButton
                      aria-label={`Удалить ${a.filename}`}
                      onClick={() => void onDelete(a)}
                    >
                      <DeleteOutlined />
                    </IconButton>
                  )}
                </>
              }
            >
              <ListItemText primary={a.filename} secondary={formatSize(a.size)} />
            </ListItem>
          ))}
        </List>
      )}
      <input
        ref={input}
        type="file"
        hidden
        aria-label="Файл вложения"
        onChange={(e) => void onFile(e.target.files?.[0])}
      />
      <Button
        variant="outlined"
        disabled={uploading.isLoading}
        onClick={() => input.current?.click()}
        sx={{ alignSelf: 'flex-start' }}
      >
        {uploading.isLoading ? 'Загрузка…' : 'Прикрепить файл'}
      </Button>
      <Typography variant="caption" color="text.secondary">
        До 10 МБ: изображения, PDF, текст, CSV, JSON, ZIP, документы Office.
      </Typography>
    </Stack>
  )
}

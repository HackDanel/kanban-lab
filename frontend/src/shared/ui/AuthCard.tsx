import type { ReactNode } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Typography from '@mui/material/Typography'

export function AuthCard({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Box sx={{ display: 'grid', placeItems: 'center', minHeight: '100vh', p: 2 }}>
      <Paper component="main" variant="outlined" sx={{ p: 4, width: '100%', maxWidth: 400 }}>
        <Typography component="h1" variant="h5" sx={{ mb: 3 }}>
          {title}
        </Typography>
        {children}
      </Paper>
    </Box>
  )
}

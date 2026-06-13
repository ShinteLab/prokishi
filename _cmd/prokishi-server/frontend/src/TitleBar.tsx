import { useState, useEffect } from 'react'
import { Box, IconButton } from '@mui/material'
import RemoveIcon from '@mui/icons-material/Remove'
import FilterNoneIcon from '@mui/icons-material/FilterNone'
import CropSquareIcon from '@mui/icons-material/CropSquare'
import CloseIcon from '@mui/icons-material/Close'
import { Window } from '@wailsio/runtime'

// CSS custom property を React の style に渡す型拡張
const draggable = { '--wails-draggable': 'drag' } as React.CSSProperties

const btnBase = {
  borderRadius: 0,
  px: 1.5,
  py: 0,
  height: 36,
  minWidth: 0,
  color: 'text.secondary',
  '&:hover': { color: 'text.primary' },
} as const

export function TitleBar({ title, onClose }: { title: string; onClose: () => void }) {
  const [maximised, setMaximised] = useState(false)

  useEffect(() => {
    Window.IsMaximised().then(setMaximised)
  }, [])

  const handleMinimise = () => Window.Minimise()

  const handleToggleMaximise = () => {
    Window.ToggleMaximise().then(() =>
      Window.IsMaximised().then(setMaximised)
    )
  }

  const handleClose = () => onClose()

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        width: '100%',
        height: 36,
        bgcolor: 'background.paper',
        borderBottom: '1px solid',
        borderColor: 'divider',
        flexShrink: 0,
        userSelect: 'none',
        boxSizing: 'border-box',
      }}
    >
      {/* ドラッグ可能なタイトル領域 */}
      <Box
        style={draggable}
        onDoubleClick={handleToggleMaximise}
        sx={{
          flex: 1,
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          px: 1.5,
          cursor: 'default',
          fontSize: '0.8125rem',
          color: 'text.secondary',
          letterSpacing: 0.3,
        }}
      >
        {title}
      </Box>

      {/* ウィンドウコントロールボタン */}
      <Box sx={{ display: 'flex', height: '100%' }}>
        <IconButton
          size="small"
          disableRipple
          onClick={handleMinimise}
          sx={{ ...btnBase, '&:hover': { bgcolor: 'action.hover', color: 'text.primary' } }}
        >
          <RemoveIcon sx={{ fontSize: 16 }} />
        </IconButton>

        <IconButton
          size="small"
          disableRipple
          onClick={handleToggleMaximise}
          sx={{ ...btnBase, '&:hover': { bgcolor: 'action.hover', color: 'text.primary' } }}
        >
          {maximised
            ? <FilterNoneIcon sx={{ fontSize: 14 }} />
            : <CropSquareIcon sx={{ fontSize: 16 }} />
          }
        </IconButton>

        <IconButton
          size="small"
          disableRipple
          onClick={handleClose}
          sx={{
            ...btnBase,
            '&:hover': { bgcolor: '#c42b1c', color: '#fff' },
          }}
        >
          <CloseIcon sx={{ fontSize: 16 }} />
        </IconButton>
      </Box>
    </Box>
  )
}

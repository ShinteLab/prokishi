import { useState, useEffect } from 'react'
import { Box, IconButton, Menu, MenuItem } from '@mui/material'
import MenuIcon from '@mui/icons-material/Menu'
import RemoveIcon from '@mui/icons-material/Remove'
import FilterNoneIcon from '@mui/icons-material/FilterNone'
import CropSquareIcon from '@mui/icons-material/CropSquare'
import CloseIcon from '@mui/icons-material/Close'
import { Window } from '@wailsio/runtime'

const draggable = { '--wails-draggable': 'drag' } as React.CSSProperties

const btnBase = {
  borderRadius: 0,
  px: 1.5,
  py: 0,
  height: 36,
  minWidth: 0,
  color: 'text.secondary',
} as const

export type View = 'monitor' | 'master' | 'system' | 'board'

interface TitleBarProps {
  title: string
  view: View
  onNavigate: (v: View) => void
  onClose: () => void
}

export function TitleBar({ title, view, onNavigate, onClose }: TitleBarProps) {
  const [maximised, setMaximised] = useState(false)
  const [menuAnchor, setMenuAnchor] = useState<null | HTMLElement>(null)

  useEffect(() => {
    Window.IsMaximised().then(setMaximised)
  }, [])

  const handleMinimise = () => Window.Minimise()

  const handleToggleMaximise = () => {
    Window.ToggleMaximise().then(() => Window.IsMaximised().then(setMaximised))
  }

  const handleMenuOpen = (e: React.MouseEvent<HTMLElement>) => setMenuAnchor(e.currentTarget)
  const handleMenuClose = () => setMenuAnchor(null)

  const navigate = (v: View) => {
    onNavigate(v)
    handleMenuClose()
  }

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
      {/* ハンバーガーメニュー */}
      <IconButton
        size="small"
        disableRipple
        onClick={handleMenuOpen}
        sx={{ ...btnBase, '&:hover': { bgcolor: 'action.hover', color: 'text.primary' } }}
      >
        <MenuIcon sx={{ fontSize: 18 }} />
      </IconButton>

      <Menu anchorEl={menuAnchor} open={!!menuAnchor} onClose={handleMenuClose}>
        <MenuItem selected={view === 'monitor'} onClick={() => navigate('monitor')}>
          モニター
        </MenuItem>
        <MenuItem selected={view === 'board'} onClick={() => navigate('board')}>
          盤面
        </MenuItem>
        <MenuItem selected={view === 'master'} onClick={() => navigate('master')}>
          マスタ管理
        </MenuItem>
        <MenuItem selected={view === 'system'} onClick={() => navigate('system')}>
          システム監視
        </MenuItem>
      </Menu>

      {/* ドラッグ可能なタイトル領域 */}
      <Box
        style={draggable}
        onDoubleClick={handleToggleMaximise}
        sx={{
          flex: 1,
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          px: 1,
          cursor: 'default',
          fontSize: '0.8125rem',
          color: 'text.secondary',
          letterSpacing: 0.3,
        }}
      >
        {title}
      </Box>

      {/* ウィンドウコントロール */}
      <Box sx={{ display: 'flex', height: '100%' }}>
        <IconButton size="small" disableRipple onClick={handleMinimise}
          sx={{ ...btnBase, '&:hover': { bgcolor: 'action.hover', color: 'text.primary' } }}>
          <RemoveIcon sx={{ fontSize: 16 }} />
        </IconButton>

        <IconButton size="small" disableRipple onClick={handleToggleMaximise}
          sx={{ ...btnBase, '&:hover': { bgcolor: 'action.hover', color: 'text.primary' } }}>
          {maximised
            ? <FilterNoneIcon sx={{ fontSize: 14 }} />
            : <CropSquareIcon sx={{ fontSize: 16 }} />
          }
        </IconButton>

        <IconButton size="small" disableRipple onClick={onClose}
          sx={{ ...btnBase, '&:hover': { bgcolor: '#c42b1c', color: '#fff' } }}>
          <CloseIcon sx={{ fontSize: 16 }} />
        </IconButton>
      </Box>
    </Box>
  )
}

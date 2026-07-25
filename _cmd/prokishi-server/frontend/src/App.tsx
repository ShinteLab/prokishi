import { useState, useEffect, useCallback } from 'react'
import {
  Box, Button, Dialog, DialogActions, DialogContent,
  DialogContentText, DialogTitle,
} from '@mui/material'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import { Events } from '@wailsio/runtime'
import { WindowService } from '../bindings/prokishi-server'
import '@shinte/web' // <shogi-board> を customElements に登録(副作用 import)
import { TitleBar, View } from './TitleBar'
import { MonitorView } from './MonitorView'
import { MasterView } from './MasterView'
import { SystemView } from './SystemView'
import { BoardView } from './BoardView'

const darkTheme = createTheme({
  palette: { mode: 'dark' },
})


export default function App() {
  const [view, setView] = useState<View>('monitor')
  const [closeDialogOpen, setCloseDialogOpen] = useState(false)

  const requestClose = useCallback(() => setCloseDialogOpen(true), [])

  useEffect(() => {
    const unsub = Events.On('request-close', requestClose)
    return () => { unsub() }
  }, [requestClose])

  const handleConfirmClose = () => {
    setCloseDialogOpen(false)
    WindowService.ShutdownAndQuit()
  }

  return (
    <ThemeProvider theme={darkTheme}>
      <CssBaseline />

      <Box sx={{
        position: 'fixed',
        top: '5px', left: '5px', right: '5px', bottom: '5px',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        borderRadius: '4px',
      }}>
        <TitleBar
          title="prokishi-server"
          view={view}
          onNavigate={setView}
          onClose={requestClose}
        />

        {view === 'monitor' && <MonitorView />}
        {view === 'master' && <MasterView />}
        {view === 'system' && <SystemView />}
        {view === 'board' && <BoardView />}
      </Box>

      {/* 終了確認ダイアログ */}
      <Dialog open={closeDialogOpen} onClose={() => setCloseDialogOpen(false)}>
        <DialogTitle>終了しますか？</DialogTitle>
        <DialogContent>
          <DialogContentText>
            サーバを停止してアプリケーションを終了します。
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCloseDialogOpen(false)}>キャンセル</Button>
          <Button onClick={handleConfirmClose} color="error" variant="contained" autoFocus>
            終了
          </Button>
        </DialogActions>
      </Dialog>
    </ThemeProvider>
  )
}

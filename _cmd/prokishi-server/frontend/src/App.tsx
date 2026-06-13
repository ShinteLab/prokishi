import { useState, useEffect, useCallback } from 'react'
import {
  Box, Button, Container, Dialog, DialogActions, DialogContent,
  DialogContentText, DialogTitle, IconButton, Paper, Snackbar, Alert,
  Tab, Tabs, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, TextField, Tooltip, CircularProgress,
} from '@mui/material'
import {
  Add as AddIcon,
  AutoFixHigh as GenerateIcon,
  Delete as DeleteIcon,
  FolderOpen as FolderOpenIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import { Events } from '@wailsio/runtime'
import { AdminService, WindowService } from '../bindings/wails'
import { TitleBar } from './TitleBar'

const darkTheme = createTheme({ palette: { mode: 'dark' } })

type EngineItem = { id: string; path: string; created: string }
type CodeItem = { code: string; created: string; used: string }
type Severity = 'success' | 'error'

export default function App() {
  const [tab, setTab] = useState(0)
  const [engines, setEngines] = useState<EngineItem[]>([])
  const [codes, setCodes] = useState<CodeItem[]>([])
  const [enginePath, setEnginePath] = useState('')
  const [codeInput, setCodeInput] = useState('')
  const [loadingEngines, setLoadingEngines] = useState(false)
  const [loadingCodes, setLoadingCodes] = useState(false)
  const [snack, setSnack] = useState<{ msg: string; sev: Severity } | null>(null)
  const [closeDialogOpen, setCloseDialogOpen] = useState(false)

  const notify = (msg: string, sev: Severity = 'success') => setSnack({ msg, sev })

  // X ボタン / OS のクローズ要求（Alt+F4 等）で確認ダイアログを開く
  const requestClose = useCallback(() => setCloseDialogOpen(true), [])

  // Go 側から "request-close" イベントが来たときも同じダイアログを開く
  useEffect(() => {
    const unsub = Events.On('request-close', requestClose)
    return () => { unsub() }
  }, [requestClose])

  const handleConfirmClose = () => {
    setCloseDialogOpen(false)
    WindowService.ShutdownAndQuit()
  }

  const handleCancelClose = () => setCloseDialogOpen(false)

  const loadEngines = useCallback(() => {
    setLoadingEngines(true)
    AdminService.ListEngines()
      .then((items: any) => setEngines(items ?? []))
      .catch((e: any) => notify(String(e), 'error'))
      .finally(() => setLoadingEngines(false))
  }, [])

  const loadCodes = useCallback(() => {
    setLoadingCodes(true)
    AdminService.ListCodes()
      .then((items: any) => setCodes(items ?? []))
      .catch((e: any) => notify(String(e), 'error'))
      .finally(() => setLoadingCodes(false))
  }, [])

  useEffect(() => { loadEngines(); loadCodes() }, [loadEngines, loadCodes])

  const handleSelectFile = () => {
    AdminService.SelectEnginePath()
      .then((path: any) => { if (path) setEnginePath(path) })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleRegisterEngine = () => {
    const path = enginePath.trim()
    if (!path) return
    AdminService.RegisterEngine(path)
      .then((id: any) => {
        notify(`登録しました: ${id}`)
        setEnginePath('')
        loadEngines()
      })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleDeleteEngine = (id: string) => {
    AdminService.DeleteEngine(id)
      .then(() => { notify('削除しました'); loadEngines() })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleGenerateCode = () => {
    AdminService.GenerateCode()
      .then((code: any) => { notify(`生成しました: ${code}`); loadCodes() })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleRegisterCode = () => {
    const code = codeInput.trim()
    if (!code) return
    AdminService.RegisterCode(code)
      .then(() => { notify('登録しました'); setCodeInput(''); loadCodes() })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleDeleteCode = (code: string) => {
    AdminService.DeleteCode(code)
      .then(() => { notify('削除しました'); loadCodes() })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const monoCell = {
    fontFamily: 'monospace',
    fontSize: '0.75rem',
    color: 'primary.light',
  } as const

  const subCell = { color: 'text.secondary', fontSize: '0.75rem' } as const

  return (
    <ThemeProvider theme={darkTheme}>
      <CssBaseline />

      {/*
        position: fixed + inset: 0 でビューポートを確実に埋める。
        height: 100vh はスクロールバー幅でずれる場合があるため使わない。
      */}
      <Box sx={{
        position: 'fixed',
        top: '5px', left: '5px', right: '5px', bottom: '5px',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        borderRadius: '4px',
      }}>
        <TitleBar title="prokishi-server" onClose={requestClose} />

        <Box sx={{ flex: 1, overflow: 'auto' }}>
          <Container maxWidth="md" sx={{ py: 3 }}>
            <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2 }}>
              <Tab label="エンジン管理" />
              <Tab label="認証コード管理" />
            </Tabs>

            {tab === 0 && (
              <Box>
                <Box sx={{ display: 'flex', gap: 1, mb: 2, alignItems: 'center' }}>
                  <TextField
                    size="small"
                    fullWidth
                    label="エンジンのパス"
                    value={enginePath}
                    onChange={e => setEnginePath(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && handleRegisterEngine()}
                    placeholder="C:\engines\apery.exe"
                  />
                  <Tooltip title="ファイルを選択">
                    <IconButton onClick={handleSelectFile} color="primary">
                      <FolderOpenIcon />
                    </IconButton>
                  </Tooltip>
                  <Button
                    variant="contained"
                    startIcon={<AddIcon />}
                    onClick={handleRegisterEngine}
                    disabled={!enginePath.trim()}
                    sx={{ whiteSpace: 'nowrap' }}
                  >
                    登録
                  </Button>
                  <Tooltip title="更新">
                    <IconButton onClick={loadEngines}>
                      <RefreshIcon />
                    </IconButton>
                  </Tooltip>
                </Box>

                <TableContainer component={Paper} variant="outlined">
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>ID</TableCell>
                        <TableCell>パス</TableCell>
                        <TableCell>登録日時</TableCell>
                        <TableCell sx={{ width: 60 }} />
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {loadingEngines ? (
                        <TableRow>
                          <TableCell colSpan={4} align="center" sx={{ py: 3 }}>
                            <CircularProgress size={24} />
                          </TableCell>
                        </TableRow>
                      ) : engines.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={4} align="center" sx={{ py: 3, color: 'text.disabled' }}>
                            登録済みエンジンがありません
                          </TableCell>
                        </TableRow>
                      ) : engines.map(e => (
                        <TableRow key={e.id} hover>
                          <TableCell sx={monoCell}>{e.id}</TableCell>
                          <TableCell>{e.path}</TableCell>
                          <TableCell sx={subCell}>{e.created}</TableCell>
                          <TableCell>
                            <Tooltip title="削除">
                              <IconButton size="small" color="error" onClick={() => handleDeleteEngine(e.id)}>
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Box>
            )}

            {tab === 1 && (
              <Box>
                <Box sx={{ display: 'flex', gap: 1, mb: 2, alignItems: 'center' }}>
                  <TextField
                    size="small"
                    fullWidth
                    label="認証コード"
                    value={codeInput}
                    onChange={e => setCodeInput(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && handleRegisterCode()}
                    placeholder="任意のコード文字列"
                  />
                  <Button
                    variant="contained"
                    startIcon={<AddIcon />}
                    onClick={handleRegisterCode}
                    disabled={!codeInput.trim()}
                    sx={{ whiteSpace: 'nowrap' }}
                  >
                    登録
                  </Button>
                  <Button
                    variant="outlined"
                    startIcon={<GenerateIcon />}
                    onClick={handleGenerateCode}
                    sx={{ whiteSpace: 'nowrap' }}
                  >
                    自動生成
                  </Button>
                  <Tooltip title="更新">
                    <IconButton onClick={loadCodes}>
                      <RefreshIcon />
                    </IconButton>
                  </Tooltip>
                </Box>

                <TableContainer component={Paper} variant="outlined">
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>コード</TableCell>
                        <TableCell>登録日時</TableCell>
                        <TableCell>最終使用</TableCell>
                        <TableCell sx={{ width: 60 }} />
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {loadingCodes ? (
                        <TableRow>
                          <TableCell colSpan={4} align="center" sx={{ py: 3 }}>
                            <CircularProgress size={24} />
                          </TableCell>
                        </TableRow>
                      ) : codes.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={4} align="center" sx={{ py: 3, color: 'text.disabled' }}>
                            登録済みコードがありません
                          </TableCell>
                        </TableRow>
                      ) : codes.map(c => (
                        <TableRow key={c.code} hover>
                          <TableCell sx={monoCell}>{c.code}</TableCell>
                          <TableCell sx={subCell}>{c.created}</TableCell>
                          <TableCell sx={subCell}>{c.used || '—'}</TableCell>
                          <TableCell>
                            <Tooltip title="削除">
                              <IconButton size="small" color="error" onClick={() => handleDeleteCode(c.code)}>
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Box>
            )}
          </Container>
        </Box>
      </Box>

      {/* 終了確認ダイアログ */}
      <Dialog open={closeDialogOpen} onClose={handleCancelClose}>
        <DialogTitle>終了しますか？</DialogTitle>
        <DialogContent>
          <DialogContentText>
            サーバを停止してアプリケーションを終了します。
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCancelClose}>キャンセル</Button>
          <Button onClick={handleConfirmClose} color="error" variant="contained" autoFocus>
            終了
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar は fixed レイアウトの外に置く（MUI 自身が position:fixed で表示するため問題なし） */}
      <Snackbar
        open={!!snack}
        autoHideDuration={4000}
        onClose={() => setSnack(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity={snack?.sev} onClose={() => setSnack(null)} variant="filled">
          {snack?.msg}
        </Alert>
      </Snackbar>
    </ThemeProvider>
  )
}

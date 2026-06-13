import { useState, useEffect, useCallback } from 'react'
import {
  Box, Button, Chip, Container, FormControlLabel, IconButton, Paper,
  Snackbar, Alert, Switch, Tab, Tabs, Table, TableBody, TableCell,
  TableContainer, TableHead, TableRow, TextField, Tooltip, CircularProgress,
  Typography,
} from '@mui/material'
import {
  Add as AddIcon,
  AutoFixHigh as GenerateIcon,
  ContentCopy as CopyIcon,
  Delete as DeleteIcon,
  FolderOpen as FolderOpenIcon,
  PlayArrow as StartIcon,
  Refresh as RefreshIcon,
  Stop as StopIcon,
} from '@mui/icons-material'
import { Events } from '@wailsio/runtime'
import { AdminService, ServerService } from '../bindings/wails'

type EngineItem = { id: string; path: string; created: string }
type CodeItem = { code: string; created: string; used: string }
type ServerConfig = { host: string; port: number; autoStart: boolean }
type ServerState = { running: boolean; url: string }
type Severity = 'success' | 'error'

export function MasterView() {
  const [tab, setTab] = useState(0)
  // エンジン
  const [engines, setEngines] = useState<EngineItem[]>([])
  const [codes, setCodes] = useState<CodeItem[]>([])
  const [enginePath, setEnginePath] = useState('')
  const [codeInput, setCodeInput] = useState('')
  const [loadingEngines, setLoadingEngines] = useState(false)
  const [loadingCodes, setLoadingCodes] = useState(false)
  // サーバ設定
  const [serverState, setServerState] = useState<ServerState>({ running: false, url: '' })
  const [cfg, setCfg] = useState<ServerConfig>({ host: '', port: 8080, autoStart: true })
  const [cfgDirty, setCfgDirty] = useState(false)
  const [savingCfg, setSavingCfg] = useState(false)
  // 共通
  const [snack, setSnack] = useState<{ msg: string; sev: Severity } | null>(null)

  const notify = (msg: string, sev: Severity = 'success') => setSnack({ msg, sev })

  // --- サーバ設定 ---
  useEffect(() => {
    ServerService.GetState().then((s: any) => setServerState(s)).catch(() => {})
    ServerService.GetConfig().then((c: any) => { if (c) setCfg(c) }).catch(() => {})

    const unsub = Events.On('server-state', (e: any) => {
      const s: ServerState = e.data
      if (s) setServerState(s)
    })
    return () => { unsub() }
  }, [])

  const handleCfgChange = (field: keyof ServerConfig, value: any) => {
    setCfg(prev => ({ ...prev, [field]: value }))
    setCfgDirty(true)
  }

  const handleSaveCfg = () => {
    setSavingCfg(true)
    ServerService.SaveConfig(cfg)
      .then(() => { notify('設定を保存しました'); setCfgDirty(false) })
      .catch((e: any) => notify(String(e), 'error'))
      .finally(() => setSavingCfg(false))
  }

  const handleStart = () => {
    ServerService.SaveConfig(cfg)
      .then(() => ServerService.Start())
      .then(() => { setCfgDirty(false); notify('サーバを起動しました') })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleStop = () => {
    ServerService.Stop()
      .then(() => notify('サーバを停止しました'))
      .catch((e: any) => notify(String(e), 'error'))
  }

  // --- エンジン・コード ---
  const copy = (text: string) => {
    navigator.clipboard.writeText(text).then(() => notify('コピーしました'))
  }

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
      .then((id: any) => { notify(`登録しました: ${id}`); setEnginePath(''); loadEngines() })
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

  const monoCell = { fontFamily: 'monospace', fontSize: '0.75rem', color: 'primary.light' } as const
  const subCell = { color: 'text.secondary', fontSize: '0.75rem' } as const

  return (
    <>
      <Box sx={{ flex: 1, overflow: 'auto' }}>
        <Container maxWidth="md" sx={{ py: 3 }}>
          <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2 }}>
            <Tab label="サーバ設定" />
            <Tab label="エンジン管理" />
            <Tab label="認証コード管理" />
          </Tabs>

          {/* サーバ設定タブ */}
          {tab === 0 && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
              {/* 状態 */}
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
                  <Chip
                    label={serverState.running ? '起動中' : '停止中'}
                    color={serverState.running ? 'success' : 'default'}
                    size="small"
                    variant={serverState.running ? 'filled' : 'outlined'}
                  />
                  {serverState.running && serverState.url && (
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                      <Typography sx={{ fontFamily: 'monospace', fontSize: '0.85rem', color: 'success.light' }}>
                        {serverState.url}
                      </Typography>
                      <Tooltip title="URLをコピー">
                        <IconButton size="small" onClick={() => copy(serverState.url)} sx={{ opacity: 0.6, '&:hover': { opacity: 1 } }}>
                          <CopyIcon sx={{ fontSize: 13 }} />
                        </IconButton>
                      </Tooltip>
                    </Box>
                  )}
                  <Box sx={{ ml: 'auto', display: 'flex', gap: 1 }}>
                    {!serverState.running ? (
                      <Button variant="contained" color="success" startIcon={<StartIcon />} onClick={handleStart} size="small">
                        起動
                      </Button>
                    ) : (
                      <Button variant="outlined" color="error" startIcon={<StopIcon />} onClick={handleStop} size="small">
                        停止
                      </Button>
                    )}
                  </Box>
                </Box>
              </Paper>

              {/* 設定フォーム */}
              <Paper variant="outlined" sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <TextField
                  label="ホスト"
                  size="small"
                  fullWidth
                  value={cfg.host}
                  onChange={e => handleCfgChange('host', e.target.value)}
                  placeholder="空欄 = すべてのインターフェース (0.0.0.0)"
                  helperText="特定のIPアドレスにバインドする場合に入力"
                />
                <TextField
                  label="ポート"
                  size="small"
                  type="number"
                  value={cfg.port}
                  onChange={e => handleCfgChange('port', parseInt(e.target.value) || 8080)}
                  slotProps={{ htmlInput: { min: 1, max: 65535 } }}
                  sx={{ width: 160 }}
                />
                <FormControlLabel
                  control={
                    <Switch
                      checked={cfg.autoStart}
                      onChange={e => handleCfgChange('autoStart', e.target.checked)}
                      color="primary"
                    />
                  }
                  label="起動時にサーバを自動起動"
                />
                <Box>
                  <Button
                    variant="contained"
                    onClick={handleSaveCfg}
                    disabled={!cfgDirty || savingCfg}
                    size="small"
                  >
                    設定を保存
                  </Button>
                  {cfgDirty && (
                    <Typography variant="caption" sx={{ ml: 1.5, color: 'warning.main' }}>
                      未保存の変更があります
                    </Typography>
                  )}
                </Box>
              </Paper>
            </Box>
          )}

          {/* エンジン管理タブ */}
          {tab === 1 && (
            <Box>
              <Box sx={{ display: 'flex', gap: 1, mb: 2, alignItems: 'center' }}>
                <TextField
                  size="small" fullWidth label="エンジンのパス" value={enginePath}
                  onChange={e => setEnginePath(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleRegisterEngine()}
                  placeholder="C:\engines\apery.exe"
                />
                <Tooltip title="ファイルを選択">
                  <IconButton onClick={handleSelectFile} color="primary"><FolderOpenIcon /></IconButton>
                </Tooltip>
                <Button variant="contained" startIcon={<AddIcon />} onClick={handleRegisterEngine}
                  disabled={!enginePath.trim()} sx={{ whiteSpace: 'nowrap' }}>登録</Button>
                <Tooltip title="更新">
                  <IconButton onClick={loadEngines}><RefreshIcon /></IconButton>
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
                      <TableRow><TableCell colSpan={4} align="center" sx={{ py: 3 }}><CircularProgress size={24} /></TableCell></TableRow>
                    ) : engines.length === 0 ? (
                      <TableRow><TableCell colSpan={4} align="center" sx={{ py: 3, color: 'text.disabled' }}>登録済みエンジンがありません</TableCell></TableRow>
                    ) : engines.map(e => (
                      <TableRow key={e.id} hover>
                        <TableCell sx={{ ...monoCell, whiteSpace: 'nowrap' }}>
                          {e.id}
                          <Tooltip title="コピー">
                            <IconButton size="small" onClick={() => copy(e.id)} sx={{ ml: 0.5, opacity: 0.5, '&:hover': { opacity: 1 } }}>
                              <CopyIcon sx={{ fontSize: 13 }} />
                            </IconButton>
                          </Tooltip>
                        </TableCell>
                        <TableCell>{e.path}</TableCell>
                        <TableCell sx={subCell}>{e.created}</TableCell>
                        <TableCell>
                          <Tooltip title="削除">
                            <IconButton size="small" color="error" onClick={() => handleDeleteEngine(e.id)}><DeleteIcon fontSize="small" /></IconButton>
                          </Tooltip>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>
          )}

          {/* 認証コード管理タブ */}
          {tab === 2 && (
            <Box>
              <Box sx={{ display: 'flex', gap: 1, mb: 2, alignItems: 'center' }}>
                <TextField
                  size="small" fullWidth label="認証コード" value={codeInput}
                  onChange={e => setCodeInput(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleRegisterCode()}
                  placeholder="任意のコード文字列"
                />
                <Button variant="contained" startIcon={<AddIcon />} onClick={handleRegisterCode}
                  disabled={!codeInput.trim()} sx={{ whiteSpace: 'nowrap' }}>登録</Button>
                <Button variant="outlined" startIcon={<GenerateIcon />} onClick={handleGenerateCode}
                  sx={{ whiteSpace: 'nowrap' }}>自動生成</Button>
                <Tooltip title="更新">
                  <IconButton onClick={loadCodes}><RefreshIcon /></IconButton>
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
                      <TableRow><TableCell colSpan={4} align="center" sx={{ py: 3 }}><CircularProgress size={24} /></TableCell></TableRow>
                    ) : codes.length === 0 ? (
                      <TableRow><TableCell colSpan={4} align="center" sx={{ py: 3, color: 'text.disabled' }}>登録済みコードがありません</TableCell></TableRow>
                    ) : codes.map(c => (
                      <TableRow key={c.code} hover>
                        <TableCell sx={{ ...monoCell, whiteSpace: 'nowrap' }}>
                          {c.code}
                          <Tooltip title="コピー">
                            <IconButton size="small" onClick={() => copy(c.code)} sx={{ ml: 0.5, opacity: 0.5, '&:hover': { opacity: 1 } }}>
                              <CopyIcon sx={{ fontSize: 13 }} />
                            </IconButton>
                          </Tooltip>
                        </TableCell>
                        <TableCell sx={subCell}>{c.created}</TableCell>
                        <TableCell sx={subCell}>{c.used || '—'}</TableCell>
                        <TableCell>
                          <Tooltip title="削除">
                            <IconButton size="small" color="error" onClick={() => handleDeleteCode(c.code)}><DeleteIcon fontSize="small" /></IconButton>
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

      <Snackbar open={!!snack} autoHideDuration={4000} onClose={() => setSnack(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        <Alert severity={snack?.sev} onClose={() => setSnack(null)} variant="filled">
          {snack?.msg}
        </Alert>
      </Snackbar>
    </>
  )
}

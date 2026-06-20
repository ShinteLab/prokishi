import { useState, useEffect, useCallback } from 'react'
import {
  Box, Button, Chip, Container, Dialog, DialogActions, DialogContent,
  DialogTitle, FormControlLabel, IconButton, MenuItem,
  Paper, Snackbar, Alert, Select, Switch, Tab, Tabs, Table, TableBody,
  TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip,
  CircularProgress, Typography, InputLabel, FormControl,
} from '@mui/material'
import {
  Add as AddIcon,
  AutoFixHigh as GenerateIcon,
  Block as BlockIcon,
  CheckCircleOutlined as EnableIcon,
  ContentCopy as CopyIcon,
  Delete as DeleteIcon,
  Download as DownloadIcon,
  FolderOpen as FolderOpenIcon,
  PlayArrow as StartIcon,
  Refresh as RefreshIcon,
  Stop as StopIcon,
} from '@mui/icons-material'
import { Events } from '@wailsio/runtime'
import { AdminService, ServerService } from '../bindings/wails'

type EngineItem = { id: string; name: string; path: string; created: string }
type CodeItem = { code: string; created: string; used: string; disabled: boolean }
type ServerConfig = { host: string; port: number; autoStart: boolean }
type ServerState = { running: boolean; url: string }
type Severity = 'success' | 'error'

export function MasterView() {
  const [tab, setTab] = useState(0)
  // エンジン
  const [engines, setEngines] = useState<EngineItem[]>([])
  const [codes, setCodes] = useState<CodeItem[]>([])
  const [enginePath, setEnginePath] = useState('')
  const [engineName, setEngineName] = useState('')
  const [editingName, setEditingName] = useState<{ id: string; value: string } | null>(null)
  const [codeInput, setCodeInput] = useState('')
  const [loadingEngines, setLoadingEngines] = useState(false)
  const [loadingCodes, setLoadingCodes] = useState(false)
  // サーバ設定
  const [serverState, setServerState] = useState<ServerState>({ running: false, url: '' })
  const [cfg, setCfg] = useState<ServerConfig>({ host: '', port: 8080, autoStart: true })
  const [cfgDirty, setCfgDirty] = useState(false)
  const [savingCfg, setSavingCfg] = useState(false)
  const [localIPs, setLocalIPs] = useState<string[]>([])
  // クライアント設定
  const [clientHost, setClientHost] = useState('localhost')
  const [clientPort, setClientPort] = useState(8080)
  const [clientEngineId, setClientEngineId] = useState('')
  const [clientCode, setClientCode] = useState('')
  const [clientLogLevel, setClientLogLevel] = useState('warn')
  // 共通
  const [snack, setSnack] = useState<{ msg: string; sev: Severity } | null>(null)
  const [errorDialog, setErrorDialog] = useState<string | null>(null)

  const notify = (msg: string, sev: Severity = 'success') => {
    if (sev === 'error') { setErrorDialog(msg); return }
    setSnack({ msg, sev })
  }

  // --- サーバ設定 ---
  useEffect(() => {
    ServerService.GetState().then((s: any) => setServerState(s)).catch(() => {})
    ServerService.GetConfig().then((c: any) => {
      if (c) { setCfg(c); setClientPort(c.port ?? 8080) }
    }).catch(() => {})
    ServerService.GetLocalIPs().then((ips: any) => setLocalIPs(ips ?? [])).catch(() => {})

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

  // --- クライアント設定 ---
  const handleDownloadClientConfig = () => {
    AdminService.SaveClientConfig(clientHost, clientPort, clientCode, clientEngineId, clientLogLevel)
      .then(() => notify('prokishi.ini を保存しました'))
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
    AdminService.RegisterEngine(path, engineName.trim())
      .then((id: any) => { notify(`登録しました: ${id}`); setEnginePath(''); setEngineName(''); loadEngines() })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleSaveEngineName = (id: string, name: string) => {
    AdminService.UpdateEngineName(id, name)
      .then(() => { setEditingName(null); loadEngines() })
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

  const handleDisableCode = (code: string) => {
    AdminService.DisableCode(code)
      .then(() => { notify('停止しました'); loadCodes() })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleEnableCode = (code: string) => {
    AdminService.EnableCode(code)
      .then(() => { notify('有効にしました'); loadCodes() })
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
            <Tab label="クライアント設定" />
          </Tabs>

          {/* サーバ設定タブ */}
          {tab === 0 && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
              {/* 状態 */}
              <Paper variant="outlined" sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
                  <Chip
                    label={serverState.running ? '起動中' : '停止中'}
                    color={serverState.running ? 'success' : 'default'}
                    size="small"
                    variant={serverState.running ? 'filled' : 'outlined'}
                  />
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
                {serverState.running && (() => {
                  // host が空 = 0.0.0.0（全IF）→ ローカルIP一覧を表示
                  // host が設定済み → その値だけを表示（localhostならlocalhostのまま）
                  const hosts = cfg.host ? [cfg.host] : localIPs
                  if (hosts.length === 0) return null
                  return (
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                      <Typography variant="caption" sx={{ color: 'text.disabled' }}>
                        クライアントの接続先 (prokishi.ini の host に設定)
                      </Typography>
                      {hosts.map(h => (
                        <Box key={h} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          <Typography sx={{ fontFamily: 'monospace', fontSize: '0.85rem', color: 'success.light' }}>
                            {h}:{cfg.port}
                          </Typography>
                          <Tooltip title="host をコピー">
                            <IconButton size="small" onClick={() => copy(h)} sx={{ opacity: 0.5, '&:hover': { opacity: 1 } }}>
                              <CopyIcon sx={{ fontSize: 13 }} />
                            </IconButton>
                          </Tooltip>
                        </Box>
                      ))}
                    </Box>
                  )
                })()}
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
              <Box sx={{ display: 'flex', gap: 1, mb: 1, alignItems: 'center' }}>
                <TextField
                  size="small" fullWidth label="エンジンのパス" value={enginePath}
                  onChange={e => setEnginePath(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleRegisterEngine()}
                  placeholder="C:\engines\apery.exe"
                />
                <Tooltip title="ファイルを選択">
                  <IconButton onClick={handleSelectFile} color="primary"><FolderOpenIcon /></IconButton>
                </Tooltip>
              </Box>
              <Box sx={{ display: 'flex', gap: 1, mb: 2, alignItems: 'center' }}>
                <TextField
                  size="small" fullWidth label="名称 (任意)" value={engineName}
                  onChange={e => setEngineName(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleRegisterEngine()}
                  placeholder="例: Apery WCSC"
                />
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
                      <TableCell>名称 / ID</TableCell>
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
                        <TableCell sx={{ whiteSpace: 'nowrap' }}>
                          {editingName?.id === e.id ? (
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                              <TextField
                                size="small" value={editingName.value} autoFocus
                                onChange={ev => setEditingName({ id: e.id, value: ev.target.value })}
                                onKeyDown={ev => {
                                  if (ev.key === 'Enter') handleSaveEngineName(e.id, editingName.value)
                                  if (ev.key === 'Escape') setEditingName(null)
                                }}
                                sx={{ width: 160 }}
                              />
                              <Button size="small" onClick={() => handleSaveEngineName(e.id, editingName.value)}>保存</Button>
                              <Button size="small" color="inherit" onClick={() => setEditingName(null)}>取消</Button>
                            </Box>
                          ) : (
                            <Box>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                <Typography sx={{ fontSize: '0.875rem', fontWeight: e.name ? 500 : 'normal', color: e.name ? 'text.primary' : 'text.disabled' }}>
                                  {e.name || '(名称未設定)'}
                                </Typography>
                                <Tooltip title="名称を編集">
                                  <IconButton size="small" onClick={() => setEditingName({ id: e.id, value: e.name })} sx={{ opacity: 0.4, '&:hover': { opacity: 1 } }}>
                                    <span style={{ fontSize: 11 }}>✏️</span>
                                  </IconButton>
                                </Tooltip>
                              </Box>
                              <Box sx={{ ...monoCell, display: 'flex', alignItems: 'center', gap: 0.3 }}>
                                {e.id}
                                <Tooltip title="IDをコピー">
                                  <IconButton size="small" onClick={() => copy(e.id)} sx={{ opacity: 0.5, '&:hover': { opacity: 1 } }}>
                                    <CopyIcon sx={{ fontSize: 12 }} />
                                  </IconButton>
                                </Tooltip>
                              </Box>
                            </Box>
                          )}
                        </TableCell>
                        <TableCell sx={{ color: 'text.secondary', fontSize: '0.78rem' }}>{e.path}</TableCell>
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
                      <TableCell sx={{ width: 90 }} />
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {loadingCodes ? (
                      <TableRow><TableCell colSpan={4} align="center" sx={{ py: 3 }}><CircularProgress size={24} /></TableCell></TableRow>
                    ) : codes.length === 0 ? (
                      <TableRow><TableCell colSpan={4} align="center" sx={{ py: 3, color: 'text.disabled' }}>登録済みコードがありません</TableCell></TableRow>
                    ) : codes.map(c => (
                      <TableRow key={c.code} hover sx={c.disabled ? { opacity: 0.45 } : undefined}>
                        <TableCell sx={{ ...monoCell, whiteSpace: 'nowrap' }}>
                          {c.code}
                          <Tooltip title="コピー">
                            <IconButton size="small" onClick={() => copy(c.code)} sx={{ ml: 0.5, opacity: 0.5, '&:hover': { opacity: 1 } }}>
                              <CopyIcon sx={{ fontSize: 13 }} />
                            </IconButton>
                          </Tooltip>
                          {c.disabled && (
                            <Chip label="停止中" size="small" color="warning" variant="outlined" sx={{ ml: 0.5, height: 16, fontSize: '0.65rem' }} />
                          )}
                        </TableCell>
                        <TableCell sx={subCell}>{c.created}</TableCell>
                        <TableCell sx={subCell}>{c.used || '—'}</TableCell>
                        <TableCell sx={{ whiteSpace: 'nowrap' }}>
                          {c.disabled ? (
                            <Tooltip title="有効にする">
                              <IconButton size="small" color="success" onClick={() => handleEnableCode(c.code)}><EnableIcon fontSize="small" /></IconButton>
                            </Tooltip>
                          ) : (
                            <Tooltip title="停止">
                              <IconButton size="small" color="warning" onClick={() => handleDisableCode(c.code)}><BlockIcon fontSize="small" /></IconButton>
                            </Tooltip>
                          )}
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
          {/* クライアント設定タブ */}
          {tab === 3 && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
              <Paper variant="outlined" sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Typography variant="body2" sx={{ color: 'text.secondary' }}>
                  クライアント (prokishi) の接続設定ファイルを生成します。
                </Typography>

                <TextField
                  label="サーバホスト"
                  size="small"
                  value={clientHost}
                  onChange={e => setClientHost(e.target.value)}
                  placeholder="localhost"
                  helperText="クライアントがサーバに接続するホスト名または IP アドレス"
                />

                <TextField
                  label="ポート"
                  size="small"
                  type="number"
                  value={clientPort}
                  onChange={e => setClientPort(parseInt(e.target.value) || 8080)}
                  slotProps={{ htmlInput: { min: 1, max: 65535 } }}
                  sx={{ width: 160 }}
                />

                <FormControl size="small" fullWidth>
                  <InputLabel>エンジン ID</InputLabel>
                  <Select
                    value={clientEngineId}
                    label="エンジン ID"
                    onChange={e => setClientEngineId(e.target.value)}
                    displayEmpty
                  >
                    <MenuItem value=""><em>— 選択してください —</em></MenuItem>
                    {engines.map(e => (
                      <MenuItem key={e.id} value={e.id}>
                        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                          <Typography sx={{ fontSize: '0.85rem', fontWeight: e.name ? 500 : 'normal' }}>
                            {e.name || e.path.split('\\').pop()}
                          </Typography>
                          <Typography sx={{ fontFamily: 'monospace', fontSize: '0.7rem', color: 'text.secondary' }}>{e.id}</Typography>
                        </Box>
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <FormControl size="small" fullWidth>
                  <InputLabel>認証コード (任意)</InputLabel>
                  <Select
                    value={clientCode}
                    label="認証コード (任意)"
                    onChange={e => setClientCode(e.target.value)}
                  >
                    <MenuItem value="">— なし —</MenuItem>
                    {codes.map(c => (
                      <MenuItem key={c.code} value={c.code}>
                        <Typography sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>{c.code}</Typography>
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <FormControl size="small" sx={{ width: 160 }}>
                  <InputLabel>ログレベル</InputLabel>
                  <Select value={clientLogLevel} label="ログレベル" onChange={e => setClientLogLevel(e.target.value)}>
                    <MenuItem value="debug">debug</MenuItem>
                    <MenuItem value="info">info</MenuItem>
                    <MenuItem value="warn">warn</MenuItem>
                    <MenuItem value="error">error</MenuItem>
                  </Select>
                </FormControl>

                {/* プレビュー */}
                <Box sx={{ bgcolor: 'background.default', borderRadius: 1, p: 1.5, border: '1px solid', borderColor: 'divider' }}>
                  <Typography variant="caption" sx={{ color: 'text.disabled', display: 'block', mb: 0.5 }}>prokishi.ini プレビュー</Typography>
                  <Box component="pre" sx={{ fontFamily: 'monospace', fontSize: '0.78rem', color: 'text.primary', m: 0 }}>
                    {`host = "${clientHost}"\nport = ${clientPort}\ncode = "${clientCode}"\nengineId = "${clientEngineId}"\nlogLevel = "${clientLogLevel}"`}
                  </Box>
                </Box>

                <Box>
                  <Button
                    variant="contained"
                    startIcon={<DownloadIcon />}
                    onClick={handleDownloadClientConfig}
                    disabled={!clientEngineId}
                  >
                    prokishi.ini を保存
                  </Button>
                  {!clientEngineId && (
                    <Typography variant="caption" sx={{ ml: 1.5, color: 'warning.main' }}>
                      エンジン ID を選択してください
                    </Typography>
                  )}
                </Box>
              </Paper>
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

      <Dialog open={!!errorDialog} onClose={() => setErrorDialog(null)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ color: 'error.main' }}>エラー</DialogTitle>
        <DialogContent>
          <Box
            component="pre"
            sx={{
              fontFamily: 'monospace', fontSize: '0.78rem', whiteSpace: 'pre-wrap',
              wordBreak: 'break-all', userSelect: 'text', m: 0,
              bgcolor: 'background.default', p: 1.5, borderRadius: 1,
              border: '1px solid', borderColor: 'divider',
            }}
          >
            {errorDialog}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => navigator.clipboard.writeText(errorDialog ?? '')} size="small">コピー</Button>
          <Button onClick={() => setErrorDialog(null)} variant="contained" size="small">閉じる</Button>
        </DialogActions>
      </Dialog>
    </>
  )
}

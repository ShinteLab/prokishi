import { useState, useEffect, useCallback } from 'react'
import {
  Box, Button, Chip, Container, Dialog, DialogActions, DialogContent,
  DialogTitle, FormControlLabel, IconButton, MenuItem,
  Paper, Snackbar, Alert, Select, Switch, Tab, Tabs, Table, TableBody,
  TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip,
  Autocomplete, CircularProgress, Typography, InputLabel, FormControl,
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
import { AdminService, ServerService } from '../bindings/prokishi-server'

type EngineItem = { id: string; name: string; path: string; created: string }
type CodeItem = { code: string; name: string; created: string; used: string; disabled: boolean }
type ServerConfig = { host: string; port: number; autoStart: boolean; useAuth: boolean }
type ServerState = { running: boolean; url: string; err?: string }
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
  const [codeName, setCodeName] = useState('')
  const [editingCode, setEditingCode] = useState<{ code: string; value: string } | null>(null)
  const [loadingEngines, setLoadingEngines] = useState(false)
  const [loadingCodes, setLoadingCodes] = useState(false)
  // サーバ設定
  const [serverState, setServerState] = useState<ServerState>({ running: false, url: '' })
  const [cfg, setCfg] = useState<ServerConfig>({ host: '', port: 8080, autoStart: true, useAuth: false })
  const [cfgDirty, setCfgDirty] = useState(false)
  const [savingCfg, setSavingCfg] = useState(false)
  const [needsRestart, setNeedsRestart] = useState(false)
  const [localIPs, setLocalIPs] = useState<string[]>([])
  // クライアント設定
  const [clientHost, setClientHost] = useState('localhost')
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
    ServerService.GetState().then((s: any) => {
      setServerState(s)
      // 自動起動が失敗していた場合はここで初めて画面に出る。
      if (s?.err && !s.running) notify(s.err, 'error')
    }).catch(() => {})
    ServerService.GetConfig().then((c: any) => {
      if (c) { setCfg(c) }
    }).catch(() => {})
    ServerService.GetLocalIPs().then((ips: any) => setLocalIPs(ips ?? [])).catch(() => {})

    const unsub = Events.On('server-state', (e: any) => {
      const s: ServerState = e.data
      if (!s) return
      setServerState(s)
      // サーバが自発的に落ちた場合（Stop() 経由の停止では err は空）。
      if (s.err && !s.running) notify(s.err, 'error')
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
      .then(() => { notify('設定を保存しました'); setCfgDirty(false); if (serverState.running) setNeedsRestart(true) })
      .catch((e: any) => notify(String(e), 'error'))
      .finally(() => setSavingCfg(false))
  }

  const handleStart = () => {
    ServerService.SaveConfig(cfg)
      .then(() => ServerService.Start())
      .then(() => { setCfgDirty(false); setNeedsRestart(false); notify('サーバを起動しました') })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleStop = () => {
    ServerService.Stop()
      .then(() => { setNeedsRestart(false); notify('サーバを停止しました') })
      .catch((e: any) => notify(String(e), 'error'))
  }

  // --- クライアント設定 ---
  const handleDownloadClientConfig = () => {
    const host = cfg.host || clientHost
    const code = cfg.useAuth ? clientCode : ''
    AdminService.SaveClientConfig(host, cfg.port, code, clientEngineId, clientLogLevel)
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
    AdminService.GenerateCode(codeName.trim())
      .then((code: any) => { notify(`生成しました: ${code}`); setCodeName(''); loadCodes() })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleRegisterCode = () => {
    const code = codeInput.trim()
    if (!code) return
    AdminService.RegisterCode(code, codeName.trim())
      .then(() => { notify('登録しました'); setCodeInput(''); setCodeName(''); loadCodes() })
      .catch((e: any) => notify(String(e), 'error'))
  }

  const handleSaveCodeName = (code: string, name: string) => {
    AdminService.UpdateCodeName(code, name)
      .then(() => { setEditingCode(null); loadCodes() })
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
          <Tabs
            value={tab}
            onChange={(_, v) => setTab(v)}
            variant="scrollable"
            scrollButtons="auto"
            sx={{
              mb: 2,
              '& .MuiTabs-flexContainer': { gap: 1 },
              '& .MuiTab-root': { minWidth: 130, px: '24px !important' },
            }}
          >
            <Tab label="サーバ設定" />
            <Tab label="エンジン管理" />
            <Tab label="認証コード管理" />
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
                {serverState.running && serverState.url && (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                      <Typography variant="caption" sx={{ color: 'text.disabled' }}>
                        リッスン中:
                      </Typography>
                      <Typography sx={{ fontFamily: 'monospace', fontSize: '0.85rem', color: 'success.light' }}>
                        {serverState.url}
                      </Typography>
                    </Box>
                    {needsRestart && (
                      <Typography variant="caption" sx={{ color: 'warning.main' }}>
                        設定が変更されています。反映するにはサーバを再起動してください。
                      </Typography>
                    )}
                  </Box>
                )}
              </Paper>

              {/* 設定フォーム */}
              <Paper variant="outlined" sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <TextField
                  label="ホスト"
                  size="small"
                  fullWidth
                  value={cfg.host}
                  onChange={e => handleCfgChange('host', e.target.value)}
                  placeholder="空欄 = すべての接続を受け入れ"
                  helperText="localhost や 127.0.0.1 を指定すると同じ端末からのみ接続可能です"
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
                <FormControlLabel
                  control={
                    <Switch
                      checked={cfg.useAuth}
                      onChange={e => handleCfgChange('useAuth', e.target.checked)}
                      color="primary"
                    />
                  }
                  label="認証コードによる接続認証を使用"
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

              {/* 設定ファイル生成 */}
              <Paper variant="outlined" sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Typography variant="subtitle2" sx={{ color: 'text.secondary' }}>
                  設定ファイル生成
                </Typography>
                <Typography variant="body2" sx={{ color: 'text.disabled', mt: -1 }}>
                  クライアント (prokishi) の接続設定ファイルを生成します。
                </Typography>

                {cfg.host ? (
                  <TextField
                    label="接続先ホスト"
                    size="small"
                    fullWidth
                    value={cfg.host}
                    disabled
                    helperText="サーバのホストが指定されているため固定です"
                  />
                ) : (
                  <Autocomplete
                    freeSolo
                    options={localIPs}
                    value={clientHost}
                    onInputChange={(_, v) => setClientHost(v)}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="接続先ホスト"
                        size="small"
                        placeholder="IP を選択またはドメインを入力"
                      />
                    )}
                  />
                )}

                <FormControl size="small" fullWidth>
                  <InputLabel>エンジン ID</InputLabel>
                  <Select
                    value={clientEngineId}
                    label="エンジン ID"
                    onChange={e => setClientEngineId(e.target.value)}
                  >
                    <MenuItem value="">— 未選択 —</MenuItem>
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

                {cfg.useAuth && (
                  <FormControl size="small" fullWidth>
                    <InputLabel>認証コード</InputLabel>
                    <Select
                      value={clientCode}
                      label="認証コード"
                      onChange={e => setClientCode(e.target.value)}
                    >
                      <MenuItem value="">— 未選択 —</MenuItem>
                      {codes.map(c => (
                        <MenuItem key={c.code} value={c.code}>
                          <Typography sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>{c.code}</Typography>
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                )}

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
                    {`host = "${cfg.host || clientHost}"\nport = ${cfg.port}\ncode = "${cfg.useAuth ? clientCode : ''}"\nengineId = "${clientEngineId}"\nlogLevel = "${clientLogLevel}"`}
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
                  disabled={!enginePath.trim()}>登録</Button>
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
              <Box sx={{ display: 'flex', gap: 1, mb: 1, alignItems: 'center' }}>
                <TextField
                  size="small" fullWidth label="認証コード" value={codeInput}
                  onChange={e => setCodeInput(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleRegisterCode()}
                  placeholder="任意のコード文字列"
                />
              </Box>
              <Box sx={{ display: 'flex', gap: 1, mb: 2, alignItems: 'center' }}>
                <TextField
                  size="small" fullWidth label="名称 (任意)" value={codeName}
                  onChange={e => setCodeName(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleRegisterCode()}
                  placeholder="例: 山田太郎"
                />
                <Button variant="contained" startIcon={<AddIcon />} onClick={handleRegisterCode}
                  disabled={!codeInput.trim()}>登録</Button>
                <Button variant="outlined" startIcon={<GenerateIcon />} onClick={handleGenerateCode}>自動生成</Button>
                <Tooltip title="更新">
                  <IconButton onClick={loadCodes}><RefreshIcon /></IconButton>
                </Tooltip>
              </Box>
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>名称 / コード</TableCell>
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
                        <TableCell sx={{ whiteSpace: 'nowrap' }}>
                          {editingCode?.code === c.code ? (
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                              <TextField
                                size="small" value={editingCode.value} autoFocus
                                onChange={ev => setEditingCode({ code: c.code, value: ev.target.value })}
                                onKeyDown={ev => {
                                  if (ev.key === 'Enter') handleSaveCodeName(c.code, editingCode.value)
                                  if (ev.key === 'Escape') setEditingCode(null)
                                }}
                                sx={{ width: 160 }}
                              />
                              <Button size="small" onClick={() => handleSaveCodeName(c.code, editingCode.value)}>保存</Button>
                              <Button size="small" color="inherit" onClick={() => setEditingCode(null)}>取消</Button>
                            </Box>
                          ) : (
                            <Box>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                <Typography sx={{ fontSize: '0.875rem', fontWeight: c.name ? 500 : 'normal', color: c.name ? 'text.primary' : 'text.disabled' }}>
                                  {c.name || '(名称未設定)'}
                                </Typography>
                                <Tooltip title="名称を編集">
                                  <IconButton size="small" onClick={() => setEditingCode({ code: c.code, value: c.name })} sx={{ opacity: 0.4, '&:hover': { opacity: 1 } }}>
                                    <span style={{ fontSize: 11 }}>✏️</span>
                                  </IconButton>
                                </Tooltip>
                                {c.disabled && (
                                  <Chip label="停止中" size="small" color="warning" variant="outlined" sx={{ height: 16, fontSize: '0.65rem' }} />
                                )}
                              </Box>
                              <Box sx={{ ...monoCell, display: 'flex', alignItems: 'center', gap: 0.3 }}>
                                {c.code}
                                <Tooltip title="コピー">
                                  <IconButton size="small" onClick={() => copy(c.code)} sx={{ opacity: 0.5, '&:hover': { opacity: 1 } }}>
                                    <CopyIcon sx={{ fontSize: 12 }} />
                                  </IconButton>
                                </Tooltip>
                              </Box>
                            </Box>
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

import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { Box, Chip, Divider, IconButton, List, ListItemButton, ListItemText, Tooltip, Typography } from '@mui/material'
import RefreshIcon from '@mui/icons-material/Refresh'
import ViewColumnIcon from '@mui/icons-material/ViewColumn'
import ViewStreamIcon from '@mui/icons-material/ViewStream'
import WrapTextIcon from '@mui/icons-material/WrapText'
import { Events } from '@wailsio/runtime'
import { DebugService, ServerService } from '../bindings/wails'

type ConnectionInfo = { id: string; engineId: string; enginePath: string; connectedAt: any; active?: boolean }
type LogEntryItem  = { timestamp: any; dir: number; message: string }
type SendRange     = { from: number; to: number } | null
type HighlightWin  = { from: number; to: number } | null

const SEND_COLOR = 'success.main'
const RECV_COLOR = 'info.main'
const SEND_LIGHT = 'success.light'
const RECV_LIGHT = 'info.light'

const toMs = (ts: any): number => {
  if (!ts) return 0
  try { return new Date(ts).getTime() } catch { return 0 }
}
const fmt = (ts: any) => {
  if (!ts) return ''
  try { return new Date(ts).toLocaleTimeString('ja-JP', { hour12: false }) } catch { return String(ts) }
}

type HL = 'normal' | 'hi' | 'dim'

function sendHL(sendRange: SendRange, i: number): HL {
  if (!sendRange) return 'normal'
  return i >= sendRange.from && i <= sendRange.to ? 'hi' : 'dim'
}
function recvHL(win: HighlightWin, ts: any): HL {
  if (!win) return 'normal'
  const ms = toMs(ts)
  return ms >= win.from && ms < win.to ? 'hi' : 'dim'
}

// 時系列1行
function TimelineLine({ entry, wrap, hl, onClick }: {
  entry: LogEntryItem; wrap: boolean; hl: HL; onClick?: (e: React.MouseEvent) => void
}) {
  const isSend = entry.dir === 0
  return (
    <Box
      onClick={isSend ? onClick : undefined}
      sx={{
        display: 'flex', gap: 1, alignItems: 'flex-start',
        opacity: hl === 'dim' ? 0.22 : 1,
        cursor: isSend ? 'pointer' : 'default',
        borderRadius: '3px', px: 0.5,
        bgcolor: hl === 'hi' && isSend ? 'action.selected' : hl === 'hi' ? 'action.hover' : 'transparent',
        '&:hover': isSend ? { bgcolor: 'action.hover' } : {},
        transition: 'opacity 0.12s, background-color 0.12s',
        userSelect: 'none',
      }}
    >
      <Box sx={{ color: 'text.disabled', whiteSpace: 'nowrap', flexShrink: 0, fontSize: '0.7rem', mt: '1px' }}>
        {fmt(entry.timestamp)}
      </Box>
      <Box sx={{ color: isSend ? SEND_LIGHT : RECV_LIGHT, flexShrink: 0 }}>
        {isSend ? '→' : '←'}
      </Box>
      <Box sx={{ color: isSend ? SEND_COLOR : RECV_COLOR, ...(wrap ? { wordBreak: 'break-all' } : { whiteSpace: 'nowrap' }) }}>
        {entry.message}
      </Box>
    </Box>
  )
}

// 分割ビュー1ペイン
function SplitPane({ logs, dir, wrap, bottomRef, sendRange, onSendClick, highlightWin }: {
  logs: LogEntryItem[]; dir: 0 | 1; wrap: boolean
  bottomRef?: React.RefObject<HTMLDivElement>
  sendRange?: SendRange
  onSendClick?: (idx: number, shift: boolean) => void
  highlightWin?: HighlightWin
}) {
  const isSend = dir === 0
  const filtered = useMemo(() => logs.filter(e => e.dir === dir), [logs, dir])
  const color = isSend ? SEND_COLOR : RECV_COLOR
  const label = isSend ? '送信 (client → engine)' : '受信 (engine → client)'

  return (
    <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', minWidth: 0 }}>
      <Box sx={{ px: 1.5, py: 0.5, borderBottom: '1px solid', borderColor: 'divider', flexShrink: 0 }}>
        <Typography variant="caption" sx={{ color, fontWeight: 600, fontSize: '0.7rem' }}>{label}</Typography>
      </Box>
      <Box sx={{ flex: 1, overflow: 'auto', px: 1.5, py: 1, fontFamily: 'monospace', fontSize: '0.75rem', lineHeight: 1.6 }}>
        {filtered.length === 0 && <Box sx={{ color: 'text.disabled' }}>なし</Box>}
        {filtered.map((entry, i) => {
          const hl: HL = isSend ? sendHL(sendRange ?? null, i) : recvHL(highlightWin ?? null, entry.timestamp)
          return (
            <Box
              key={i}
              onClick={isSend && onSendClick ? (e: React.MouseEvent) => onSendClick(i, e.shiftKey) : undefined}
              sx={{
                display: 'flex', gap: 1, alignItems: 'flex-start',
                opacity: hl === 'dim' ? 0.22 : 1,
                cursor: isSend ? 'pointer' : 'default',
                borderRadius: '3px', px: 0.5,
                bgcolor: hl === 'hi' ? 'action.selected' : 'transparent',
                '&:hover': isSend ? { bgcolor: 'action.hover' } : {},
                transition: 'opacity 0.12s, background-color 0.12s',
                userSelect: 'none',
              }}
            >
              <Box sx={{ color: 'text.disabled', whiteSpace: 'nowrap', flexShrink: 0, fontSize: '0.7rem', mt: '1px' }}>
                {fmt(entry.timestamp)}
              </Box>
              <Box sx={{ color, ...(wrap ? { wordBreak: 'break-all' } : { whiteSpace: 'nowrap' }) }}>
                {entry.message}
              </Box>
            </Box>
          )
        })}
        {bottomRef && <div ref={bottomRef} />}
      </Box>
    </Box>
  )
}

export function MonitorView() {
  const [serverState, setServerState] = useState<{ running: boolean; url: string }>({ running: false, url: '' })
  const [connections, setConnections] = useState<ConnectionInfo[]>([])
  const [selectedID, setSelectedID] = useState<string | null>(null)
  const [logs, setLogs] = useState<LogEntryItem[]>([])
  const [wrap, setWrap] = useState(false)
  const [splitMode, setSplitMode] = useState(true)
  const [sendRange, setSendRange] = useState<SendRange>(null)
  const [anchorIdx, setAnchorIdx] = useState<number | null>(null)
  const logBottomRef  = useRef<HTMLDivElement>(null)
  const splitBottomRef = useRef<HTMLDivElement>(null)
  const selectedIDRef = useRef<string | null>(null)

  useEffect(() => { selectedIDRef.current = selectedID }, [selectedID])
  useEffect(() => { setSendRange(null); setAnchorIdx(null) }, [selectedID, splitMode])

  // サーバ状態の取得・購読
  useEffect(() => {
    ServerService.GetState().then((s: any) => { if (s) setServerState(s) }).catch(() => {})
    const unsub = Events.On('server-state', (e: any) => {
      const s = e.data
      if (s) setServerState(s)
    })
    return () => { unsub() }
  }, [])

  const loadConnections = useCallback(() => {
    DebugService.ListConnections()
      .then((items: any) => {
        const fresh: ConnectionInfo[] = (items ?? []).map((c: any) => ({ ...c, active: true }))
        setConnections(fresh)
        if (selectedIDRef.current && !fresh.find(c => c.id === selectedIDRef.current)) {
          setSelectedID(null); setLogs([])
        }
      })
      .catch(() => {})
  }, [])

  const loadLogs = useCallback((id: string) => {
    DebugService.GetLogs(id)
      .then((items: any) => setLogs(items ?? []))
      .catch(() => {})
  }, [])

  useEffect(() => {
    loadConnections()
    const unsubAdded = Events.On('conn-added', (e: any) => {
      const conn: ConnectionInfo = e.data
      if (!conn?.id) return
      setConnections(prev => prev.find(c => c.id === conn.id) ? prev : [...prev, { ...conn, active: true }])
    })
    const unsubRemoved = Events.On('conn-removed', (e: any) => {
      const id: string = e.data
      if (!id) return
      setConnections(prev => prev.map(c => c.id === id ? { ...c, active: false } : c))
    })
    const unsubLog = Events.On('usi-log', (e: any) => {
      const payload = e.data
      if (!payload?.connId) return
      if (selectedIDRef.current === payload.connId) {
        const r = payload.entry
        setLogs(prev => [...prev, { timestamp: r?.timestamp ?? null, dir: r?.dir ?? 0, message: r?.message ?? '' }])
      }
    })
    return () => { unsubAdded(); unsubRemoved(); unsubLog() }
  }, [loadConnections])

  useEffect(() => {
    if (selectedID) { setLogs([]); loadLogs(selectedID) } else { setLogs([]) }
  }, [selectedID, loadLogs])

  useEffect(() => {
    logBottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    splitBottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  const sendLogs = useMemo(() => logs.filter(e => e.dir === 0), [logs])

  // 送信クリックハンドラ（時系列・分割共通）
  const handleSendClick = useCallback((i: number, shift: boolean) => {
    if (shift && anchorIdx !== null) {
      // Shift: アンカー〜クリック位置の範囲
      setSendRange({ from: Math.min(anchorIdx, i), to: Math.max(anchorIdx, i) })
    } else {
      // 通常: 同一行クリックで解除、それ以外は単一選択しアンカー更新
      setSendRange(prev => (prev?.from === i && prev?.to === i) ? null : { from: i, to: i })
      setAnchorIdx(prev => (prev === i && sendRange?.from === i && sendRange?.to === i) ? null : i)
    }
  }, [anchorIdx, sendRange])

  // 受信ハイライト窓
  const highlightWin: HighlightWin = useMemo(() => {
    if (!sendRange) return null
    return {
      from: toMs(sendLogs[sendRange.from]?.timestamp),
      to: sendLogs[sendRange.to + 1] ? toMs(sendLogs[sendRange.to + 1].timestamp) : Infinity,
    }
  }, [sendRange, sendLogs])

  // 時系列ノード（送信連番を振りながら生成）
  const timelineNodes = useMemo(() => {
    let si = 0
    return logs.map((entry, i) => {
      const idx = entry.dir === 0 ? si++ : -1
      const hl: HL = entry.dir === 0 ? sendHL(sendRange, idx) : recvHL(highlightWin, entry.timestamp)
      return (
        <TimelineLine
          key={i} entry={entry} wrap={wrap} hl={hl}
          onClick={entry.dir === 0 ? (e) => handleSendClick(idx, e.shiftKey) : undefined}
        />
      )
    })
  }, [logs, wrap, sendRange, highlightWin, handleSendClick])

  const selectedConn = connections.find(c => c.id === selectedID)

  const rangeLabel = sendRange
    ? sendRange.from === sendRange.to
      ? `送信 #${sendRange.from + 1}`
      : `送信 #${sendRange.from + 1}〜#${sendRange.to + 1}`
    : null

  return (
    <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
      {/* 接続一覧 */}
      <Box sx={{ width: 260, flexShrink: 0, borderRight: '1px solid', borderColor: 'divider', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <Box sx={{ px: 1.5, py: 0.75, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Chip
              label={serverState.running ? '起動中' : '停止中'}
              color={serverState.running ? 'success' : 'default'}
              size="small"
              variant={serverState.running ? 'filled' : 'outlined'}
              sx={{ height: 18, fontSize: '0.65rem' }}
            />
            {serverState.running && serverState.url && (
              <Typography sx={{ fontFamily: 'monospace', fontSize: '0.7rem', color: 'success.light', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
                {serverState.url}
              </Typography>
            )}
            <Tooltip title="一覧を更新（切断済みを削除）">
              <IconButton size="small" onClick={loadConnections} sx={{ color: 'text.disabled', '&:hover': { color: 'text.primary' }, ml: 'auto' }}>
                <RefreshIcon sx={{ fontSize: 14 }} />
              </IconButton>
            </Tooltip>
          </Box>
          <Typography variant="caption" sx={{ color: 'text.secondary', fontWeight: 600, letterSpacing: 0.5 }}>
            接続中 ({connections.filter(c => c.active !== false).length})
          </Typography>
        </Box>
        <Divider />
        <List dense disablePadding sx={{ flex: 1, overflow: 'auto' }}>
          {connections.length === 0 ? (
            <Box sx={{ px: 2, py: 4, color: 'text.disabled', fontSize: '0.8rem', textAlign: 'center' }}>接続なし</Box>
          ) : connections.map(c => {
            const disconnected = c.active === false
            return (
              <ListItemButton key={c.id} selected={selectedID === c.id} onClick={() => setSelectedID(c.id)} sx={{ px: 2, py: 1, opacity: disconnected ? 0.45 : 1 }}>
                <ListItemText
                  primary={
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                      <Typography sx={{ fontFamily: 'monospace', fontSize: '0.7rem', color: disconnected ? 'text.disabled' : 'primary.light' }}>
                        {c.id.slice(0, 8)}…
                      </Typography>
                      {disconnected && (
                        <Typography sx={{ fontSize: '0.6rem', color: 'error.dark', border: '1px solid', borderColor: 'error.dark', borderRadius: '2px', px: 0.4, lineHeight: 1.4 }}>
                          切断
                        </Typography>
                      )}
                    </Box>
                  }
                  secondary={
                    <Typography sx={{ fontSize: '0.7rem', color: 'text.secondary', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {String(c.enginePath).split(/[\\/]/).pop()}
                    </Typography>
                  }
                />
              </ListItemButton>
            )
          })}
        </List>
      </Box>

      {/* ログエリア */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* ツールバー */}
        <Box sx={{ px: 2, py: 0.5, borderBottom: '1px solid', borderColor: 'divider', flexShrink: 0, display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box sx={{ flex: 1, overflow: 'hidden' }}>
            {selectedConn ? (
              <Typography variant="caption" sx={{ fontFamily: 'monospace', color: 'text.secondary', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>
                {selectedConn.id} &nbsp;|&nbsp; {selectedConn.enginePath}
              </Typography>
            ) : (
              <Typography variant="caption" sx={{ color: 'text.disabled' }}>接続を選択してください</Typography>
            )}
          </Box>
          {rangeLabel && (
            <Tooltip title="フィルタ解除">
              <Typography
                variant="caption"
                onClick={() => { setSendRange(null); setAnchorIdx(null) }}
                sx={{ color: 'warning.main', cursor: 'pointer', whiteSpace: 'nowrap', flexShrink: 0, '&:hover': { textDecoration: 'underline' } }}
              >
                {rangeLabel} ×
              </Typography>
            </Tooltip>
          )}
          <Tooltip title={splitMode ? '時系列表示' : '送受信分割表示'}>
            <IconButton size="small" onClick={() => setSplitMode(m => !m)} sx={{ color: splitMode ? 'primary.main' : 'text.disabled' }}>
              {splitMode ? <ViewStreamIcon sx={{ fontSize: 16 }} /> : <ViewColumnIcon sx={{ fontSize: 16 }} />}
            </IconButton>
          </Tooltip>
          <Tooltip title={wrap ? '折り返しオフ' : '折り返しオン'}>
            <IconButton size="small" onClick={() => setWrap(w => !w)} sx={{ color: wrap ? 'primary.main' : 'text.disabled' }}>
              <WrapTextIcon sx={{ fontSize: 16 }} />
            </IconButton>
          </Tooltip>
        </Box>

        {!splitMode && (
          <Box sx={{ flex: 1, overflow: 'auto', px: 2, py: 1, fontFamily: 'monospace', fontSize: '0.75rem', lineHeight: 1.6 }}>
            {logs.length === 0 && selectedID && <Box sx={{ color: 'text.disabled', mt: 2 }}>ログなし</Box>}
            {timelineNodes}
            <div ref={logBottomRef} />
          </Box>
        )}

        {splitMode && (
          <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
            <SplitPane logs={logs} dir={0} wrap={wrap} sendRange={sendRange} onSendClick={handleSendClick} />
            <Divider orientation="vertical" flexItem />
            <SplitPane logs={logs} dir={1} wrap={wrap} bottomRef={splitBottomRef} highlightWin={highlightWin} />
          </Box>
        )}
      </Box>
    </Box>
  )
}

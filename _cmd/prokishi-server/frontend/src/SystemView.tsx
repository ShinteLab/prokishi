import { useState, useEffect, useRef, useCallback } from 'react'
import { Box, Typography, LinearProgress, Chip } from '@mui/material'
import { SystemService } from '../bindings/prokishi-server'

type ProcessCpuInfo = {
  connId: string
  engineName: string
  pid: number
  cpuPercent: number
}

type CpuInfo = {
  total: number
  perCpu: number[]
  modelName: string
  cores: number
}

type SystemInfo = {
  cpu: CpuInfo
  processes: ProcessCpuInfo[]
}

type Snapshot = {
  time: string
  total: number
  processes: Record<string, number>
}

const MAX_HISTORY = 60
const INTERVAL_MS = 2000

const COLORS = ['#4caf50', '#2196f3', '#ff9800', '#e91e63', '#9c27b0', '#00bcd4', '#cddc39', '#ff5722']

function cpuColor(pct: number): string {
  if (pct >= 90) return '#f44336'
  if (pct >= 70) return '#ff9800'
  if (pct >= 50) return '#ffeb3b'
  return '#4caf50'
}

function LineGraph({ history, processKeys }: { history: Snapshot[]; processKeys: { key: string; label: string }[] }) {
  const w = 500
  const h = 120
  if (history.length < 2) return null

  const allKeys = [{ key: '__total__', label: 'CPU 全体' }, ...processKeys]

  const buildPoints = (key: string) =>
    history.map((s, i) => {
      const x = (i / (MAX_HISTORY - 1)) * w
      const val = key === '__total__' ? s.total : (s.processes[key] ?? 0)
      const y = h - (Math.min(val, 100) / 100) * h
      return { x, y }
    })

  return (
    <Box>
      <svg width="100%" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" style={{ display: 'block' }}>
        {[25, 50, 75].map(v => (
          <line key={v} x1={0} y1={h - (v / 100) * h} x2={w} y2={h - (v / 100) * h}
            stroke="rgba(255,255,255,0.08)" strokeWidth="0.5" />
        ))}
        {allKeys.map((k, ci) => {
          const pts = buildPoints(k.key)
          const polyline = pts.map(p => `${p.x},${p.y}`).join(' ')
          const color = ci === 0 ? 'rgba(255,255,255,0.35)' : COLORS[(ci - 1) % COLORS.length]
          const sw = ci === 0 ? 1 : 1.5
          return <polyline key={k.key} points={polyline} fill="none" stroke={color} strokeWidth={sw} />
        })}
      </svg>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1.5, mt: 0.5 }}>
        {allKeys.map((k, ci) => {
          const color = ci === 0 ? 'rgba(255,255,255,0.5)' : COLORS[(ci - 1) % COLORS.length]
          const last = history[history.length - 1]
          const val = k.key === '__total__' ? last.total : (last.processes[k.key] ?? 0)
          return (
            <Box key={k.key} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <Box sx={{ width: 10, height: 3, bgcolor: color, borderRadius: 1 }} />
              <Typography sx={{ fontSize: '0.6rem', color: 'text.secondary' }}>
                {k.label}
              </Typography>
              <Typography sx={{ fontSize: '0.6rem', fontFamily: 'monospace', color }}>
                {val.toFixed(1)}%
              </Typography>
            </Box>
          )
        })}
      </Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 0.25 }}>
        <Typography sx={{ fontSize: '0.55rem', color: 'text.disabled' }}>
          {history[0].time}
        </Typography>
        <Typography sx={{ fontSize: '0.55rem', color: 'text.disabled' }}>
          {history[history.length - 1].time}
        </Typography>
      </Box>
    </Box>
  )
}

export function SystemView() {
  const [info, setInfo] = useState<SystemInfo | null>(null)
  const [history, setHistory] = useState<Snapshot[]>([])
  const [processKeys, setProcessKeys] = useState<{ key: string; label: string }[]>([])
  const [error, setError] = useState<string | null>(null)
  const timerRef = useRef<number | null>(null)

  const fetchInfo = useCallback(() => {
    SystemService.GetSystemInfo()
      .then((data: any) => {
        if (!data) return
        setInfo(data)
        setError(null)

        const now = new Date().toLocaleTimeString('ja-JP', { hour12: false })
        const procMap: Record<string, number> = {}
        const keys: { key: string; label: string }[] = []
        if (data.processes) {
          for (const p of data.processes) {
            procMap[p.connId] = p.cpuPercent
            keys.push({ key: p.connId, label: `${p.engineName} (PID:${p.pid})` })
          }
        }
        setProcessKeys(keys)

        setHistory(prev => {
          const next = [...prev, { time: now, total: data.cpu.total, processes: procMap }]
          return next.length > MAX_HISTORY ? next.slice(-MAX_HISTORY) : next
        })
      })
      .catch((e: any) => setError(String(e)))
  }, [])

  useEffect(() => {
    fetchInfo()
    timerRef.current = window.setInterval(fetchInfo, INTERVAL_MS)
    return () => {
      if (timerRef.current !== null) clearInterval(timerRef.current)
    }
  }, [fetchInfo])

  const cpu = info?.cpu

  return (
    <Box sx={{ flex: 1, overflow: 'auto', p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>

      {error && (
        <Typography color="error" sx={{ fontSize: '0.8rem' }}>{error}</Typography>
      )}

      {/* CPU 情報ヘッダ */}
      {cpu && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Typography sx={{ fontSize: '0.8rem', color: 'text.secondary' }}>
            {cpu.modelName || 'CPU'}
          </Typography>
          <Chip label={`${cpu.cores} コア`} size="small" variant="outlined"
            sx={{ height: 20, fontSize: '0.65rem' }} />
        </Box>
      )}

      {/* 全体使用率 */}
      {cpu && (
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1, mb: 0.5 }}>
            <Typography sx={{ fontSize: '0.75rem', color: 'text.secondary' }}>CPU 使用率</Typography>
            <Typography sx={{ fontSize: '1.5rem', fontWeight: 700, fontFamily: 'monospace', color: cpuColor(cpu.total) }}>
              {cpu.total.toFixed(1)}%
            </Typography>
          </Box>
          <LinearProgress
            variant="determinate"
            value={cpu.total}
            sx={{
              height: 8, borderRadius: 4,
              bgcolor: 'rgba(255,255,255,0.08)',
              '& .MuiLinearProgress-bar': { bgcolor: cpuColor(cpu.total), borderRadius: 4 },
            }}
          />
        </Box>
      )}

      {/* 折れ線グラフ */}
      <Box sx={{
        bgcolor: 'rgba(255,255,255,0.03)',
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: 1,
        p: 1,
      }}>
        <Typography sx={{ fontSize: '0.65rem', color: 'text.disabled', mb: 0.5 }}>
          CPU 使用率 推移（直近 {MAX_HISTORY * INTERVAL_MS / 1000} 秒）
        </Typography>
        <LineGraph history={history} processKeys={processKeys} />
      </Box>

      {/* エンジンプロセス一覧 */}
      {info?.processes && info.processes.length > 0 && (
        <Box>
          <Typography sx={{ fontSize: '0.75rem', color: 'text.secondary', mb: 1 }}>
            エンジンプロセス
          </Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
            {info.processes.map((p, i) => (
              <Box key={p.connId} sx={{
                display: 'flex', alignItems: 'center', gap: 1.5,
                bgcolor: 'rgba(255,255,255,0.03)',
                border: '1px solid', borderColor: 'divider',
                borderRadius: 1, px: 1.5, py: 0.75,
              }}>
                <Box sx={{ width: 10, height: 10, borderRadius: '50%', bgcolor: COLORS[i % COLORS.length], flexShrink: 0 }} />
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <Typography sx={{ fontSize: '0.75rem', color: 'text.primary', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {p.engineName}
                  </Typography>
                  <Typography sx={{ fontSize: '0.6rem', color: 'text.disabled', fontFamily: 'monospace' }}>
                    PID: {p.pid} &nbsp;|&nbsp; 接続: {p.connId.slice(0, 8)}…
                  </Typography>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexShrink: 0 }}>
                  <LinearProgress
                    variant="determinate"
                    value={Math.min(p.cpuPercent, 100)}
                    sx={{
                      width: 80, height: 6, borderRadius: 3,
                      bgcolor: 'rgba(255,255,255,0.08)',
                      '& .MuiLinearProgress-bar': { bgcolor: COLORS[i % COLORS.length], borderRadius: 3 },
                    }}
                  />
                  <Typography sx={{ fontSize: '0.75rem', fontFamily: 'monospace', fontWeight: 600, color: COLORS[i % COLORS.length], width: 50, textAlign: 'right' }}>
                    {p.cpuPercent.toFixed(1)}%
                  </Typography>
                </Box>
              </Box>
            ))}
          </Box>
        </Box>
      )}

      {/* コア別使用率 */}
      {cpu && cpu.perCpu.length > 0 && (
        <Box>
          <Typography sx={{ fontSize: '0.75rem', color: 'text.secondary', mb: 1 }}>
            コア別使用率
          </Typography>
          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 1 }}>
            {cpu.perCpu.map((pct, i) => (
              <Box key={i} sx={{
                display: 'flex', alignItems: 'center', gap: 1,
                bgcolor: 'rgba(255,255,255,0.03)',
                border: '1px solid', borderColor: 'divider',
                borderRadius: 1, px: 1, py: 0.5,
              }}>
                <Typography sx={{ fontSize: '0.65rem', color: 'text.disabled', width: 28, flexShrink: 0 }}>
                  #{i}
                </Typography>
                <LinearProgress
                  variant="determinate"
                  value={pct}
                  sx={{
                    flex: 1, height: 6, borderRadius: 3,
                    bgcolor: 'rgba(255,255,255,0.08)',
                    '& .MuiLinearProgress-bar': { bgcolor: cpuColor(pct), borderRadius: 3 },
                  }}
                />
                <Typography sx={{ fontSize: '0.65rem', fontFamily: 'monospace', color: cpuColor(pct), width: 40, textAlign: 'right', flexShrink: 0 }}>
                  {pct.toFixed(0)}%
                </Typography>
              </Box>
            ))}
          </Box>
        </Box>
      )}
    </Box>
  )
}

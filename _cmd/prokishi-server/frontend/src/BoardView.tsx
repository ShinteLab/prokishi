import { useState, useEffect, useCallback } from 'react'
import {
  Box, Button, Chip, Divider, MenuItem, Select, TextField, Typography,
} from '@mui/material'
import { resolvePosition } from '@shinte/web'
import { DebugService } from '../bindings/prokishi-server'

const STARTPOS_BOARD = 'lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL'

type ConnectionInfo = { id: string; engineName?: string; enginePath?: string }
type LogEntryItem = { dir: number; message: string }

export function BoardView() {
  const [board, setBoard] = useState(STARTPOS_BOARD)
  const [lastMove, setLastMove] = useState('')
  const [hands, setHands] = useState('-')
  const [turn, setTurn] = useState<'b' | 'w'>('b')
  const [flip, setFlip] = useState(false)
  const [connections, setConnections] = useState<ConnectionInfo[]>([])
  const [selectedID, setSelectedID] = useState('')
  const [info, setInfo] = useState<{ moveCount: number; raw: string } | null>(null)

  const loadConnections = useCallback(() => {
    DebugService.ListConnections()
      .then((items: any) => setConnections((items ?? []) as ConnectionInfo[]))
      .catch(() => {})
  }, [])

  useEffect(() => { loadConnections() }, [loadConnections])

  // 選択中の接続の直近 position コマンドを解決し、手を適用した現在局面を表示する。
  const pullFromConnection = useCallback(() => {
    if (!selectedID) return
    DebugService.GetLogs(selectedID)
      .then((items: any) => {
        const logs = (items ?? []) as LogEntryItem[]
        for (let i = logs.length - 1; i >= 0; i--) {
          if (logs[i].dir !== 0) continue
          const p = resolvePosition(logs[i].message)
          if (p) {
            setBoard(p.board)
            setLastMove(p.lastMove)
            setHands(p.hands)
            setTurn(p.turn)
            setInfo({ moveCount: p.moveCount, raw: logs[i].message })
            return
          }
        }
        setInfo({ moveCount: 0, raw: '(position コマンドが見つかりません)' })
      })
      .catch(() => {})
  }, [selectedID])

  return (
    <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
      {/* 左: 盤 + 持ち駒 */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1, overflow: 'auto', p: 3 }}>
        <Box sx={{ alignSelf: 'flex-start', display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography variant="caption" sx={{ color: 'text.secondary' }}>
            {flip ? '☗ 先手' : '☖ 後手'} 持駒:
          </Typography>
          <shogi-hand hands={hands} side={flip ? 'b' : 'w'} flip={flip || undefined} />
        </Box>

        <Box sx={{ maxWidth: 560, width: '100%' }}>
          <shogi-board sfen={board} last-move={lastMove || undefined} flip={flip || undefined} />
        </Box>

        <Box sx={{ alignSelf: 'flex-end', display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography variant="caption" sx={{ color: 'text.secondary' }}>
            {flip ? '☖ 後手' : '☗ 先手'} 持駒:
          </Typography>
          <shogi-hand hands={hands} side={flip ? 'w' : 'b'} flip={flip || undefined} />
        </Box>
      </Box>

      {/* 右: 操作パネル */}
      <Box sx={{ width: 320, flexShrink: 0, borderLeft: '1px solid', borderColor: 'divider', display: 'flex', flexDirection: 'column', gap: 2, p: 2, overflow: 'auto' }}>
        <Typography variant="subtitle2" sx={{ color: 'text.secondary' }}>盤面表示</Typography>

        <TextField
          label="SFEN (盤面部分または完全SFEN)"
          value={board}
          onChange={(e) => setBoard(e.target.value.trim())}
          size="small"
          multiline
          minRows={2}
          slotProps={{ input: { sx: { fontFamily: 'monospace', fontSize: '0.75rem' } } }}
        />
        <TextField
          label="直前手 (USI, 例 7g7f)"
          value={lastMove}
          onChange={(e) => setLastMove(e.target.value.trim())}
          size="small"
          slotProps={{ input: { sx: { fontFamily: 'monospace', fontSize: '0.8rem' } } }}
        />
        <Box>
          <Button size="small" variant="outlined" onClick={() => setFlip(f => !f)}>
            {flip ? '先手視点に戻す' : '後手視点(flip)'}
          </Button>
        </Box>

        <Divider />

        <Typography variant="subtitle2" sx={{ color: 'text.secondary' }}>接続から取り込み</Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Select
            size="small"
            value={selectedID}
            displayEmpty
            onChange={(e) => setSelectedID(e.target.value)}
            onOpen={loadConnections}
            sx={{ flex: 1, fontSize: '0.8rem' }}
          >
            <MenuItem value=""><em>接続を選択</em></MenuItem>
            {connections.map(c => (
              <MenuItem key={c.id} value={c.id} sx={{ fontSize: '0.8rem' }}>
                {c.id.slice(0, 8)}… {c.engineName || String(c.enginePath ?? '').split(/[\\/]/).pop()}
              </MenuItem>
            ))}
          </Select>
          <Button size="small" variant="contained" disabled={!selectedID} onClick={pullFromConnection}>
            取得
          </Button>
        </Box>

        {info && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
              <Chip size="small" label={`手数 ${info.moveCount}`} sx={{ height: 20, fontSize: '0.65rem' }} />
              <Chip size="small" label={`手番 ${turn === 'b' ? '先手' : '後手'}`} sx={{ height: 20, fontSize: '0.65rem' }} />
              {lastMove && <Chip size="small" color="warning" label={`直前 ${lastMove}`} sx={{ height: 20, fontSize: '0.65rem' }} />}
            </Box>
            <Typography sx={{ fontFamily: 'monospace', fontSize: '0.68rem', color: 'text.disabled', wordBreak: 'break-all' }}>
              {info.raw}
            </Typography>
          </Box>
        )}

        <Typography variant="caption" sx={{ color: 'text.disabled', mt: 'auto' }}>
          ※ position コマンドの moves を適用した現在局面を表示します(捕獲・成り・打ちを反映)。
          手の適用は表示用で、合法性の判定は行いません。
        </Typography>
      </Box>
    </Box>
  )
}

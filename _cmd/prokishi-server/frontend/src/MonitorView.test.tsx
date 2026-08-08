import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MonitorView } from './MonitorView'
import { Events } from '@wailsio/runtime'
import { DebugService, ServerService } from '../bindings/prokishi-server'

vi.mock('@wailsio/runtime', () => ({
  Events: { On: vi.fn(() => () => {}) },
}))

vi.mock('../bindings/wails', () => ({
  DebugService: {
    ListConnections: vi.fn(),
    GetLogs: vi.fn(),
  },
  ServerService: {
    GetState: vi.fn(),
  },
}))

function getEventHandler(event: string): (e: any) => void {
  const call = vi
    .mocked(Events.On)
    .mock.calls.reverse()
    .find(([name]) => name === event)
  if (!call) throw new Error(`no Events.On registration found for "${event}"`)
  return call[1] as (e: any) => void
}

const conn1 = {
  id: 'conn-aaaaaaaa-1111-1111-1111-111111111111',
  engineId: 'engine-1',
  engineName: 'MyEngine',
  enginePath: 'C:\\engines\\myengine.exe',
  connectedAt: '2024-01-01T00:00:00.000Z',
}

describe('MonitorView', () => {
  beforeEach(() => {
    vi.mocked(Events.On).mockClear()
    vi.mocked(DebugService.ListConnections).mockReset().mockResolvedValue([])
    vi.mocked(DebugService.GetLogs).mockReset().mockResolvedValue([])
    vi.mocked(ServerService.GetState).mockReset().mockResolvedValue({ running: false, url: '', err: '' })
  })

  it('renders the connection list from ListConnections', async () => {
    vi.mocked(DebugService.ListConnections).mockResolvedValue([conn1])

    render(<MonitorView />)

    expect(await screen.findByText('MyEngine')).toBeInTheDocument()
    expect(screen.getByText('接続中 (1)')).toBeInTheDocument()
  })

  it('loads logs for the clicked connection and renders them in the default split-pane view', async () => {
    vi.mocked(DebugService.ListConnections).mockResolvedValue([conn1])
    vi.mocked(DebugService.GetLogs).mockResolvedValue([
      { timestamp: '2024-01-01T00:00:01.000Z', dir: 0, message: 'usinewgame' },
      { timestamp: '2024-01-01T00:00:02.000Z', dir: 1, message: 'readyok' },
    ])

    render(<MonitorView />)

    const user = userEvent.setup()
    await user.click(await screen.findByText('MyEngine'))

    expect(DebugService.GetLogs).toHaveBeenCalledWith(conn1.id)

    // default splitMode is `true` (useState(true) in MonitorView), so both
    // the send and receive panes should be visible simultaneously.
    expect(await screen.findByText('送信 (client → engine)')).toBeInTheDocument()
    expect(screen.getByText('受信 (engine → client)')).toBeInTheDocument()
    expect(await screen.findByText('usinewgame')).toBeInTheDocument()
    expect(screen.getByText('readyok')).toBeInTheDocument()
  })

  it('appends a new log line when usi-log fires for the currently selected connection', async () => {
    vi.mocked(DebugService.ListConnections).mockResolvedValue([conn1])
    vi.mocked(DebugService.GetLogs).mockResolvedValue([
      { timestamp: '2024-01-01T00:00:01.000Z', dir: 0, message: 'usinewgame' },
    ])

    render(<MonitorView />)

    const user = userEvent.setup()
    await user.click(await screen.findByText('MyEngine'))
    expect(await screen.findByText('usinewgame')).toBeInTheDocument()

    const usiLogHandler = getEventHandler('usi-log')
    act(() => {
      usiLogHandler({
        data: {
          connId: conn1.id,
          entry: { timestamp: '2024-01-01T00:00:05.000Z', dir: 0, message: 'go infinite' },
        },
      })
    })

    expect(await screen.findByText('go infinite')).toBeInTheDocument()
  })

  it('marks a connection as disconnected on conn-removed without removing it from the list', async () => {
    vi.mocked(DebugService.ListConnections).mockResolvedValue([conn1])

    render(<MonitorView />)

    expect(await screen.findByText('MyEngine')).toBeInTheDocument()
    expect(screen.queryByText('切断')).not.toBeInTheDocument()

    const connRemovedHandler = getEventHandler('conn-removed')
    act(() => { connRemovedHandler({ data: conn1.id }) })

    await waitFor(() => {
      expect(screen.getByText('切断')).toBeInTheDocument()
    })
    // Still present in the rendered list, just visually marked as disconnected.
    expect(screen.getByText('MyEngine')).toBeInTheDocument()
  })
})

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'
import { Events } from '@wailsio/runtime'
import { WindowService } from '../bindings/prokishi-server'

// App.tsx subscribes to the 'request-close' Wails event (emitted by the Go
// side in response to WindowClosing) and shows a confirmation dialog.
// MonitorView (rendered as the default view) also runs on mount, so its
// bindings need mocking too or the component tree throws during render.
// @wailsio/runtime は setupTests.ts の automock（__mocks__/@wailsio/runtime.ts）に任せる。
// ここでファクトリを書くと bindings が使う Create エクスポートが欠けて読み込みに失敗する。

vi.mock('../bindings/prokishi-server', () => ({
  WindowService: { ShutdownAndQuit: vi.fn() },
  DebugService: {
    ListConnections: vi.fn().mockResolvedValue([]),
    GetLogs: vi.fn().mockResolvedValue([]),
  },
  ServerService: { GetState: vi.fn().mockResolvedValue({ running: false, url: '' }) },
}))

function getEventHandler(event: string): (e?: any) => void {
  const call = vi.mocked(Events.On).mock.calls.find(([name]) => name === event)
  if (!call) throw new Error(`no Events.On registration found for "${event}"`)
  return call[1] as (e?: any) => void
}

describe('App close confirmation flow', () => {
  beforeEach(() => {
    vi.mocked(Events.On).mockClear()
    vi.mocked(WindowService.ShutdownAndQuit).mockClear()
  })

  it('shows the confirmation dialog when request-close fires, and quits when 終了 is clicked', async () => {
    render(<App />)

    const requestClose = getEventHandler('request-close')
    act(() => { requestClose() })

    expect(await screen.findByText('終了しますか？')).toBeInTheDocument()

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: '終了' }))

    expect(WindowService.ShutdownAndQuit).toHaveBeenCalledTimes(1)
    await waitFor(() => {
      expect(screen.queryByText('終了しますか？')).not.toBeInTheDocument()
    })
  })

  it('closes the dialog without quitting when キャンセル is clicked', async () => {
    render(<App />)

    const requestClose = getEventHandler('request-close')
    act(() => { requestClose() })

    expect(await screen.findByText('終了しますか？')).toBeInTheDocument()

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'キャンセル' }))

    expect(WindowService.ShutdownAndQuit).not.toHaveBeenCalled()
    await waitFor(() => {
      expect(screen.queryByText('終了しますか？')).not.toBeInTheDocument()
    })
  })
})

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import App from '../App'

vi.mock('../../bindings/wails', () => ({
  WindowService: {
    ShutdownAndQuit: vi.fn(),
  },
  ServerService: {
    GetState: vi.fn(() => Promise.resolve({ running: false, address: '' })),
    GetConfig: vi.fn(() => Promise.resolve({
      port: 8080, host: '', auth: false, autoStart: false,
    })),
    GetLocalIPs: vi.fn(() => Promise.resolve([])),
    IsRunning: vi.fn(() => Promise.resolve(false)),
  },
  AdminService: {
    GetEngines: vi.fn(() => Promise.resolve([])),
    GetCodes: vi.fn(() => Promise.resolve([])),
  },
  DebugService: {
    ListConnections: vi.fn(() => Promise.resolve([])),
    GetLogs: vi.fn(() => Promise.resolve([])),
  },
  SystemService: {
    GetSystemInfo: vi.fn(() => Promise.resolve(null)),
  },
}))

describe('App', () => {
  it('renders without crashing', () => {
    render(<App />)
    expect(screen.getByText('prokishi-server')).toBeDefined()
  })

  it('shows close confirmation dialog on close button click', () => {
    render(<App />)

    const closeButton = screen.getByTestId('CloseIcon').closest('button')!
    fireEvent.click(closeButton)

    expect(screen.getByText('終了しますか？')).toBeDefined()
    expect(screen.getByText('サーバを停止してアプリケーションを終了します。')).toBeDefined()
  })

  it('dismisses close dialog on cancel', async () => {
    render(<App />)

    const closeButton = screen.getByTestId('CloseIcon').closest('button')!
    fireEvent.click(closeButton)

    fireEvent.click(screen.getByText('キャンセル'))

    await waitFor(() => {
      expect(screen.queryByText('終了しますか？')).toBeNull()
    })
  })
})

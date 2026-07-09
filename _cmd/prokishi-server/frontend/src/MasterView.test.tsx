import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MasterView } from './MasterView'
import { Events } from '@wailsio/runtime'
import { AdminService, ServerService } from '../bindings/wails'

// MasterView subscribes to the 'server-state' event to keep the running/
// stopped chip in sync with the Go side.
vi.mock('@wailsio/runtime', () => ({
  Events: { On: vi.fn(() => () => {}) },
}))

vi.mock('../bindings/wails', () => ({
  AdminService: {
    ListEngines: vi.fn(),
    ListCodes: vi.fn(),
    GenerateCode: vi.fn(),
    RegisterCode: vi.fn(),
    RegisterEngine: vi.fn(),
    DeleteEngine: vi.fn(),
    DeleteCode: vi.fn(),
    UpdateEngineName: vi.fn(),
    UpdateCodeName: vi.fn(),
    DisableCode: vi.fn(),
    EnableCode: vi.fn(),
    SelectEnginePath: vi.fn(),
    SaveClientConfig: vi.fn(),
  },
  ServerService: {
    GetState: vi.fn(),
    GetConfig: vi.fn(),
    GetLocalIPs: vi.fn(),
    SaveConfig: vi.fn(),
    Start: vi.fn(),
    Stop: vi.fn(),
  },
}))

const engine1 = { id: 'engine-aaaa-1111', name: 'Apery', path: 'C:\\engines\\apery.exe', created: '2024-01-01 00:00:00' }
const engine2 = { id: 'engine-bbbb-2222', name: 'YaneuraOu', path: 'C:\\engines\\yaneuraou.exe', created: '2024-01-02 00:00:00' }
const code1 = { code: 'ABC123', name: 'Yamada', created: '2024-01-01 00:00:00', used: '', disabled: false }
const code2 = { code: 'XYZ789', name: '', created: '2024-01-03 00:00:00', used: '', disabled: false }

describe('MasterView', () => {
  beforeEach(() => {
    vi.mocked(Events.On).mockClear()

    vi.mocked(AdminService.ListEngines).mockReset().mockResolvedValue([engine1])
    vi.mocked(AdminService.ListCodes).mockReset().mockResolvedValue([code1])
    vi.mocked(AdminService.GenerateCode).mockReset().mockResolvedValue('XYZ789')
    vi.mocked(AdminService.RegisterCode).mockReset().mockResolvedValue(undefined)
    vi.mocked(AdminService.RegisterEngine).mockReset().mockResolvedValue('engine-bbbb-2222')
    vi.mocked(AdminService.DeleteEngine).mockReset().mockResolvedValue(undefined)
    vi.mocked(AdminService.DeleteCode).mockReset().mockResolvedValue(undefined)
    vi.mocked(AdminService.UpdateEngineName).mockReset().mockResolvedValue(undefined)
    vi.mocked(AdminService.UpdateCodeName).mockReset().mockResolvedValue(undefined)
    vi.mocked(AdminService.DisableCode).mockReset().mockResolvedValue(undefined)
    vi.mocked(AdminService.EnableCode).mockReset().mockResolvedValue(undefined)
    vi.mocked(AdminService.SelectEnginePath).mockReset().mockResolvedValue('')
    vi.mocked(AdminService.SaveClientConfig).mockReset().mockResolvedValue(undefined)

    vi.mocked(ServerService.GetState).mockReset().mockResolvedValue({ running: false, url: '' })
    vi.mocked(ServerService.GetConfig).mockReset().mockResolvedValue({ host: '', port: 8080, autoStart: true })
    vi.mocked(ServerService.GetLocalIPs).mockReset().mockResolvedValue(['192.168.1.10'])
    vi.mocked(ServerService.SaveConfig).mockReset().mockResolvedValue(undefined)
    vi.mocked(ServerService.Start).mockReset().mockResolvedValue(undefined)
    vi.mocked(ServerService.Stop).mockReset().mockResolvedValue(undefined)
  })

  it('renders without crashing and shows expected content across all four tabs', async () => {
    render(<MasterView />)

    // サーバ設定 (default tab, index 0)
    expect(await screen.findByText('停止中')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '起動' })).toBeInTheDocument()

    const user = userEvent.setup()

    await user.click(screen.getByRole('tab', { name: 'エンジン管理' }))
    expect(await screen.findByText('Apery')).toBeInTheDocument()
    expect(screen.getByText('C:\\engines\\apery.exe')).toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: '認証コード管理' }))
    expect(await screen.findByText('Yamada')).toBeInTheDocument()
    expect(screen.getByText('ABC123')).toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: 'クライアント設定' }))
    expect(await screen.findByText('prokishi.ini プレビュー')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'prokishi.ini を保存' })).toBeDisabled()
  })

  // Representative interaction 1: a simple one-click action (自動生成) with no
  // required input, whose effect (a fresh entry appearing after the list
  // reloads) is unambiguous to assert on.
  it('generates an auth code and refreshes the code list', async () => {
    vi.mocked(AdminService.ListCodes)
      .mockReset()
      .mockResolvedValueOnce([code1])
      .mockResolvedValueOnce([code1, code2])

    render(<MasterView />)

    const user = userEvent.setup()
    await user.click(screen.getByRole('tab', { name: '認証コード管理' }))
    expect(await screen.findByText('Yamada')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '自動生成' }))

    expect(AdminService.GenerateCode).toHaveBeenCalledWith('')
    expect(await screen.findByText('生成しました: XYZ789')).toBeInTheDocument()
    await waitFor(() => expect(AdminService.ListCodes).toHaveBeenCalledTimes(2))
    expect(await screen.findByText('XYZ789')).toBeInTheDocument()
  })

  // Representative interaction 2: a form-driven create action (fill two
  // fields, click 登録) exercising RegisterEngine wiring + list reload.
  it('registers a new engine from the path/name fields and refreshes the engine list', async () => {
    vi.mocked(AdminService.ListEngines)
      .mockReset()
      .mockResolvedValueOnce([engine1])
      .mockResolvedValueOnce([engine1, engine2])

    render(<MasterView />)

    const user = userEvent.setup()
    await user.click(screen.getByRole('tab', { name: 'エンジン管理' }))
    expect(await screen.findByText('Apery')).toBeInTheDocument()

    await user.type(screen.getByLabelText('エンジンのパス'), 'C:\\engines\\yaneuraou.exe')
    await user.type(screen.getByLabelText('名称 (任意)'), 'YaneuraOu')
    await user.click(screen.getByRole('button', { name: '登録' }))

    expect(AdminService.RegisterEngine).toHaveBeenCalledWith('C:\\engines\\yaneuraou.exe', 'YaneuraOu')
    expect(await screen.findByText('登録しました: engine-bbbb-2222')).toBeInTheDocument()
    await waitFor(() => expect(AdminService.ListEngines).toHaveBeenCalledTimes(2))
    expect(await screen.findByText('YaneuraOu')).toBeInTheDocument()
  })

  // Representative interaction 3: server config save flow, covering the
  // cfgDirty state machine (warning appears on edit, save calls
  // ServerService.SaveConfig with the edited value, warning clears after).
  it('edits the server host, saves the config, and clears the unsaved-changes warning', async () => {
    render(<MasterView />)

    expect(await screen.findByText('停止中')).toBeInTheDocument()

    const user = userEvent.setup()
    const hostField = screen.getByLabelText('ホスト')
    await user.type(hostField, '0.0.0.0')

    expect(await screen.findByText('未保存の変更があります')).toBeInTheDocument()

    const saveButton = screen.getByRole('button', { name: '設定を保存' })
    expect(saveButton).toBeEnabled()
    await user.click(saveButton)

    expect(ServerService.SaveConfig).toHaveBeenCalledWith(
      expect.objectContaining({ host: '0.0.0.0', port: 8080, autoStart: true })
    )
    expect(await screen.findByText('設定を保存しました')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.queryByText('未保存の変更があります')).not.toBeInTheDocument()
    })
  })
})

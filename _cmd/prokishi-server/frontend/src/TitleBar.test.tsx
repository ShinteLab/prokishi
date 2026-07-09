import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TitleBar } from './TitleBar'
import { Window } from '@wailsio/runtime'

// TitleBar only touches the `Window` export of @wailsio/runtime (IsMaximised
// on mount, Minimise/ToggleMaximise on button clicks).
vi.mock('@wailsio/runtime', () => ({
  Window: {
    IsMaximised: vi.fn().mockResolvedValue(false),
    Minimise: vi.fn(),
    ToggleMaximise: vi.fn().mockResolvedValue(undefined),
  },
}))

describe('TitleBar', () => {
  beforeEach(() => {
    vi.mocked(Window.IsMaximised).mockReset().mockResolvedValue(false)
    vi.mocked(Window.Minimise).mockReset()
    vi.mocked(Window.ToggleMaximise).mockReset().mockResolvedValue(undefined)
  })

  it('opens the navigation menu, shows both destinations, and calls onNavigate with the selected view', async () => {
    const onNavigate = vi.fn()
    render(<TitleBar title="prokishi-server" view="monitor" onNavigate={onNavigate} onClose={vi.fn()} />)

    const user = userEvent.setup()
    await user.click(screen.getByTestId('MenuIcon'))

    expect(await screen.findByText('モニター')).toBeInTheDocument()
    expect(screen.getByText('マスタ管理')).toBeInTheDocument()

    await user.click(screen.getByText('マスタ管理'))

    expect(onNavigate).toHaveBeenCalledWith('master')
    // navigate() also closes the menu, so the item unmounts afterward.
    await waitFor(() => {
      expect(screen.queryByText('マスタ管理')).not.toBeInTheDocument()
    })
  })

  it('calls Window.Minimise when the minimise button is clicked', async () => {
    render(<TitleBar title="prokishi-server" view="monitor" onNavigate={vi.fn()} onClose={vi.fn()} />)

    const user = userEvent.setup()
    await user.click(screen.getByTestId('RemoveIcon'))

    expect(Window.Minimise).toHaveBeenCalledTimes(1)
  })

  it('calls Window.ToggleMaximise and re-checks IsMaximised when the maximise button is clicked', async () => {
    render(<TitleBar title="prokishi-server" view="monitor" onNavigate={vi.fn()} onClose={vi.fn()} />)

    // Initial mount check.
    await waitFor(() => expect(Window.IsMaximised).toHaveBeenCalledTimes(1))

    const user = userEvent.setup()
    await user.click(screen.getByTestId('CropSquareIcon'))

    expect(Window.ToggleMaximise).toHaveBeenCalledTimes(1)
    // handleToggleMaximise re-checks IsMaximised after toggling.
    await waitFor(() => expect(Window.IsMaximised).toHaveBeenCalledTimes(2))
  })

  it('shows the restore icon instead of the maximise icon once the window is maximised', async () => {
    vi.mocked(Window.IsMaximised).mockResolvedValue(true)
    render(<TitleBar title="prokishi-server" view="monitor" onNavigate={vi.fn()} onClose={vi.fn()} />)

    expect(await screen.findByTestId('FilterNoneIcon')).toBeInTheDocument()
    expect(screen.queryByTestId('CropSquareIcon')).not.toBeInTheDocument()
  })

  it('calls the onClose prop (not a Window method) when the close button is clicked', async () => {
    const onClose = vi.fn()
    render(<TitleBar title="prokishi-server" view="monitor" onNavigate={vi.fn()} onClose={onClose} />)

    const user = userEvent.setup()
    await user.click(screen.getByTestId('CloseIcon'))

    expect(onClose).toHaveBeenCalledTimes(1)
    expect(Window.Minimise).not.toHaveBeenCalled()
    expect(Window.ToggleMaximise).not.toHaveBeenCalled()
  })
})

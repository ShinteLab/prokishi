import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TitleBar } from '../TitleBar'

describe('TitleBar', () => {
  const defaultProps = {
    title: 'prokishi-server',
    view: 'monitor' as const,
    onNavigate: vi.fn(),
    onClose: vi.fn(),
  }

  it('renders the title', () => {
    render(<TitleBar {...defaultProps} />)
    expect(screen.getByText('prokishi-server')).toBeDefined()
  })

  it('opens menu on hamburger click', () => {
    render(<TitleBar {...defaultProps} />)
    const menuButton = screen.getByTestId('MenuIcon').closest('button')!
    fireEvent.click(menuButton)
    expect(screen.getByText('モニター')).toBeDefined()
    expect(screen.getByText('マスタ管理')).toBeDefined()
    expect(screen.getByText('システム監視')).toBeDefined()
  })

  it('calls onNavigate when menu item is clicked', () => {
    const onNavigate = vi.fn()
    render(<TitleBar {...defaultProps} onNavigate={onNavigate} />)

    const menuButton = screen.getByTestId('MenuIcon').closest('button')!
    fireEvent.click(menuButton)
    fireEvent.click(screen.getByText('マスタ管理'))

    expect(onNavigate).toHaveBeenCalledWith('master')
  })

  it('calls onClose when close button is clicked', () => {
    const onClose = vi.fn()
    render(<TitleBar {...defaultProps} onClose={onClose} />)

    const closeButton = screen.getByTestId('CloseIcon').closest('button')!
    fireEvent.click(closeButton)

    expect(onClose).toHaveBeenCalledOnce()
  })
})

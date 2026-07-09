import { vi } from 'vitest'

vi.mock('@wailsio/runtime')

Element.prototype.scrollIntoView = vi.fn()

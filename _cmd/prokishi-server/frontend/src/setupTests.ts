import '@testing-library/jest-dom/vitest'
import { vi } from 'vitest'

// Auto-mock the raw Wails runtime for any test that doesn't provide its own
// vi.mock('@wailsio/runtime', ...) factory (see __mocks__/@wailsio/runtime.ts).
// Test files that call vi.mock('@wailsio/runtime', ...) themselves override
// this on a per-file basis, since vi.mock factories are hoisted and scoped
// to the file that declares them.
vi.mock('@wailsio/runtime')

// jsdom does not implement scrollIntoView; MonitorView calls it on log
// updates to keep the view pinned to the bottom.
if (!window.HTMLElement.prototype.scrollIntoView) {
  window.HTMLElement.prototype.scrollIntoView = () => {}
}

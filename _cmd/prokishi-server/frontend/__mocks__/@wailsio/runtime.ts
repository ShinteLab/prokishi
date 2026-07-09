import { vi } from 'vitest'

export const Create = {
  Array: (fn: any) => fn,
  Nullable: (fn: any) => fn,
  Any: () => null,
  Map: () => ({}),
  Struct: (c: any) => c,
}

export const Events = {
  On: vi.fn(() => () => {}),
  Emit: vi.fn(),
  Off: vi.fn(),
}

export const CancellablePromise = Promise

export const Window = {
  SetTitle: vi.fn(),
  SetMaxSize: vi.fn(),
  SetMinSize: vi.fn(),
  SetSize: vi.fn(),
  Minimise: vi.fn(),
  ToggleMaximise: vi.fn(),
  IsMaximised: vi.fn(() => Promise.resolve(false)),
  Close: vi.fn(),
}

export const Browser = {
  OpenURL: vi.fn(),
}

export const Dialogs = {
  OpenFile: vi.fn(),
  SaveFile: vi.fn(),
  Message: vi.fn(),
  OpenDirectory: vi.fn(),
}

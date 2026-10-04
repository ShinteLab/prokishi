import { describe, it, expect } from 'vitest'
import { toMs, fmt, sendHL, recvHL, computeSendRange, limitLogs, MAX_LINES } from './MonitorView'
import type { SendRange } from './MonitorView'

describe('limitLogs', () => {
  it('returns the array as-is when the limit is off', () => {
    const logs = Array.from({ length: MAX_LINES + 500 }, (_, i) => i)
    expect(limitLogs(logs, false)).toBe(logs)
  })
  it('returns the array as-is when the limit is on but the length is within MAX_LINES', () => {
    const logs = Array.from({ length: MAX_LINES }, (_, i) => i)
    expect(limitLogs(logs, true)).toBe(logs)
  })
  it('keeps only the newest MAX_LINES entries when over the limit', () => {
    const logs = Array.from({ length: MAX_LINES + 3 }, (_, i) => i)
    const out = limitLogs(logs, true)
    expect(out).toHaveLength(MAX_LINES)
    expect(out[0]).toBe(3)
    expect(out[out.length - 1]).toBe(MAX_LINES + 2)
  })
  it('honors an explicit max', () => {
    expect(limitLogs([1, 2, 3, 4, 5], true, 2)).toEqual([4, 5])
  })
})

describe('toMs', () => {
  it('parses a valid ISO timestamp to epoch ms', () => {
    const ts = '2024-01-15T12:34:56.000Z'
    expect(toMs(ts)).toBe(new Date(ts).getTime())
  })
  it('returns 0 for null', () => {
    expect(toMs(null)).toBe(0)
  })
  it('returns 0 for undefined', () => {
    expect(toMs(undefined)).toBe(0)
  })
  it('returns NaN for a garbage non-date string (new Date() does not throw, so the catch never fires)', () => {
    // `new Date('not-a-date').getTime()` is NaN rather than throwing, so the
    // try/catch in toMs never engages and the fallback `0` is not reached.
    expect(Number.isNaN(toMs('not-a-date'))).toBe(true)
  })
})

describe('fmt', () => {
  it('formats a valid timestamp to a non-empty string different from the raw input', () => {
    const ts = '2024-01-15T12:34:56.000Z'
    const out = fmt(ts)
    expect(out).not.toBe('')
    expect(out).not.toBe(ts)
  })
  it('returns "" for null', () => {
    expect(fmt(null)).toBe('')
  })
  it('returns "" for undefined', () => {
    expect(fmt(undefined)).toBe('')
  })
  it('returns "Invalid Date" for a garbage string (toLocaleTimeString does not throw either)', () => {
    // Like toMs, Date#toLocaleTimeString on an Invalid Date returns the
    // string "Invalid Date" instead of throwing, so the catch fallback
    // `String(ts)` is never actually reached for this input.
    expect(fmt('not-a-date')).toBe('Invalid Date')
  })
})

describe('sendHL', () => {
  it('returns "normal" when sendRange is null', () => {
    expect(sendHL(null, 5)).toBe('normal')
  })
  it('returns "hi" for an index inside the range (inclusive)', () => {
    expect(sendHL({ from: 2, to: 5 }, 3)).toBe('hi')
  })
  it('returns "hi" at the exact "from" boundary', () => {
    expect(sendHL({ from: 2, to: 5 }, 2)).toBe('hi')
  })
  it('returns "hi" at the exact "to" boundary', () => {
    expect(sendHL({ from: 2, to: 5 }, 5)).toBe('hi')
  })
  it('returns "dim" for indices outside the range', () => {
    expect(sendHL({ from: 2, to: 5 }, 1)).toBe('dim')
    expect(sendHL({ from: 2, to: 5 }, 6)).toBe('dim')
  })
})

describe('recvHL', () => {
  it('returns "normal" when win is null', () => {
    expect(recvHL(null, '2024-01-15T12:00:00.000Z')).toBe('normal')
  })
  it('returns "hi" for a timestamp inside the half-open window [from, to)', () => {
    const win = { from: 1000, to: 2000 }
    expect(recvHL(win, new Date(1500).toISOString())).toBe('hi')
  })
  it('returns "hi" at the exact "from" boundary (inclusive lower bound)', () => {
    const win = { from: 1000, to: 2000 }
    expect(recvHL(win, new Date(1000).toISOString())).toBe('hi')
  })
  it('returns "dim" exactly at "to" (half-open, exclusive upper bound)', () => {
    const win = { from: 1000, to: 2000 }
    expect(recvHL(win, new Date(2000).toISOString())).toBe('dim')
  })
  it('returns "dim" outside the window', () => {
    const win = { from: 1000, to: 2000 }
    expect(recvHL(win, new Date(500).toISOString())).toBe('dim')
    expect(recvHL(win, new Date(2500).toISOString())).toBe('dim')
  })
})

describe('computeSendRange', () => {
  it('first click without shift selects a single cell and sets the anchor', () => {
    const { range, anchor } = computeSendRange(null, null, 3, false)
    expect(range).toEqual({ from: 3, to: 3 })
    expect(anchor).toBe(3)
  })

  it('clicking the same single-cell range again (no shift) clears both range and anchor', () => {
    // Mirrors the original inline condition:
    //   setSendRange(prev => (prev?.from === i && prev?.to === i) ? null : {from:i,to:i})
    //   setAnchorIdx(prev => (prev === i && sendRange?.from === i && sendRange?.to === i) ? null : i)
    const prevRange: SendRange = { from: 3, to: 3 }
    const { range, anchor } = computeSendRange(prevRange, 3, 3, false)
    expect(range).toBeNull()
    expect(anchor).toBeNull()
  })

  it('a different single click (no shift, previous range existed) selects a new single cell and updates the anchor', () => {
    const prevRange: SendRange = { from: 3, to: 3 }
    const { range, anchor } = computeSendRange(prevRange, 3, 7, false)
    expect(range).toEqual({ from: 7, to: 7 })
    expect(anchor).toBe(7)
  })

  it('shift-click with a prior anchor spans min(anchor,i) to max(anchor,i) and leaves the anchor untouched', () => {
    const prevRange: SendRange = { from: 3, to: 3 }
    const forward = computeSendRange(prevRange, 3, 8, true)
    expect(forward.range).toEqual({ from: 3, to: 8 })
    expect(forward.anchor).toBe(3)

    // Order-independence: clicking "backwards" past the anchor still spans min..max.
    const backward = computeSendRange(prevRange, 8, 3, true)
    expect(backward.range).toEqual({ from: 3, to: 8 })
    expect(backward.anchor).toBe(8)
  })

  it('shift-click with no prior anchor (anchorIdx === null) falls through to plain-click behavior', () => {
    const prevRange: SendRange = { from: 2, to: 2 }
    const shiftResult = computeSendRange(prevRange, null, 5, true)
    const plainResult = computeSendRange(prevRange, null, 5, false)
    expect(shiftResult).toEqual(plainResult)
    expect(shiftResult.range).toEqual({ from: 5, to: 5 })
    expect(shiftResult.anchor).toBe(5)
  })
})

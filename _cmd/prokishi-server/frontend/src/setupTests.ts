import '@testing-library/jest-dom/vitest'

// jsdom does not implement scrollIntoView; MonitorView calls it on log
// updates to keep the view pinned to the bottom.
if (!window.HTMLElement.prototype.scrollIntoView) {
  window.HTMLElement.prototype.scrollIntoView = () => {}
}

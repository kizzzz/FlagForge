import { JSDOM } from 'jsdom'

const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/console', pretendToBeVisual: true })
for (const name of ['window', 'document', 'navigator', 'location', 'HTMLElement', 'Element', 'SVGElement', 'Node', 'Document', 'ShadowRoot', 'MutationObserver', 'Event', 'MouseEvent', 'KeyboardEvent', 'FocusEvent']) {
  Object.defineProperty(globalThis, name, { value: dom.window[name], configurable: true })
}
globalThis.getComputedStyle = element => dom.window.getComputedStyle(element)
globalThis.requestAnimationFrame = dom.window.requestAnimationFrame.bind(dom.window)
globalThis.cancelAnimationFrame = dom.window.cancelAnimationFrame.bind(dom.window)
globalThis.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} }
window.matchMedia = () => ({ matches: false, addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {} })
window.scrollTo = () => {}
globalThis.IS_REACT_ACT_ENVIRONMENT = true

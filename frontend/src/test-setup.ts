// Vitest setup — runs once before every test file.
//
// jsdom 29 + Vitest 4 do not expose `localStorage` on the global scope by
// default (window.localStorage exists but the bare `localStorage` identifier
// is undefined). Components like NewSessionModal touch `localStorage` during
// useState initializers, so render-based tests crash on mount without this
// polyfill.
//
// Added by Phase 101-02 (SHELL-01 GUI half) — first plan to need DOM-render
// tests against components that use localStorage.

if (typeof globalThis.localStorage === 'undefined' && typeof window !== 'undefined') {
  const store = new Map<string, string>()
  const fallback: Storage = {
    get length() {
      return store.size
    },
    clear() {
      store.clear()
    },
    getItem(key) {
      return store.has(key) ? store.get(key)! : null
    },
    key(index) {
      return Array.from(store.keys())[index] ?? null
    },
    removeItem(key) {
      store.delete(key)
    },
    setItem(key, value) {
      store.set(key, String(value))
    },
  }
  // Prefer the jsdom-provided implementation if window.localStorage exists.
  const target: Storage = (window as any).localStorage ?? fallback
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    get: () => target,
  })
}

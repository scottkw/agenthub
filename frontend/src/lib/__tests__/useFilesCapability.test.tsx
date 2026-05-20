import React from 'react'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createRoot, type Root } from 'react-dom/client'
import { act } from 'react'
import { useFilesCapability, type CapabilityState } from '../useFilesCapability'
import { FilesApiClient, FilesApiError, type FileListResponse } from '../filesApi'

// Phase 120-02 Task 4 — useFilesCapability hook tests.
// Minimal manual renderHook harness — @testing-library/react is NOT in
// devDependencies (verified via frontend/package.json), so we drive a
// hook host through react-dom/client + the React 19 `act` import.

interface HookSnapshot {
  state: CapabilityState
  retry: () => void
}

function renderHook<T>(useHook: () => T): { current: { value: T }; root: Root; container: HTMLDivElement; flush: () => Promise<void> } {
  const ref: { value: T } = { value: undefined as unknown as T }
  function Host(): React.ReactElement | null {
    ref.value = useHook()
    return null
  }
  const container = document.createElement('div')
  document.body.appendChild(container)
  let root!: Root
  // act in React 19 returns a thenable; awaiting it flushes effects.
  // Synchronous mount block — capture root after creation.
  // (Vitest/jsdom env.)
  // We perform the act() in the caller as needed.
  // Initial mount:
  // eslint-disable-next-line @typescript-eslint/no-floating-promises
  void act(() => {
    root = createRoot(container)
    root.render(<Host />)
  })
  async function flush(): Promise<void> {
    await act(async () => {
      // Allow microtasks (Promise resolution inside the effect) to settle.
      await Promise.resolve()
      await Promise.resolve()
    })
  }
  return { current: ref, root, container, flush }
}

function makeMockClient(overrides: Partial<FilesApiClient> = {}): FilesApiClient {
  // Construct a real instance so `instanceof FilesApiClient` works if the hook checks it.
  const client = new FilesApiClient({ baseURL: 'http://host' })
  // Stub listFiles per test
  ;(client as unknown as { listFiles: FilesApiClient['listFiles'] }).listFiles =
    (overrides.listFiles ??
      (async () => {
        throw new Error('not stubbed')
      })) as FilesApiClient['listFiles']
  return client
}

beforeEach(() => {
  vi.useRealTimers()
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('useFilesCapability', () => {
  it('returns "unknown" when client is null', async () => {
    const { current, flush } = renderHook<HookSnapshot>(() => useFilesCapability(null, 'sid'))
    await flush()
    expect(current.value.state).toBe('unknown')
  })

  it('returns "present" on successful listFiles probe', async () => {
    const okResp: FileListResponse = { entries: [], truncated: false, refreshedAt: null }
    const client = makeMockClient({ listFiles: vi.fn(async () => okResp) as FilesApiClient['listFiles'] })
    const { current, flush } = renderHook<HookSnapshot>(() => useFilesCapability(client, 'sid'))
    await flush()
    expect(current.value.state).toBe('present')
  })

  it('returns "denied" on 403 with "files.read" in body', async () => {
    const client = makeMockClient({
      listFiles: vi.fn(async () => {
        throw new FilesApiError(403, 'cap missing files.read perm')
      }) as FilesApiClient['listFiles'],
    })
    const { current, flush } = renderHook<HookSnapshot>(() => useFilesCapability(client, 'sid'))
    await flush()
    expect(current.value.state).toBe('denied')
  })

  it('returns "probe-failed" on 403 without "files.read" in body', async () => {
    const client = makeMockClient({
      listFiles: vi.fn(async () => {
        throw new FilesApiError(403, 'permission denied for path')
      }) as FilesApiClient['listFiles'],
    })
    const { current, flush } = renderHook<HookSnapshot>(() => useFilesCapability(client, 'sid'))
    await flush()
    expect(current.value.state).toBe('probe-failed')
  })

  it('returns "probe-failed" on network throw', async () => {
    const client = makeMockClient({
      listFiles: vi.fn(async () => {
        throw new TypeError('fetch failed')
      }) as FilesApiClient['listFiles'],
    })
    const { current, flush } = renderHook<HookSnapshot>(() => useFilesCapability(client, 'sid'))
    await flush()
    expect(current.value.state).toBe('probe-failed')
  })

  it('retry() re-issues the probe', async () => {
    const okResp: FileListResponse = { entries: [], truncated: false, refreshedAt: null }
    let call = 0
    const listFiles = vi.fn(async () => {
      call++
      if (call === 1) throw new TypeError('first call fails')
      return okResp
    }) as FilesApiClient['listFiles']
    const client = makeMockClient({ listFiles })
    const { current, flush } = renderHook<HookSnapshot>(() => useFilesCapability(client, 'sid'))
    await flush()
    expect(current.value.state).toBe('probe-failed')

    await act(async () => {
      current.value.retry()
    })
    await flush()
    expect(current.value.state).toBe('present')
    expect(listFiles).toHaveBeenCalledTimes(2)
  })
})

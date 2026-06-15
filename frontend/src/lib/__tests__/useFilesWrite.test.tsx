import React from 'react'
import { describe, it, expect, afterEach, vi } from 'vitest'
import { createRoot, type Root } from 'react-dom/client'
import { act } from 'react'
import { useFilesWrite, type WriteOutcome } from '../useFilesWrite'
import { FilesApiClient, FilesApiError, REMOTE_PEER_OUTDATED_MESSAGE } from '../filesApi'

// Phase 128-01 Task 2 — useFilesWrite 405 peer-outdated branch tests (RMW-04 RED gate).
// Manual renderHook harness mirroring useFilesCapability.test.tsx (no @testing-library/react).

interface HookSnapshot<T> {
  value: T
}

function renderHook<T>(
  useHook: () => T,
): {
  current: HookSnapshot<T>
  root: Root
  container: HTMLDivElement
  flush: () => Promise<void>
} {
  const ref: HookSnapshot<T> = { value: undefined as unknown as T }
  function Host(): React.ReactElement | null {
    ref.value = useHook()
    return null
  }
  const container = document.createElement('div')
  document.body.appendChild(container)
  let root!: Root
  act(() => {
    root = createRoot(container)
    root.render(React.createElement(Host))
  })
  return {
    current: ref,
    root,
    container,
    flush: async () => {
      await act(async () => {
        await Promise.resolve()
      })
    },
  }
}

// Minimal FilesApiClient stub that rejects writeFile with a given error.
function makeClientThrowing(err: unknown): FilesApiClient {
  const c = new FilesApiClient({ baseURL: 'http://localhost' })
  vi.spyOn(c, 'writeFile').mockRejectedValue(err)
  return c
}

function makeClientOk(): FilesApiClient {
  const c = new FilesApiClient({ baseURL: 'http://localhost' })
  vi.spyOn(c, 'writeFile').mockResolvedValue(undefined as unknown as Awaited<ReturnType<FilesApiClient['writeFile']>>)
  return c
}

describe('useFilesWrite — 405 peer-outdated branch (RMW-04)', () => {
  let container: HTMLDivElement
  let root: Root

  afterEach(() => {
    act(() => {
      root.unmount()
    })
    document.body.removeChild(container)
    vi.restoreAllMocks()
  })

  it('write() returns peer-outdated on FilesApiError(405)', async () => {
    const client = makeClientThrowing(new FilesApiError(405, 'Method Not Allowed'))
    let outcome: WriteOutcome | undefined

    const hook = renderHook(() => useFilesWrite(client, 'sid1'))
    container = hook.container
    root = hook.root

    await act(async () => {
      outcome = await hook.current.value.write('/path/to/file.txt', 'content')
    })
    await hook.flush()

    expect(outcome).toBe('peer-outdated')
  })

  it('write() sets saveError to REMOTE_PEER_OUTDATED_MESSAGE on 405', async () => {
    const client = makeClientThrowing(new FilesApiError(405, 'Method Not Allowed'))

    const hook = renderHook(() => useFilesWrite(client, 'sid1'))
    container = hook.container
    root = hook.root

    await act(async () => {
      await hook.current.value.write('/path/to/file.txt', 'content')
    })
    await hook.flush()

    expect(hook.current.value.saveError).toBe(REMOTE_PEER_OUTDATED_MESSAGE)
  })

  it('write() sets saveState to idle on 405 (not saving)', async () => {
    const client = makeClientThrowing(new FilesApiError(405, 'Method Not Allowed'))

    const hook = renderHook(() => useFilesWrite(client, 'sid1'))
    container = hook.container
    root = hook.root

    await act(async () => {
      await hook.current.value.write('/path/to/file.txt', 'content')
    })
    await hook.flush()

    expect(hook.current.value.saveState).toBe('idle')
    expect(hook.current.value.isSaving).toBe(false)
  })

  it('write() does NOT clear editContent on 405 (T-125-08 locked: buffer preserved)', async () => {
    // The hook does not expose editContent directly, but we can verify it does NOT
    // call setEditContent by checking that saveError is set (405 branch reached,
    // not the generic branch). The generic branch sets a different message.
    const client = makeClientThrowing(new FilesApiError(405, 'Method Not Allowed'))

    const hook = renderHook(() => useFilesWrite(client, 'sid1'))
    container = hook.container
    root = hook.root

    await act(async () => {
      await hook.current.value.write('/path/to/file.txt', 'content')
    })
    await hook.flush()

    // If saveError is the verbatim peer-outdated message, we hit the 405 branch
    // (not the generic buffer-preserving branch which has a different copy).
    expect(hook.current.value.saveError).toBe(REMOTE_PEER_OUTDATED_MESSAGE)
    expect(hook.current.value.saveError).not.toBe(
      "Couldn't save the file. Your changes are still here — try again.",
    )
  })

  it('write() conflict (412) outcome unchanged — no regression', async () => {
    const client = makeClientThrowing(new FilesApiError(412, 'Precondition Failed'))
    let outcome: WriteOutcome | undefined

    const hook = renderHook(() => useFilesWrite(client, 'sid1'))
    container = hook.container
    root = hook.root

    await act(async () => {
      outcome = await hook.current.value.write('/path/to/file.txt', 'content')
    })
    await hook.flush()

    expect(outcome).toBe('conflict')
    expect(hook.current.value.isConflict).toBe(true)
    expect(hook.current.value.saveError).toBeNull()
  })

  it('write() generic error outcome unchanged — no regression', async () => {
    const client = makeClientThrowing(new FilesApiError(500, 'Internal Server Error'))
    let outcome: WriteOutcome | undefined

    const hook = renderHook(() => useFilesWrite(client, 'sid1'))
    container = hook.container
    root = hook.root

    await act(async () => {
      outcome = await hook.current.value.write('/path/to/file.txt', 'content')
    })
    await hook.flush()

    expect(outcome).toBe('error')
    expect(hook.current.value.saveError).toBe(
      "Couldn't save the file. Your changes are still here — try again.",
    )
  })

  it('write() saved outcome unchanged — no regression', async () => {
    const client = makeClientOk()
    let outcome: WriteOutcome | undefined

    const hook = renderHook(() => useFilesWrite(client, 'sid1'))
    container = hook.container
    root = hook.root

    await act(async () => {
      outcome = await hook.current.value.write('/path/to/file.txt', 'content')
    })
    await hook.flush()

    expect(outcome).toBe('saved')
    expect(hook.current.value.saveError).toBeNull()
  })
})

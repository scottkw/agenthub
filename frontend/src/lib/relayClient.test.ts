import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  MSG_INPUT,
  MSG_OUTPUT,
  MSG_RESIZE,
  MSG_RESIZE2,
  MSG_PRESENCE,
  MSG_TYPING,
  MSG_ALIAS_SET,
  MSG_CHAT,
  MSG_CHAT_SEND,
  MSG_SESSION_INJECT,
  MSG_INJECT_ERROR,
  encodeInputFrame,
  encodeResizeFrame,
  encodeTypingFrame,
  encodeAliasSetFrame,
  encodeChatSendFrame,
  encodeSessionInjectFrame,
  parseServerFrame,
  RelayClient,
  type ChatMessage,
  type RelayClientCallbacks,
} from './relayClient'

describe('encodeInputFrame', () => {
  it('prepends MSG_INPUT byte (0x10) to UTF-8 encoded text', () => {
    const result = encodeInputFrame('hello')
    expect(result).toBeInstanceOf(Uint8Array)
    expect(result[0]).toBe(MSG_INPUT) // 0x10
    expect(Array.from(result)).toEqual([0x10, 104, 101, 108, 108, 111])
  })

  it('handles empty string', () => {
    const result = encodeInputFrame('')
    expect(result).toBeInstanceOf(Uint8Array)
    expect(result.length).toBe(1)
    expect(result[0]).toBe(MSG_INPUT)
  })

  it('handles multi-byte unicode characters', () => {
    const result = encodeInputFrame('A')
    expect(result[0]).toBe(MSG_INPUT)
    expect(result[1]).toBe(65) // 'A' = 0x41
  })
})

describe('encodeResizeFrame', () => {
  it('produces 5-byte frame [MSG_RESIZE2, cols_hi, cols_lo, rows_hi, rows_lo]', () => {
    const result = encodeResizeFrame(120, 40)
    expect(result).toBeInstanceOf(Uint8Array)
    expect(result.length).toBe(5)
    expect(Array.from(result)).toEqual([0x11, 0, 120, 0, 40])
  })

  it('handles large col/row values', () => {
    const result = encodeResizeFrame(256, 128)
    expect(result[0]).toBe(MSG_RESIZE2) // 0x11
    expect(result[1]).toBe(1) // cols_hi = 256 >> 8 = 1
    expect(result[2]).toBe(0) // cols_lo = 256 & 0xff = 0
    expect(result[3]).toBe(0) // rows_hi = 128 >> 8 = 0
    expect(result[4]).toBe(128) // rows_lo = 128 & 0xff = 128
  })

  it('handles 80x24 standard terminal size', () => {
    const result = encodeResizeFrame(80, 24)
    expect(Array.from(result)).toEqual([0x11, 0, 80, 0, 24])
  })
})

describe('parseServerFrame', () => {
  it('parses output frame (0x01) returning type output and payload', () => {
    const data = new Uint8Array([MSG_OUTPUT, 72, 101, 108, 108, 111])
    const result = parseServerFrame(data)
    expect(result.type).toBe('output')
    if (result.type === 'output') {
      expect(result.payload).toBeInstanceOf(Uint8Array)
      expect(Array.from(result.payload)).toEqual([72, 101, 108, 108, 111])
    }
  })

  it('parses resize frame (0x02) returning cols and rows', () => {
    const data = new Uint8Array([0x02, 0, 80, 0, 24])
    const result = parseServerFrame(data)
    expect(result.type).toBe('resize')
    if (result.type === 'resize') {
      expect(result.cols).toBe(80)
      expect(result.rows).toBe(24)
    }
  })

  it('returns unknown for empty frame', () => {
    const data = new Uint8Array([])
    const result = parseServerFrame(data)
    expect(result.type).toBe('unknown')
  })

  it('returns unknown for unrecognized type byte', () => {
    const data = new Uint8Array([0xff, 1, 2, 3])
    const result = parseServerFrame(data)
    expect(result.type).toBe('unknown')
  })

  it('parses output frame with no payload bytes', () => {
    const data = new Uint8Array([0x01])
    const result = parseServerFrame(data)
    expect(result.type).toBe('output')
    if (result.type === 'output') {
      expect(result.payload.length).toBe(0)
    }
  })
})

// Phase 152 presence/typing/alias wire protocol tests
describe('MSG_PRESENCE / MSG_TYPING / MSG_ALIAS_SET constants', () => {
  it('MSG_PRESENCE is 0x32', () => {
    expect(MSG_PRESENCE).toBe(0x32)
  })
  it('MSG_TYPING is 0x33', () => {
    expect(MSG_TYPING).toBe(0x33)
  })
  it('MSG_ALIAS_SET is 0x34', () => {
    expect(MSG_ALIAS_SET).toBe(0x34)
  })
})

describe('parseServerFrame — presence/typing/alias frame decoding', () => {
  it('decodes a MsgPresence (0x32) frame into a presence variant with participants', () => {
    const body = JSON.stringify({
      participants: [
        { personKey: 'local:local', tailnetID: 'local', origin: 'local', alias: 'ken', connCount: 1 },
      ],
    })
    const encoded = new TextEncoder().encode(body)
    const data = new Uint8Array([0x32, ...encoded])
    const result = parseServerFrame(data)
    expect(result.type).toBe('presence')
    if (result.type === 'presence') {
      expect(result.participants).toHaveLength(1)
      expect(result.participants[0].personKey).toBe('local:local')
      expect(result.participants[0].tailnetID).toBe('local')
      expect(result.participants[0].origin).toBe('local')
      expect(result.participants[0].alias).toBe('ken')
      expect(result.participants[0].connCount).toBe(1)
    }
  })

  it('decodes a MsgTyping (0x33) frame into a typing variant', () => {
    const body = JSON.stringify({ personKey: 'k:web', alias: 'sam', typing: true })
    const encoded = new TextEncoder().encode(body)
    const data = new Uint8Array([0x33, ...encoded])
    const result = parseServerFrame(data)
    expect(result.type).toBe('typing')
    if (result.type === 'typing') {
      expect(result.personKey).toBe('k:web')
      expect(result.alias).toBe('sam')
      expect(result.typing).toBe(true)
    }
  })

  it('decodes a MsgChat (0x30) frame into a chat variant (Phase 154 wired)', () => {
    const msg: ChatMessage = {
      v: 1, id: 'id1', sessionID: 'sess', authorID: 'local',
      alias: 'ken', content: 'hello', ts: 1_700_000_000_000,
    }
    const body = JSON.stringify(msg)
    const encoded = new TextEncoder().encode(body)
    const data = new Uint8Array([0x30, ...encoded])
    const result = parseServerFrame(data)
    expect(result.type).toBe('chat')
    if (result.type === 'chat') {
      expect(result.message.alias).toBe('ken')
      expect(result.message.content).toBe('hello')
    }
  })

  it('returns unknown for MsgChatSend (0x31) — client-to-server frame; not parsed', () => {
    const body = JSON.stringify({ content: 'hello' })
    const encoded = new TextEncoder().encode(body)
    const data = new Uint8Array([0x31, ...encoded])
    const result = parseServerFrame(data)
    expect(result.type).toBe('unknown')
  })

  it('returns unknown for a malformed presence body (invalid JSON after 0x32)', () => {
    const data = new Uint8Array([0x32, ...new TextEncoder().encode('not-json{')])
    const result = parseServerFrame(data)
    expect(result.type).toBe('unknown')
  })

  it('presence frame with missing participants field returns empty array', () => {
    const body = JSON.stringify({})
    const encoded = new TextEncoder().encode(body)
    const data = new Uint8Array([0x32, ...encoded])
    const result = parseServerFrame(data)
    expect(result.type).toBe('presence')
    if (result.type === 'presence') {
      expect(result.participants).toEqual([])
    }
  })
})

describe('encodeTypingFrame', () => {
  it('produces frame with leading byte 0x33', () => {
    const frame = encodeTypingFrame(true)
    expect(frame[0]).toBe(0x33)
  })

  it('JSON body parses to {typing:true}', () => {
    const frame = encodeTypingFrame(true)
    const json = new TextDecoder().decode(frame.slice(1))
    expect(JSON.parse(json)).toEqual({ typing: true })
  })

  it('JSON body parses to {typing:false} for typing stop', () => {
    const frame = encodeTypingFrame(false)
    const json = new TextDecoder().decode(frame.slice(1))
    expect(JSON.parse(json)).toEqual({ typing: false })
  })
})

describe('encodeAliasSetFrame', () => {
  it('produces frame with leading byte 0x34', () => {
    const frame = encodeAliasSetFrame('ken')
    expect(frame[0]).toBe(0x34)
  })

  it('JSON body parses to {alias:"ken"}', () => {
    const frame = encodeAliasSetFrame('ken')
    const json = new TextDecoder().decode(frame.slice(1))
    expect(JSON.parse(json)).toEqual({ alias: 'ken' })
  })

  it('handles alias with unicode characters', () => {
    const frame = encodeAliasSetFrame('café')
    const json = new TextDecoder().decode(frame.slice(1))
    expect(JSON.parse(json)).toEqual({ alias: 'café' })
  })
})

// FE-URL-01: RelayClient URL construction — local-direct vs daemon-proxy paths
describe('RelayClient URL construction', () => {
  let capturedUrl: string | null = null

  // Stub global WebSocket so constructing RelayClient does not open a real socket
  // and we can capture the URL passed to the constructor.
  beforeEach(() => {
    capturedUrl = null
    // Minimal WebSocket stub: record the constructed URL, expose a no-op API
    const MockWebSocket = vi.fn(function (this: WebSocket, url: string) {
      capturedUrl = url
      // Satisfy the RelayClient constructor's property access
      this.binaryType = 'arraybuffer'
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      ;(this as any).readyState = 0 // CONNECTING
    }) as unknown as typeof WebSocket
    // Assign prototype properties so the shape looks like a real WebSocket
    Object.assign(MockWebSocket.prototype, {
      close: vi.fn(),
      send: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })
    vi.stubGlobal('WebSocket', MockWebSocket)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  // FE-URL-01a: local-direct path (no opts — preserves every existing caller)
  it('FE-URL-01a: builds the local-direct path when no opts are passed', () => {
    const port = 51234
    const sid = 'test-session-abc'
    new RelayClient(port, sid, { onOutput: () => {} })
    expect(capturedUrl).toBe(`ws://127.0.0.1:${port}/sessions/${sid}/ws`)
    // The cap token is absent from the URL (T-134-07-01)
    expect(capturedUrl).not.toContain('cap')
  })

  // FE-URL-01b: remote path (remote: true) — daemon-proxy URL
  it('FE-URL-01b: builds the daemon-proxy path when opts.remote is true', () => {
    const port = 51234
    const sid = 'remote-session-xyz'
    new RelayClient(port, sid, { onOutput: () => {} }, { remote: true })
    expect(capturedUrl).toBe(`ws://127.0.0.1:${port}/api/relay/remote/${sid}/ws`)
    // The cap token is absent from the URL (T-134-07-01)
    expect(capturedUrl).not.toContain('cap')
  })

  // FE-URL-01a variant: explicit remote: false is treated as local-direct
  it('FE-URL-01a variant: explicit remote: false builds the local-direct path', () => {
    const port = 51234
    const sid = 'local-session-def'
    new RelayClient(port, sid, { onOutput: () => {} }, { remote: false })
    expect(capturedUrl).toBe(`ws://127.0.0.1:${port}/sessions/${sid}/ws`)
  })
})

// ─── Phase 154: new constants, encoders, parse cases, dispatch ────────────────

describe('Phase 154 constants', () => {
  it('MSG_CHAT is 0x30', () => { expect(MSG_CHAT).toBe(0x30) })
  it('MSG_CHAT_SEND is 0x31', () => { expect(MSG_CHAT_SEND).toBe(0x31) })
  it('MSG_SESSION_INJECT is 0x35', () => { expect(MSG_SESSION_INJECT).toBe(0x35) })
  it('MSG_INJECT_ERROR is 0x36', () => { expect(MSG_INJECT_ERROR).toBe(0x36) })
})

describe('encodeChatSendFrame (Phase 154 behavior)', () => {
  it('byte[0] === MSG_CHAT_SEND (0x31)', () => {
    const frame = encodeChatSendFrame('hi')
    expect(frame[0]).toBe(0x31)
  })

  it('remaining bytes JSON-decode to { content: "hi" }', () => {
    const frame = encodeChatSendFrame('hi')
    const body = new TextDecoder().decode(frame.slice(1))
    expect(JSON.parse(body)).toEqual({ content: 'hi' })
  })
})

describe('encodeSessionInjectFrame (Phase 154 behavior)', () => {
  it('byte[0] === MSG_SESSION_INJECT (0x35)', () => {
    const frame = encodeSessionInjectFrame('run')
    expect(frame[0]).toBe(0x35)
  })

  it('remaining bytes JSON-decode to { text: "run" }', () => {
    const frame = encodeSessionInjectFrame('run')
    const body = new TextDecoder().decode(frame.slice(1))
    expect(JSON.parse(body)).toEqual({ text: 'run' })
  })
})

// Helper: build a binary MSG_CHAT frame from a partial ChatMessage
function buildChatFrame(msg: Partial<ChatMessage>): Uint8Array {
  const defaults: ChatMessage = {
    v: 1,
    id: 'test-id',
    sessionID: 'sess-1',
    authorID: 'local',
    alias: 'Alice',   // mirrors Go json:"alias" tag — NOT authorAlias
    content: 'Hello',
    ts: 1_700_000_000_000,
  }
  const json = JSON.stringify({ ...defaults, ...msg })
  const encoded = new TextEncoder().encode(json)
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_CHAT
  frame.set(encoded, 1)
  return frame
}

describe('parseServerFrame — Phase 154 chat/inject_error behaviors', () => {
  // Behavior 3: MSG_CHAT (0x30) decodes and exposes message.alias (proves alias not authorAlias)
  it('decodes 0x30 frame → { type:"chat", message } with message.alias populated', () => {
    const frame = buildChatFrame({ alias: 'Bob', content: 'hey there' })
    const result = parseServerFrame(frame)
    expect(result.type).toBe('chat')
    if (result.type === 'chat') {
      expect(result.message.alias).toBe('Bob')
      expect(result.message.content).toBe('hey there')
      // Explicitly confirm the field name — alias (not authorAlias)
      expect('alias' in result.message).toBe(true)
    }
  })

  // Behavior 4: malformed 0x30 body → { type:'unknown' } (try/catch guard — T-154-05)
  it('returns { type:"unknown" } when 0x30 body is malformed JSON', () => {
    const garbage = new Uint8Array(5)
    garbage[0] = MSG_CHAT
    // bytes 1-4 are 0x00 — not valid JSON
    const result = parseServerFrame(garbage)
    expect(result.type).toBe('unknown')
  })

  // Behavior 5: MSG_INJECT_ERROR (0x36) → { type:'inject_error', reason }
  it('decodes 0x36 frame → { type:"inject_error", reason }', () => {
    const json = JSON.stringify({ reason: 'read only' })
    const encoded = new TextEncoder().encode(json)
    const frame = new Uint8Array(1 + encoded.length)
    frame[0] = MSG_INJECT_ERROR
    frame.set(encoded, 1)

    const result = parseServerFrame(frame)
    expect(result.type).toBe('inject_error')
    if (result.type === 'inject_error') {
      expect(result.reason).toBe('read only')
    }
  })
})

// Behavior 6: backward compat — RelayClient with only onOutput does not throw when a chat frame arrives
describe('RelayClient Phase 154 backward compatibility', () => {
  // Track the most recently constructed mock WS instance so we can fire onmessage
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let lastMockWS: any

  beforeEach(() => {
    lastMockWS = null
    // Use a regular function (not arrow) so `new MockWS(url)` works as a constructor
    const MockWS = vi.fn(function (this: Record<string, unknown>) {
      this.binaryType = 'arraybuffer'
      this.readyState = 1 // OPEN
      this.onopen = null
      this.onmessage = null
      this.onclose = null
      this.onerror = null
      this.send = vi.fn()
      this.close = vi.fn()
      // eslint-disable-next-line @typescript-eslint/no-this-alias
      lastMockWS = this
    }) as unknown as typeof WebSocket
    vi.stubGlobal('WebSocket', MockWS)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('constructs successfully when only onOutput provided (TerminalPanel pattern)', () => {
    // All chat callbacks are intentionally omitted — this is TerminalPanel's usage pattern
    const callbacks: RelayClientCallbacks = { onOutput: vi.fn() }
    expect(() => new RelayClient(34115, 'sess-1', callbacks)).not.toThrow()
  })

  it('does not throw when a chat frame arrives and onChat is not provided', () => {
    const callbacks: RelayClientCallbacks = { onOutput: vi.fn() }
    const client = new RelayClient(34115, 'sess-1', callbacks)

    const chatFrame = buildChatFrame({ content: 'test backward compat' })
    const buf = chatFrame.buffer.slice(
      chatFrame.byteOffset,
      chatFrame.byteOffset + chatFrame.byteLength,
    ) as ArrayBuffer

    // onmessage is set by RelayClient constructor; simulate incoming chat frame
    expect(() => {
      // eslint-disable-next-line @typescript-eslint/no-unsafe-call
      lastMockWS.onmessage?.({ data: buf })
    }).not.toThrow()

    client.close()
  })
})

// ─── Phase 157 VIEW-04: 0x02 MsgResize dispatch → onResize callback ───────────

describe('RelayClient Phase 157 — 0x02 MsgResize dispatch', () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let lastMockWS: any

  beforeEach(() => {
    lastMockWS = null
    const MockWS = vi.fn(function (this: Record<string, unknown>) {
      this.binaryType = 'arraybuffer'
      this.readyState = 1 // OPEN
      this.onopen = null
      this.onmessage = null
      this.onclose = null
      this.onerror = null
      this.send = vi.fn()
      this.close = vi.fn()
      // eslint-disable-next-line @typescript-eslint/no-this-alias
      lastMockWS = this
    }) as unknown as typeof WebSocket
    vi.stubGlobal('WebSocket', MockWS)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('MSG_RESIZE constant is 0x02', () => {
    expect(MSG_RESIZE).toBe(0x02)
  })

  it('fires onResize(cols, rows) when a 5-byte 0x02 frame arrives', () => {
    const onResizeFn = vi.fn()
    const callbacks: RelayClientCallbacks = { onOutput: vi.fn(), onResize: onResizeFn }
    const client = new RelayClient(34115, 'sess-resize', callbacks)

    // Build a 0x02 frame: [0x02, 0, 80, 0, 24] → cols=80, rows=24
    const frame = new Uint8Array([0x02, 0, 80, 0, 24])
    const buf = frame.buffer as ArrayBuffer
    lastMockWS.onmessage?.({ data: buf })

    expect(onResizeFn).toHaveBeenCalledTimes(1)
    expect(onResizeFn).toHaveBeenCalledWith(80, 24)

    client.close()
  })

  it('fires onResize with large col/row values (big-endian decode)', () => {
    const onResizeFn = vi.fn()
    const callbacks: RelayClientCallbacks = { onOutput: vi.fn(), onResize: onResizeFn }
    const client = new RelayClient(34115, 'sess-big', callbacks)

    // cols=256 (0x01, 0x00), rows=128 (0x00, 0x80)
    const frame = new Uint8Array([0x02, 0x01, 0x00, 0x00, 0x80])
    lastMockWS.onmessage?.({ data: frame.buffer as ArrayBuffer })

    expect(onResizeFn).toHaveBeenCalledWith(256, 128)

    client.close()
  })

  it('does not fire onResize when a short (< 5 byte) 0x02 frame arrives', () => {
    const onResizeFn = vi.fn()
    const callbacks: RelayClientCallbacks = { onOutput: vi.fn(), onResize: onResizeFn }
    const client = new RelayClient(34115, 'sess-short', callbacks)

    // Truncated frame — parseServerFrame returns { type: 'unknown' }
    const frame = new Uint8Array([0x02, 0, 80, 0]) // only 4 bytes
    lastMockWS.onmessage?.({ data: frame.buffer as ArrayBuffer })

    expect(onResizeFn).not.toHaveBeenCalled()

    client.close()
  })

  it('does not throw when onResize is omitted (host-path backward compat)', () => {
    // Host path: only onOutput provided — no onResize
    const callbacks: RelayClientCallbacks = { onOutput: vi.fn() }
    const client = new RelayClient(34115, 'sess-host', callbacks)

    const frame = new Uint8Array([0x02, 0, 80, 0, 24])
    expect(() => {
      lastMockWS.onmessage?.({ data: frame.buffer as ArrayBuffer })
    }).not.toThrow()

    client.close()
  })
})

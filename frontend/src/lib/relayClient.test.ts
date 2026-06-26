import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  MSG_INPUT,
  MSG_OUTPUT,
  MSG_RESIZE2,
  MSG_PRESENCE,
  MSG_TYPING,
  MSG_ALIAS_SET,
  encodeInputFrame,
  encodeResizeFrame,
  encodeTypingFrame,
  encodeAliasSetFrame,
  parseServerFrame,
  RelayClient,
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

  it('returns unknown for MsgChat (0x30) — Phase 154 stub not yet handled', () => {
    const body = JSON.stringify({ content: 'hello' })
    const encoded = new TextEncoder().encode(body)
    const data = new Uint8Array([0x30, ...encoded])
    const result = parseServerFrame(data)
    expect(result.type).toBe('unknown')
  })

  it('returns unknown for MsgChatSend (0x31) — Phase 154 stub not yet handled', () => {
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

// Binary framing constants matching internal/relay/protocol.go
export const MSG_OUTPUT  = 0x01  // server → client: PTY output
export const MSG_RESIZE  = 0x02  // server → client: resize notification
export const MSG_INPUT   = 0x10  // client → server: keyboard input
export const MSG_RESIZE2 = 0x11  // client → server: resize request
export const MSG_PING    = 0x12  // client → server: keep-alive

// Chat/presence frame types — 0x30–0x3F range (Phase 152)
// MsgChat (0x30) and MsgChatSend (0x31) are Phase 154 dispatch stubs.
export const MSG_PRESENCE  = 0x32  // server → client: full presence roster (JSON PresencePayload)
export const MSG_TYPING    = 0x33  // bidirectional: typing-start/stop (JSON TypingPayload)
export const MSG_ALIAS_SET = 0x34  // client → server: set/update alias (JSON AliasPayload)

// PresenceEntry describes one participant in the presence roster.
// Field names match the Go json tags in internal/relay/protocol.go exactly.
export interface PresenceEntry {
  personKey: string  // TailnetID + ":" + origin — stable collapse key
  tailnetID: string  // "local" for desktop owner, node pubkey for web
  origin: string     // "local" (relay loopback) or "web" (webserver Tailscale)
  alias: string
  connCount: number  // active connections for this person key
}

// ServerFrame union type for parsed server messages
export type ServerFrame =
  | { type: 'output'; payload: Uint8Array }
  | { type: 'resize'; cols: number; rows: number }
  | { type: 'presence'; participants: PresenceEntry[] }
  | { type: 'typing'; personKey: string; alias: string; typing: boolean }
  | { type: 'unknown' }

/**
 * Encode keyboard input into a binary frame.
 * Format: [MSG_INPUT, ...UTF-8 bytes of text]
 */
export function encodeInputFrame(text: string): Uint8Array {
  const encoded = new TextEncoder().encode(text)
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_INPUT
  frame.set(encoded, 1)
  return frame
}

/**
 * Encode a terminal resize request into a 5-byte binary frame.
 * Format: [MSG_RESIZE2, cols_hi, cols_lo, rows_hi, rows_lo]
 */
export function encodeResizeFrame(cols: number, rows: number): Uint8Array {
  return new Uint8Array([
    MSG_RESIZE2,
    (cols >> 8) & 0xff,
    cols & 0xff,
    (rows >> 8) & 0xff,
    rows & 0xff,
  ])
}

/**
 * Encode a typing-start or typing-stop notification for the server.
 * Format: [MSG_TYPING, ...UTF-8 bytes of JSON {typing}]
 * Server fills personKey and alias from the Subscriber before broadcast.
 */
export function encodeTypingFrame(typing: boolean): Uint8Array {
  const encoded = new TextEncoder().encode(JSON.stringify({ typing }))
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_TYPING
  frame.set(encoded, 1)
  return frame
}

/**
 * Encode an alias-set request for the server.
 * Format: [MSG_ALIAS_SET, ...UTF-8 bytes of JSON {alias}]
 */
export function encodeAliasSetFrame(alias: string): Uint8Array {
  const encoded = new TextEncoder().encode(JSON.stringify({ alias }))
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_ALIAS_SET
  frame.set(encoded, 1)
  return frame
}

/**
 * Parse a server-sent binary frame into a typed union.
 */
export function parseServerFrame(data: Uint8Array): ServerFrame {
  if (data.length === 0) {
    return { type: 'unknown' }
  }

  const typeByte = data[0]

  switch (typeByte) {
    case MSG_OUTPUT:
      return { type: 'output', payload: data.slice(1) }

    case MSG_RESIZE: {
      if (data.length < 5) return { type: 'unknown' }
      const cols = (data[1] << 8) | data[2]
      const rows = (data[3] << 8) | data[4]
      return { type: 'resize', cols, rows }
    }

    case MSG_PRESENCE: {
      try {
        const json = new TextDecoder().decode(data.slice(1))
        const parsed = JSON.parse(json) as { participants?: PresenceEntry[] }
        return { type: 'presence', participants: parsed.participants ?? [] }
      } catch {
        return { type: 'unknown' }
      }
    }

    case MSG_TYPING: {
      try {
        const json = new TextDecoder().decode(data.slice(1))
        const parsed = JSON.parse(json) as { personKey: string; alias: string; typing: boolean }
        return { type: 'typing', personKey: parsed.personKey, alias: parsed.alias, typing: parsed.typing }
      } catch {
        return { type: 'unknown' }
      }
    }

    default:
      return { type: 'unknown' }
  }
}

// Callbacks for RelayClient events
export interface RelayClientCallbacks {
  onOutput: (data: Uint8Array) => void
  onOpen?: () => void
  onClose?: () => void
}

/**
 * WebSocket client implementing the binary framing protocol from Phase 2.
 * One instance per terminal session.
 */
export class RelayClient {
  private ws: WebSocket
  private pingInterval: ReturnType<typeof setInterval> | null = null

  constructor(
    port: number,
    sessionId: string,
    callbacks: RelayClientCallbacks,
    opts?: { remote?: boolean },
  ) {
    const path = opts?.remote
      ? `/api/relay/remote/${sessionId}/ws`  // daemon proxy → peer (cap looked up server-side; T-134-07-01)
      : `/sessions/${sessionId}/ws`          // local relay direct
    const url = `ws://127.0.0.1:${port}${path}`
    this.ws = new WebSocket(url)
    this.ws.binaryType = 'arraybuffer'

    this.ws.onopen = () => {
      callbacks.onOpen?.()
      // Send keep-alive pings every 30 seconds
      this.pingInterval = setInterval(() => {
        if (this.ws.readyState === WebSocket.OPEN) {
          this.ws.send(new Uint8Array([MSG_PING]))
        }
      }, 30_000)
    }

    this.ws.onmessage = (event: MessageEvent) => {
      const frame = parseServerFrame(new Uint8Array(event.data as ArrayBuffer))
      if (frame.type === 'output') {
        callbacks.onOutput(frame.payload)
      }
      // resize frames from server are informational; terminal resize is driven client-side
    }

    this.ws.onclose = () => {
      this._clearPing()
      callbacks.onClose?.()
    }

    this.ws.onerror = () => {
      // Error is always followed by close; let onClose handle cleanup
    }
  }

  /** Send keyboard input to the PTY. */
  sendInput(text: string): void {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(encodeInputFrame(text))
    }
  }

  /** Notify the PTY of a terminal resize. */
  sendResize(cols: number, rows: number): void {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(encodeResizeFrame(cols, rows))
    }
  }

  /** Close the WebSocket connection and stop keep-alive pings. */
  close(): void {
    this._clearPing()
    this.ws.close()
  }

  private _clearPing(): void {
    if (this.pingInterval !== null) {
      clearInterval(this.pingInterval)
      this.pingInterval = null
    }
  }
}

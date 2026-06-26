// Binary framing constants matching internal/relay/protocol.go
export const MSG_OUTPUT  = 0x01  // server → client: PTY output
export const MSG_RESIZE  = 0x02  // server → client: resize notification
export const MSG_INPUT   = 0x10  // client → server: keyboard input
export const MSG_RESIZE2 = 0x11  // client → server: resize request
export const MSG_PING    = 0x12  // client → server: keep-alive

// Chat/presence frame types — 0x30–0x3F range (Phase 152/154)
export const MSG_CHAT            = 0x30  // server → client: chat message broadcast
export const MSG_CHAT_SEND       = 0x31  // client → server: post chat message
export const MSG_PRESENCE        = 0x32  // server → client: full presence roster (JSON PresencePayload)
export const MSG_TYPING          = 0x33  // bidirectional: typing-start/stop (JSON TypingPayload)
export const MSG_ALIAS_SET       = 0x34  // client → server: set/update alias (JSON AliasPayload)
export const MSG_SESSION_INJECT  = 0x35  // client → server: inject text into PTY
export const MSG_INJECT_ERROR    = 0x36  // server → client: inject rejected (SEC-01 or oversize)

// PresenceEntry describes one participant in the presence roster.
// Field names match the Go json tags in internal/relay/protocol.go exactly.
export interface PresenceEntry {
  personKey: string  // TailnetID + ":" + origin — stable collapse key
  tailnetID: string  // "local" for desktop owner, node pubkey for web
  origin: string     // "local" (relay loopback) or "web" (webserver Tailscale)
  alias: string
  connCount: number  // active connections for this person key
}

// ChatMessage mirrors the Go ChatMessage struct json tags in internal/relay/protocol.go exactly.
// CRITICAL: AuthorAlias has json tag "alias" in Go — this field MUST be named `alias` here.
export interface ChatMessage {
  v: number             // schema version (always 1)
  id: string
  sessionID: string
  authorID: string      // "local" for desktop owner, node pubkey for web
  alias: string         // json:"alias" in Go (NOT authorAlias)
  content: string
  mentions?: string[]   // AuthorIDs of mentioned participants; omitted when empty
  sessionInject?: boolean
  ts: number            // UNIX milliseconds
}

// ServerFrame union type for parsed server messages
export type ServerFrame =
  | { type: 'output'; payload: Uint8Array }
  | { type: 'resize'; cols: number; rows: number }
  | { type: 'presence'; participants: PresenceEntry[] }
  | { type: 'typing'; personKey: string; alias: string; typing: boolean }
  | { type: 'chat'; message: ChatMessage }
  | { type: 'inject_error'; reason: string }
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
 * Encode a chat send request for the server.
 * Format: [MSG_CHAT_SEND, ...UTF-8 bytes of JSON {content}]
 */
export function encodeChatSendFrame(content: string): Uint8Array {
  const encoded = new TextEncoder().encode(JSON.stringify({ content }))
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_CHAT_SEND
  frame.set(encoded, 1)
  return frame
}

/**
 * Encode a session inject request for the server.
 * Format: [MSG_SESSION_INJECT, ...UTF-8 bytes of JSON {text}]
 */
export function encodeSessionInjectFrame(text: string): Uint8Array {
  const encoded = new TextEncoder().encode(JSON.stringify({ text }))
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_SESSION_INJECT
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

    case MSG_CHAT: {
      try {
        const json = new TextDecoder().decode(data.slice(1))
        const message = JSON.parse(json) as ChatMessage
        return { type: 'chat', message }
      } catch {
        return { type: 'unknown' }
      }
    }

    case MSG_INJECT_ERROR: {
      try {
        const json = new TextDecoder().decode(data.slice(1))
        const parsed = JSON.parse(json) as { reason: string }
        return { type: 'inject_error', reason: parsed.reason }
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
  onPresence?: (participants: PresenceEntry[]) => void
  onTyping?: (personKey: string, alias: string, typing: boolean) => void
  onChat?: (message: ChatMessage) => void
  onInjectError?: (reason: string) => void
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
    private callbacks: RelayClientCallbacks,
    opts?: { remote?: boolean; wsURL?: string },
  ) {
    // Phase 155: wsURL override short-circuits BEFORE port is used — port is 0
    // on web-share so it must never appear in the URL (Pitfall 1).
    let url: string
    if (opts?.wsURL) {
      url = opts.wsURL                          // web-share override (wss://host/...?cap=)
    } else {
      const path = opts?.remote
        ? `/api/relay/remote/${sessionId}/ws`   // daemon proxy → peer (cap looked up server-side; T-134-07-01)
        : `/sessions/${sessionId}/ws`           // local relay direct
      url = `ws://127.0.0.1:${port}${path}`
    }
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
      switch (frame.type) {
        case 'output':
          this.callbacks.onOutput(frame.payload)
          break
        case 'presence':
          this.callbacks.onPresence?.(frame.participants)
          break
        case 'typing':
          this.callbacks.onTyping?.(frame.personKey, frame.alias, frame.typing)
          break
        case 'chat':
          this.callbacks.onChat?.(frame.message)
          break
        case 'inject_error':
          this.callbacks.onInjectError?.(frame.reason)
          break
        // resize frames from server are informational; terminal resize is driven client-side
        // 'unknown' frames are silently dropped
      }
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

  /** Send a chat message to the relay. */
  sendChat(content: string): void {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(encodeChatSendFrame(content))
    }
  }

  /** Inject text into the PTY via the relay. */
  sendSessionInject(text: string): void {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(encodeSessionInjectFrame(text))
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

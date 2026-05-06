/**
 * urlSafety — Phase 95 (LNK-01, LNK-03) — pure security helpers for the
 * web-links addon click pipeline. NO network calls. NO logging. NO DOM.
 *
 * NOTE: TYPOSQUAT_LIST is best-effort, NOT a security boundary. The
 * confirmation popover surfaces the resolved URL even if the heuristic
 * misses; the user retains final say.
 */

/** Hardcoded scheme allowlist — LNK-01 boundary. NO user override. */
export const ALLOWED_SCHEMES = ['https:', 'http:', 'mailto:'] as const;

export type RiskKind = 'osc8' | 'idn' | 'typosquat';

/**
 * LNK-01: scheme allowlist. Defense-in-depth — caller (handler) also gates.
 * Returns false for any input that fails URL parsing or whose protocol is
 * not in the hardcoded ALLOWED_SCHEMES list.
 */
export function isAllowedScheme(href: string): boolean {
  try {
    const u = new URL(href);
    return (ALLOWED_SCHEMES as readonly string[]).includes(u.protocol);
  } catch {
    return false;
  }
}

/**
 * LNK-03: OSC 8 display-vs-href mismatch.
 * Strict: if display !== href textually, it's a mismatch by default. If both
 * parse as URLs, compare host + protocol. If display does NOT parse, treat
 * as plain prose ("click here") — mismatch IS the case.
 */
export function osc8Mismatch(displayText: string, href: string): boolean {
  if (displayText === href) return false;
  try {
    const dispUrl = new URL(displayText.trim());
    const hrefUrl = new URL(href);
    return dispUrl.host !== hrefUrl.host || dispUrl.protocol !== hrefUrl.protocol;
  } catch {
    // displayText is not a URL → mismatch (e.g. "click here" → evil.com)
    return true;
  }
}

/**
 * LNK-03: IDN/Punycode detection — both encoded form (xn--) and Unicode
 * form (non-ASCII codepoints in the host).
 *
 * Falls back to raw-href inspection if URL parsing throws (some platforms'
 * URL parsers reject punycode labels that don't decode cleanly — e.g.
 * Node's whatwg URL impl rejects 'xn--google-jzd.com'. We still want to
 * surface that as IDN for the popover.)
 */
export function hasIDN(href: string): boolean {
  try {
    const u = new URL(href);
    // WR-01: mailto URLs have no .hostname (WHATWG URL spec — mailto is
    // not a "special" scheme, so the host component is empty). Pull the
    // domain out of the pathname (RFC 6068: "mailto" "?" *( "&" hfield ))
    // and run the IDN check on it so a Cyrillic / xn-- mailto address
    // surfaces the popover too.
    if (u.protocol === 'mailto:') {
      const at = u.pathname.lastIndexOf('@');
      if (at < 0) return false;
      const domain = u.pathname.slice(at + 1);
      if (domain.toLowerCase().includes('xn--')) return true;
      if (/[^\x00-\x7F]/.test(domain)) return true;
      return false;
    }
    if (u.hostname.includes('xn--')) return true;
    if (/[^\x00-\x7F]/.test(u.hostname)) return true;
    return false;
  } catch {
    // Fallback: extract host portion textually and re-check.
    const m = /^[a-z][a-z0-9+.-]*:\/\/([^/?#]+)/i.exec(href);
    if (!m) return false;
    const host = m[1];
    if (host.toLowerCase().includes('xn--')) return true;
    if (/[^\x00-\x7F]/.test(host)) return true;
    return false;
  }
}

/**
 * LNK-03: Static typosquat heuristic. Best-effort — see file header.
 * Append entries via PR review, not via remote fetch.
 *
 * Rationale: this is a friction-on-suspicion list, NOT a security boundary.
 * The popover surfaces the URL regardless; the user retains final say.
 */
const TYPOSQUAT_LIST: ReadonlySet<string> = new Set([
  'paypa1.com', 'goog1e.com', 'arnazon.com', 'amaz0n.com',
  'microsft.com', 'micros0ft.com', 'app1e.com', 'git-hub.com',
  'tw1tter.com', 'twltter.com', 'face-book.com', 'faceb00k.com',
  'linked1n.com', 'linkedln.com', 'g00gle.com', 'goggle.com',
  'youtub3.com', 'reddlt.com', 'instagrarn.com', 'wlkipedia.org',
  'netfllx.com', 'spot1fy.com', 'dropb0x.com', 'aple.com',
  'rnicrosoft.com', 'arnzon.com', 'gocgle.com', 'githab.com',
  'gitlub.com', 'app1eid.com',
]);

/** LNK-03: typosquat lookup. www. prefix is stripped before comparison. */
export function isTypoSquat(href: string): boolean {
  try {
    const u = new URL(href);
    const host = u.hostname.toLowerCase().replace(/^www\./, '');
    return TYPOSQUAT_LIST.has(host);
  } catch {
    return false;
  }
}

/**
 * LNK-03: Risk prioritization. First match wins.
 * Order resolved by 95-RESEARCH §"Open Questions" item 5:
 *   osc8 (most informative — explicit deception)
 *     > idn (homograph spoofing class)
 *       > typosquat (heuristic class)
 *
 * Returns null when no risk detected — the caller should treat null as
 * "open without confirmation" (subject to scheme allowlist + modifier gate
 * which run upstream).
 */
export function getRisk(displayText: string, href: string): RiskKind | null {
  if (osc8Mismatch(displayText, href)) return 'osc8';
  if (hasIDN(href)) return 'idn';
  if (isTypoSquat(href)) return 'typosquat';
  return null;
}

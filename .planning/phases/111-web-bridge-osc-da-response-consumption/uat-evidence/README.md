# Phase 111 UAT evidence

This directory holds the two screenshots captured during the cross-surface
chafa parity UAT (WEB-02 release gate). See `../111-VERIFICATION.md`.

Expected files (committed by the macOS operator after sign-off):

- `web-chafa.png` — Chrome on localhost, post-`chafa --format=sixel` prompt
  state. Must be clean (no `10;rgb:...`, `11;rgb:...`, `?1;2c`, or
  `62;4;9;22c` leaked into the prompt line).
- `desktop-chafa.png` — Wails GUI, same shell session, same chafa command,
  same clean-prompt expectation.

Currently empty pending the operator UAT.

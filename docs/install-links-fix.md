# Fixing the Windows & Linux Install Links

**Scope:** The install commands shown on the app's Welcome screen for **Linux** and **Windows** don't work. This document captures everything that needs to happen to correct each one.

**Source of the displayed commands:** `frontend/src/components/WelcomeTab.tsx` (lines 36–47)

| OS | Currently displayed | Status |
|----|--------------------|--------|
| macOS | `brew tap scottkw/agenthub && brew install --cask agenthub` | ✅ Works (Homebrew tap auto-updated by `distribute.yml`) |
| Linux | `curl -fsSL https://agenthub.dev/install.sh | sh` | ❌ URL invalid — `install.sh` is not hosted anywhere |
| Windows | `winget install agenthub` | ❌ Wrong package id **and** package not yet in the winget catalog |

---

## 1. Linux — the curl URL is invalid

### Why it's broken
- The command points at `https://agenthub.dev/install.sh`.
- **There is no `install.sh` anywhere in the repo**, and nothing publishes one to `agenthub.dev`. The domain/path returns nothing, so `curl -fsSL` fails.
- Linux release artifacts *do* exist — `release.yml` produces:
  - `agenthub-v<VERSION>-linux-amd64.tar.gz`
  - `agenthub-v<VERSION>-linux-amd64.deb`
  - both hosted on GitHub Releases, with SHA256s in `checksums.txt`.

So the artifacts are real; only the installer entrypoint is missing.

### What needs to happen (decided: host the script from the repo source tree)

The script lives in the **code** section and is served via `raw.githubusercontent.com`; it pulls the **binary** from GitHub Releases at runtime. No per-release upkeep, one permanent URL, and the broken `agenthub.dev` dependency goes away.

1. Add a script at `scripts/install.sh` that:
   - Detects arch (currently only `amd64` is built — fail clearly on others, or build arm64 too).
   - Queries the GitHub Releases API for the latest tag
     (`https://api.github.com/repos/scottkw/agenthub/releases/latest`), or uses the
     `releases/latest/download/...` redirect.
   - Downloads `agenthub-v<VERSION>-linux-amd64.tar.gz`.
   - Verifies the SHA256 against `checksums.txt`.
   - Extracts the binary to `/usr/local/bin` (or `~/.local/bin`).
2. Update `WelcomeTab.tsx` line 42 to point at the raw GitHub URL:
   ```
   curl -fsSL https://raw.githubusercontent.com/scottkw/agenthub/main/scripts/install.sh | sh
   ```
3. Add the script to `TESTING.md` (repo convention: new behavior that can't be automated gets a manual M-NN checklist item; the script itself can have a shellcheck/CI test).

**Caveats for raw-from-`main` hosting:** the repo must stay public (it is, per the public release URLs), and `raw.githubusercontent.com` serves with a short (~5 min) cache — fine for an installer. Do **not** keep the `agenthub.dev/install.sh` URL unless that domain is actually stood up to serve the file.

### Verification
- `curl -fsSL <url> | sh` on a clean Linux box installs a runnable `agenthub` binary.
- SHA256 check inside the script passes against the published `checksums.txt`.

---

## 2. Windows — winget install won't work yet

There are **two separate problems**: a wrong command string, and an incomplete one-time catalog submission.

### Problem 2a — the displayed command uses the wrong id
- App shows `winget install agenthub`.
- The actual package identifier (per `packaging/winget/manifests/` and `distribute.yml`) is **`scottkw.agenthub`**.
- **Fix:** update `WelcomeTab.tsx` line 46 to:
  ```
  winget install scottkw.agenthub
  ```

### Problem 2b — the package is not in the winget catalog yet (this is the "something that still needs to happen")
The plumbing is built but the **first submission to `microsoft/winget-pkgs` has never been accepted**. Evidence in `.github/workflows/distribute.yml`:

- The `submit-winget` job carries `continue-on-error: true` with the comment:
  `# WinGet first submission pending — remove after accepted` (line 76)
- It branches on a repo variable `WINGET_FIRST_SUBMISSION` (line 108):
  - `true` → runs `wingetcreate new … --package-identifier scottkw.agenthub --submit` (creates the package entry)
  - `false` (default) → runs `wingetcreate update scottkw.agenthub …` (steady-state, **fails until the package exists**)

Until a `new` submission is merged into `microsoft/winget-pkgs`, `winget install scottkw.agenthub` returns "No package found."

### What needs to happen (one-time)
1. **Provision the token.** Ensure repo secret **`WINGET_TOKEN`** exists — a GitHub PAT (classic, `public_repo` scope) on an account that can fork `microsoft/winget-pkgs`. `wingetcreate --submit` opens the PR from that fork.
2. **Flip the first-submission flag.** Set repo variable **`WINGET_FIRST_SUBMISSION = true`** (Settings → Secrets and variables → Actions → Variables). This makes the job take the `wingetcreate new` path.
3. **Trigger a release** (or run `distribute.yml` via `workflow_dispatch` with the release `tag`). The job submits a PR to `microsoft/winget-pkgs` creating `scottkw.agenthub`.
4. **Get the PR merged.** Microsoft's validation pipeline + maintainers review it. The installer (`agenthub-v<VERSION>-windows-amd64-installer.exe`, NSIS) must pass automated validation (silent install/uninstall, etc.). Expect possible back-and-forth on the manifest.
5. **After acceptance, reset to steady state:**
   - Set **`WINGET_FIRST_SUBMISSION = false`** (or delete the variable — it defaults to `false`).
   - Remove `continue-on-error: true` from the `submit-winget` job (line 76) so future submission failures are surfaced instead of silently passing.
6. **Confirm:** on a Windows box, `winget install scottkw.agenthub` finds and installs the package. After that, every release auto-updates the manifest via the `update` path.

### Notes / gotchas
- `release.yml` must actually be producing `agenthub-v<VERSION>-windows-amd64-installer.exe` for the installer URL on line 116 to resolve — confirm the latest release has that asset before triggering the submission.
- The `submit-winget` job is skipped for `-rc` tags (line 77), so test the first submission with a real (non-rc) release tag.

---

## 3. Summary checklist

**Linux**
- [ ] Add `scripts/install.sh` (arch detect → fetch latest GitHub release tarball → verify SHA256 → install binary).
- [ ] Update `WelcomeTab.tsx:42` to the chosen URL (raw GitHub URL for Option A).
- [ ] Add a manual test item to `TESTING.md`.
- [ ] Verify end-to-end on a clean Linux box.

**Windows**
- [ ] Update `WelcomeTab.tsx:46` → `winget install scottkw.agenthub`.
- [ ] Confirm `WINGET_TOKEN` secret is set (PAT that can fork `microsoft/winget-pkgs`).
- [ ] Set repo variable `WINGET_FIRST_SUBMISSION = true`.
- [ ] Trigger `distribute.yml` (release or `workflow_dispatch`) → submits `wingetcreate new` PR.
- [ ] Shepherd the `microsoft/winget-pkgs` PR to merge.
- [ ] Reset `WINGET_FIRST_SUBMISSION = false` and remove `continue-on-error: true` from `submit-winget`.
- [ ] Verify `winget install scottkw.agenthub` on Windows.

---

## Bonus finding (not requested, but adjacent)
`WelcomeTab.tsx:54` shows the repo link as **`github.com/agenthub-dev/agenthub`**, but the real repo is **`github.com/scottkw/agenthub`** (per every release/winget URL in the workflows). The displayed link is wrong and should likely be `github.com/scottkw/agenthub`.

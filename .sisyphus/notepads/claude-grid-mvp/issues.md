# Issues — claude-grid-mvp

## Known Gotchas

- Terminal.app `bounds` format is `{left, top, right, bottom}` NOT `{x, y, width, height}`
- Warp has no AppleScript API — must use URI scheme + System Events
- System Events window positioning requires Accessibility permission
- AppleScript strings must be sanitized (escape `\` then `"`) to prevent injection
- Capture Terminal.app window IDs immediately after `do script` in same `tell` block

## Session Storage (Task 6)

- Session name generation uses `crypto/rand` with 2-byte (4 hex char) format: `grid-XXXX`
- Collision check implemented by checking file existence before returning name
- Store auto-creates `~/.claude-grid/sessions/` directory on first SaveSession call
- JSON serialization uses `json.MarshalIndent` for readable files
- ListSessions gracefully handles missing directory (returns empty slice)
- All file I/O errors wrapped with context for debugging
- Tests use `t.TempDir()` for isolation — no pollution of real filesystem

## [2026-02-17] Task 15: E2E Issues Found

### Warp Window Close Limitation (non-blocking)
- Warp does not support `close` message via System Events AppleScript
- Error: `window 1 of process "Warp" doesn't understand the "close" message. (-1708)`
- Impact: `kill` command warns but still deletes session file — windows remain open
- Workaround: Users must manually close Warp windows after `kill`
- Potential fix: Use `keystroke "w" using command down` via System Events instead of `close`

### Exit Code for Invalid Count (minor)
- `claude-grid 0` prints error but exits with code 0
- Should return non-zero exit code for scripting compatibility

## [2026-02-17] Task F2: Code Quality Review

### Minor Issues (non-blocking)
- `internal/screen/detect.go:22` — Non-idiomatic timeout: `10*1000*1000*1000` instead of `10*time.Second`. Works correctly but could import `time` package for clarity.
- Duplicate `ScreenInfo` types in `grid` and `screen` packages with manual field copy in `root.go:103-108`. Deliberate package independence tradeoff.

### Positive Findings
- Zero `fmt.Println` in production code — all output through `cmd.OutOrStdout()`/`cmd.ErrOrStderr()`
- Zero `interface{}` usage — all concrete types or well-defined interfaces
- Zero empty error checks — all `if err != nil` blocks take action
- Zero dead code — all functions used
- Zero AI slop — no excessive comments, no generic names, no over-abstraction
- 38 tests across 5 packages, all passing
## [2026-02-17] Task F4: Scope Fidelity Check

### Scope Violations (blocking)
- Task 9 (`internal/terminal/warp.go`) CloseSession ignores `sessionID` and closes all Warp windows; spec requires session-scoped best-effort close.
- Task 16 (`README.md`) exceeds 200-line limit (272 lines) and is missing required flags reference table.
- Task 17 (`.goreleaser.yml`) missing explicit manual homebrew-tap setup note; includes extra brew fields (`token`, `caveats`) outside strict task spec.

### Cross-Task Contamination
- Commit `948d81b` (Task 1 message) touched Task 2/4/5 files (`internal/grid/*`, `internal/screen/*`, `internal/terminal/backend.go`).
- Commit `88250af` touched `cmd/root.go` before Task 13 ownership window.
- Commit `64be8f5` (Task 13/14 wave commit) also touched Task 5 file `internal/terminal/backend.go`.

### Unaccounted Files
- CLEAN for scoped inventory (`**/*.go`, `**/*.md`, `Makefile`, `.github/**/*`, `.goreleaser.yml`).

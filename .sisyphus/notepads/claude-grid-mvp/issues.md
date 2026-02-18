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

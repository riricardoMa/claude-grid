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

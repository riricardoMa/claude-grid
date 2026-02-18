# Decisions — claude-grid-mvp

## Architecture Decisions

- **No CGO**: Using AppleScript `Finder` bounds for screen detection instead of CoreGraphics
- **Warp Strategy B**: Individual `warp://action/new_window` URIs + System Events tiling (no Launch Config YAML)
- **Uniform grid only**: count=5 → 3×2 with one empty cell (no uneven row expansion)
- **Spawn+tile atomic**: No separate `Tile()` method — tiling happens inside `SpawnWindows()`

## Implementation Decisions

- Module path: `github.com/riricardoMa/claude-grid`
- Test strategy: TDD (test-first)
- Backend auto-detect order: Warp (if installed) > Terminal.app
- Session names: `grid-<4 hex chars>` with collision check

# Learnings — claude-grid-mvp

## [2026-02-18T05:54] Session Start

Plan initialized. Wave 1 starting: 7 parallel tasks (scaffolding, grid calc, AppleScript utils, screen detection, backend interface, session storage, Makefile).


## AppleScript Executor Implementation (Task 3)

### Sanitization Pattern Verified
- Escaping order is CRITICAL: backslash first, then quotes
- Example: `\\"` → `\\\\\\\"` (not `\\\\\\"`)
- This prevents double-escaping attacks where attacker input contains backslashes

### Context-Based Execution Benefits
- Prevents hanging on permission dialogs (10s timeout default)
- Allows graceful cancellation from caller
- Respects context deadlines set by parent operations
- Enables testing without real AppleScript execution via mocking

### Interface Design for Testability
- ScriptExecutor interface allows MockExecutor in tests
- Backend can use OSAExecutor for real execution
- Decouples AppleScript dependency from business logic

### Error Handling Pattern
- Capture both stdout and stderr via CombinedOutput()
- Include output in error messages for debugging
- Wrap errors with context (osascript execution failed)

## [2026-02-17T22:00] Task 5: TerminalBackend Interface + Types

**Interface Design Pattern:**
- Small, focused interface (4 methods) for easy mocking and testing
- SpawnWindows returns []WindowInfo for tracking spawned windows
- SessionID in SpawnOptions enables session tracking and cleanup
- Context parameter allows timeout/cancellation control
- No SendKeys() or Tile() methods - kept minimal per Metis guardrails

**Key Decisions:**
- Spawn+tile atomic: No separate Tile() method, tiling happens inside SpawnWindows()
- DetectBackend skeleton prepared for future implementation (Warp > Terminal.app)
- Build constraint //go:build darwin ensures macOS-only compilation
- Types properly exported for use by backend implementations

**Type Structure:**
- SpawnOptions: Configuration for spawning (count, command, dir, grid, screen, bounds, sessionID)
- WindowInfo: Result of spawning (ID, Index, Backend name)
- TerminalBackend: Interface for implementations to satisfy

**Build Status:**
- go build ./internal/terminal/ ✓
- go vet ./internal/terminal/ ✓
- No circular dependencies

## [2026-02-17T21:56] Task 4: macOS Screen Detection via AppleScript (TDD)

### Finder Bounds Approach Validated
- AppleScript: `tell application "Finder" to get bounds of window of desktop`
- Returns format: `{left, top, right, bottom}` (absolute coordinates)
- Live test result: `0, 0, 1728, 1117` (usable area, menu bar + Dock already subtracted)
- This eliminates CGO entirely — no CoreGraphics dependency needed
- Same technique used by Rectangle/Magnet window managers

### Bounds Conversion Formula
- Input: `left, top, right, bottom` (4 integers)
- Output: `ScreenInfo{X: left, Y: top, Width: right-left, Height: bottom-top}`
- Critical: Width and Height are calculated, not provided by AppleScript
- Handles various input formats: with/without spaces, extra spaces

### TDD Implementation Pattern
- RED: 10 test cases covering happy path, edge cases, error handling
- GREEN: Minimal implementation with proper error messages
- Test cases:
  - Standard bounds parsing
  - Different origins (non-zero X, Y)
  - Format variations (no spaces, extra spaces)
  - Invalid inputs (wrong count, non-numeric)
  - Executor errors
  - Empty output
  - Bounds calculation verification

### Error Handling Strategy
- Descriptive errors for each failure mode
- Wraps executor errors with context
- Validates input count and format before parsing
- No panics on invalid input

### Build & Test Results
- `go test ./internal/screen/ -v` → PASS (all 10 test cases)
- `go build ./internal/screen/` → Success
- `go vet ./internal/screen/` → Clean
- Live AppleScript test → Returns valid bounds
- Evidence saved: `.sisyphus/evidence/task-4-*.txt`

### Integration Notes
- Depends on Task 3 (ScriptExecutor interface)
- Used by Task 13 (Root command) for screen detection
- Blocks Task 13 (Root command wiring)

## [2026-02-18T06:15] Task 2: Grid Layout Calculator (TDD) — COMPLETED

### Implementation Summary
- **RED**: Created `internal/grid/layout_test.go` with 11 table-driven test cases covering:
  - Grid calculations for count=1-3 (horizontal), count=4-16 (sqrt-based)
  - Window bounds calculation with pixel-accurate distribution
  - Edge case: count=5 in 2×3 grid (6 bounds with 1 empty cell)
  - ParseLayout validation (valid/invalid formats)

- **GREEN**: Implemented `internal/grid/layout.go` with:
  - `CalculateGrid(count int) GridLayout` — special cases 1-3, sqrt-based for rest
  - `CalculateWindowBounds(grid, screen, count) []WindowBounds` — pixel-perfect bounds
  - `ParseLayout(s string) (GridLayout, error)` — case-insensitive "RxC" parsing

### Algorithm Insights
**Grid Calculation Strategy** (after 5 iterations):
1. Special case: count ≤ 3 → 1×N (horizontal)
2. For count ≥ 4: Start from ceil(sqrt(count)) rows, iterate down to 2
3. **Tiebreaker logic** (critical for correctness):
   - **Tier 1**: Prefer perfect fit (0 wasted cells) over layouts with wasted cells
   - **Tier 2**: Among perfect fits, prefer closest to square (min cols-rows diff)
   - **Tier 3**: Among non-perfect fits, prefer closest to square first, then fewer wasted cells
4. Only consider layouts where cols ≥ rows (wider than tall)

**Why this works**:
- count=7: No perfect fit. 3×3 (diff=0, wasted=2) beats 2×4 (diff=2, wasted=1) ✓
- count=8: 2×4 (perfect fit, wasted=0) beats 3×3 (wasted=1) ✓
- count=12: Both 2×6 and 3×4 are perfect fits. 3×4 (diff=1) beats 2×6 (diff=4) ✓

### Window Bounds Distribution
- Divides screen into rows×cols cells
- Distributes remainder pixels to last row/col to avoid gaps
- Example: 2560×1575 in 2×2 grid:
  - Cell width: 1280, Cell height: 787
  - Last row gets +1 height (1575 % 2 = 1)
  - Result: 4 windows with no gaps or overlaps

### Test Coverage
- ✅ All 11 grid calculation cases pass
- ✅ Window bounds calculation (2×2 grid, 2560×1575 screen)
- ✅ Edge case: count=5 in 2×3 grid (6 bounds returned)
- ✅ ParseLayout: valid (2x3, 3X2) and invalid (abc, 0x1, -1x2)

### Files Created
- `internal/grid/layout.go` (140 lines)
- `internal/grid/layout_test.go` (180 lines)

### Commit
- `feat(grid): add layout calculator with TDD` (6faec28)

### Next Steps
- Task 3: AppleScript executor utility (depends on nothing)
- Task 4: Screen detection via AppleScript (depends on Task 3)
- Task 5: Backend interface + types (depends on nothing)
- Task 6: Session storage (depends on nothing)

## [2026-02-18T07:20] Task 9: Warp Backend (TDD)

### Warp URI Behavior
- `url.PathEscape()` encodes `/` as `%2F`; Warp URI tests expected slash-preserving paths
- Reliable pattern: `strings.ReplaceAll(url.PathEscape(dir), "%2F", "/")`
- Example: `/Users/bob/my project` -> `warp://action/new_window?path=/Users/bob/my%20project`

### Timing and Readiness
- Spawn sequence uses fixed per-window delay (500ms) plus extra 3s only when Warp was not already running
- Window creation is asynchronous; polling System Events count with exponential backoff avoids flakiness
- Backoff used: 100ms start, doubles to 1s max, 15s total timeout

### System Events Integration
- Warp has no direct AppleScript dictionary; all positioning/resize must route through System Events
- Tiling script must use 1-based window indexes (`window 1`, `window 2`, ...)
- Accessibility failures containing `not allowed assistive access` should be wrapped with explicit setup guidance

### Testability Pattern
- Warp backend is easiest to test by injecting function deps (`runOpen`, `statFn`, `sleepFn`, poll/tile hooks)
- ScriptExecutor mock captures script text for assertions on `count windows`, `set position`, `set size`, and `close window`
- Table-driven spawn scenarios kept coverage for success and open failure without running real `open` commands

## Task 10: Version Command

### Implementation Pattern
- Created `cmd/version.go` with `NewVersionCmd(version, commit, date string)` factory function
- Follows Cobra factory pattern established in Task 1
- Uses `fmt.Fprintf(cmd.OutOrStdout(), ...)` for testable output
- Imports `runtime` package for `GOOS` and `GOARCH`

### Output Format
- Format: `claude-grid <version> (<os>/<arch>) commit:<commit> built:<date>`
- Example with ldflags: `claude-grid v0.1.0 (darwin/arm64) commit:abc123 built:2026-02-17`
- Example without ldflags: `claude-grid dev (darwin/arm64) commit:unknown built:unknown`

### Integration
- Registered in `NewRootCommand()` via `cmd.AddCommand(NewVersionCmd(version, commit, date))`
- Version vars passed from `main.go` to root command, then to version command
- Maintains dependency injection pattern for testability

### ldflags Pattern (from Makefile)
- Format: `-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"`
- Works seamlessly with `go run` and `go build`
- Defaults in `main.go`: version="dev", commit="unknown", date="unknown"

### Testing
- ✓ `go run . version` → shows dev defaults
- ✓ `go run -ldflags "..." . version` → shows injected values
- ✓ `go build ./...` → compiles successfully
- ✓ `go vet ./cmd/...` → no issues

## [2026-02-17] Task 8: Terminal.app Backend (TDD)

### Spawn Script Construction
- Kept all window creation, ID capture, bounds positioning, and final ID return inside one `tell application "Terminal"` block to avoid race conditions.
- Captured ID immediately after each `do script` as `set windowIDN to id of front window`.
- Converted bounds from `WindowBounds{X,Y,Width,Height}` to Terminal format `{left, top, right, bottom}` using `right=X+Width`, `bottom=Y+Height`.

### AppleScript Safety
- Applied `script.SanitizeForAppleScript()` to both directory and command before interpolation.
- Built per-window command as `cd \"<dir>\" && <command>` when directory is provided.

### Close Behavior
- `CloseSession()` loads saved session windows and closes each by ID.
- Each close script wraps `close window id X` in `try/on error/end try` and ignores execution error so already-closed windows do not fail cleanup.

### Verification Notes
- RED confirmed first: tests failed on undefined `TerminalAppBackend` symbols before implementation.
- GREEN: `go test ./internal/terminal/ -v -run TestTerminalApp` passes with mock executor only.
- `go build ./internal/terminal/` passes.
- `go vet ./internal/terminal/` is currently blocked by pre-existing `warp_test.go` references to not-yet-implemented Warp symbols; `go vet -tests=false ./internal/terminal/` passes for non-test package code.

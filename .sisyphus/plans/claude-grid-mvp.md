# claude-grid MVP (v0.1) — Build Plan

## TL;DR

> **Quick Summary**: Build a Go CLI tool that spawns N terminal windows (Terminal.app or Warp) each running `claude`, and tiles them in an auto-calculated grid on macOS. Zero config, one command: `claude-grid 4`.
> 
> **Deliverables**:
> - `claude-grid` Go binary with cobra CLI
> - Terminal.app backend (AppleScript spawn + bounds positioning)
> - Warp backend (URI scheme spawn + System Events tiling)
> - Auto grid layout calculator (count → rows×cols)
> - macOS screen detection (AppleScript Finder bounds — no CGO)
> - Basic session tracking (`list`, `kill` commands)
> - CI/CD pipeline (GitHub Actions + goreleaser + Homebrew formula)
> - README with install + usage docs
> 
> **Estimated Effort**: Medium-Large (~2 weeks)
> **Parallel Execution**: YES — 5 waves
> **Critical Path**: Scaffolding → Interface+Types → Backends → Root Command → E2E QA

---

## Context

### Original Request
Build the Phase 1 MVP of `claude-grid` per `prd.md` — a lightweight CLI tool that spawns and visually tiles multiple Claude Code instances with a single command on macOS.

### Interview Summary
**Key Discussions**:
- **Scope**: Phase 1 MVP only. Phase 2 (prompts, tmux, config) and Phase 3 (broadcast, worktrees, Linux) are deferred.
- **Backends**: Terminal.app (AppleScript) + Warp (URI + System Events). No iTerm2, Kitty, or tmux for MVP.
- **Warp mode**: Windows only. Panes mode deferred.
- **Prompts**: `--prompt`, `--prompt-all` deferred to v0.2.
- **Session management**: Basic tracking only — auto-name, store handles, `list` + `kill` commands. No `resume`.
- **Test strategy**: TDD (test-first for all modules).
- **Distribution**: CI + goreleaser + Homebrew formula all included in MVP.
- **Module path**: `github.com/riricardoMa/claude-grid`

**Research Findings**:
- **Terminal.app**: `do script` creates window + runs command, `bounds` property (format: `{left, top, right, bottom}`) positions directly — simplest backend. `id of front window` gives stable window handle.
- **Warp**: No AppleScript API. `warp://action/new_window?path=<abs_path>` opens windows. System Events (`set position`, `set size`) tiles them. No reliable window ID API.
- **Screen detection**: `CGDisplayBounds` via CGO gives total display (not usable area). `tell application "Finder" to get bounds of window of desktop` returns usable area (menu bar + Dock already subtracted) — eliminates CGO entirely.
- **Cobra patterns**: Factory functions returning `*cobra.Command`, `PreRunE` for validation, `SilenceErrors: true`.
- **claude-squad**: Shell-aware `claude` detection (sources `.zshrc`), exponential backoff polling, JSON session storage at `~/.claude-squad/`.

### Metis Review
**Identified Gaps** (all addressed — see decisions below):

1. **CGO vs no-CGO**: Metis recommends eliminating CGO by using AppleScript Finder bounds for screen detection. This simplifies build, distribution, and `go install`. → **Applied as default.**
2. **Warp strategy**: Metis recommends Strategy B only (individual `warp://action/new_window` URIs) over Strategy A (Launch Config YAML). Avoids touching `~/.warp/launch_configurations/`, simpler, stateless. → **Applied as default.**
3. **Dead interface methods**: `SendKeys()` and separate `Tile()` not needed in MVP. → **Stripped.**
4. **Uneven grids**: Metis recommends uniform rectangular grid (e.g., 5 → 3×2 with one empty cell). → **Applied.**
5. **AppleScript injection**: All user strings must be sanitized before interpolation. → **Added as guardrail.**
6. **Stale sessions**: `list` must verify windows still exist (liveness check). → **Added to kill/list tasks.**
7. **Accessibility permissions**: System Events positioning requires Accessibility access. CLI must detect and guide user. → **Added to Warp backend task.**
8. **Window ordering**: Capture Terminal.app `id of front window` immediately after `do script`. → **Added to Terminal.app backend task.**
9. **Warp launch timing**: Exponential backoff polling after URI open to wait for windows. → **Added to Warp backend task.**
10. **Session name collisions**: Use `grid-<random4hex>` with collision check. → **Added to session task.**

---

## Work Objectives

### Core Objective
Deliver a working `claude-grid` binary that a macOS user can install and immediately run `claude-grid 4` to get 4 tiled terminal windows each running `claude`.

### Concrete Deliverables
- `claude-grid` Go binary (macOS arm64 + amd64 universal binary)
- Commands: `claude-grid <count>`, `claude-grid list`, `claude-grid kill <session>`, `claude-grid version`
- Flags: `--terminal terminal|warp`, `--dir <path>`, `--name <name>`, `--layout <RxC>`, `--verbose`, `--version`
- Terminal.app backend: spawn via AppleScript `do script`, position via `bounds`
- Warp backend: spawn via `warp://action/new_window` URI, position via System Events
- Session files at `~/.claude-grid/sessions/<name>.json`
- GitHub Actions CI (test + build), goreleaser config, Homebrew formula
- README with install instructions, usage examples, supported backends

### Definition of Done
- [ ] `go build ./...` succeeds with zero errors
- [ ] `go test ./...` passes all tests
- [ ] `go vet ./...` reports zero issues
- [ ] `claude-grid 2 --terminal terminal` spawns 2 Terminal.app windows tiled side-by-side
- [ ] `claude-grid 4 --terminal warp` spawns 4 Warp windows in 2×2 grid
- [ ] `claude-grid list` shows active sessions
- [ ] `claude-grid kill <session>` closes windows and removes session file
- [ ] `claude-grid version` outputs version string
- [ ] goreleaser builds universal macOS binary
- [ ] GitHub Actions CI runs tests on push

### Must Have
- Auto-detect screen usable area (minus menu bar + Dock)
- Auto-calculate grid layout (1→1×1, 2→1×2, 4→2×2, 6→3×2, 9→3×3)
- Terminal.app backend with reliable window ID tracking
- Warp backend with System Events tiling
- Session persistence for `kill` command
- `claude` binary detection in PATH (with clear error if missing)
- Backend auto-detection (Warp if installed, else Terminal.app)
- AppleScript timeout wrappers (10s default)
- Input sanitization for all user strings in AppleScript
- `//go:build darwin` on all macOS-specific files
- Exit code 0 on success, 1 on failure

### Must NOT Have (Guardrails)
- ❌ CGO dependency — use AppleScript Finder bounds for screen detection
- ❌ `SendKeys()` or separate `Tile()` interface methods — spawn+tile is atomic
- ❌ Uneven grid expansion (5→3+2 with different cell sizes) — use uniform rectangular grid
- ❌ `--program`, `--gap`, `--no-tile` flags
- ❌ `resume` command
- ❌ Per-instance prompts (`--prompt`, `--prompt-all`, `--prompts-file`)
- ❌ Config file support (`~/.claude-grid.toml`)
- ❌ Warp Launch Config YAML files (no touching `~/.warp/launch_configurations/`)
- ❌ Warp panes mode (`--mode panes`)
- ❌ iTerm2, Kitty, or tmux backends
- ❌ Emoji or ASCII art in CLI output — plain structured text only
- ❌ Multi-monitor support — use main display only
- ❌ Linux or Windows support
- ❌ Excessive comments, over-abstraction, generic variable names (`data`, `result`, `temp`)

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: NO (greenfield repo)
- **Automated tests**: YES — TDD (test-first)
- **Framework**: `go test` (standard library, no external test framework)
- **TDD flow**: Each task follows RED (failing test) → GREEN (minimal implementation) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

| Deliverable Type | Verification Tool | Method |
|------------------|-------------------|--------|
| Pure Go logic (grid, session) | Bash (`go test`) | Table-driven tests, assert exact outputs |
| AppleScript generation | Bash (`go test`) | Mock executor, verify generated scripts |
| Terminal backends (e2e) | Bash (`osascript`) | Spawn windows, query positions via AppleScript |
| CLI commands | Bash | Run binary, check stdout/stderr/exit code |
| Build/CI config | Bash | `go build`, `go vet`, goreleaser check |

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — 7 parallel, start immediately):
├── Task 1: Project scaffolding (go.mod, dirs, main.go) [quick]
├── Task 2: Grid layout calculator + tests (TDD) [quick]
├── Task 3: AppleScript executor utility + tests [quick]
├── Task 4: Screen detection via AppleScript + tests [quick]
├── Task 5: TerminalBackend interface + types [quick]
├── Task 6: Session storage + tests (TDD) [quick]
└── Task 7: Makefile [quick]

Wave 2 (Backends + CLI basics — 5 parallel):
├── Task 8: Terminal.app backend + tests (depends: 3, 5) [deep]
├── Task 9: Warp backend + tests (depends: 3, 5) [deep]
├── Task 10: Version command (depends: 1) [quick]
├── Task 11: List command (depends: 6) [quick]
└── Task 12: CI + goreleaser config (depends: 1, 7) [quick]

Wave 3 (Core CLI — 2 parallel):
├── Task 13: Root command wiring (depends: 2, 4, 6, 8, 9) [deep]
└── Task 14: Kill command (depends: 6, 8, 9) [unspecified-high]

Wave 4 (Polish — 3 parallel):
├── Task 15: End-to-end integration QA (depends: 13, 14) [unspecified-high]
├── Task 16: README (depends: 13) [writing]
└── Task 17: Homebrew formula in goreleaser (depends: 12) [quick]

Wave FINAL (Verification — 4 parallel):
├── F1: Plan compliance audit [oracle]
├── F2: Code quality review [unspecified-high]
├── F3: Real manual QA [unspecified-high]
└── F4: Scope fidelity check [deep]
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|------------|--------|------|
| 1. Scaffolding | — | 10, 12, 13 | 1 |
| 2. Grid layout | — | 13 | 1 |
| 3. AppleScript executor | — | 8, 9 | 1 |
| 4. Screen detection | — | 13 | 1 |
| 5. Backend interface | — | 8, 9 | 1 |
| 6. Session storage | — | 11, 13, 14 | 1 |
| 7. Makefile | — | 12 | 1 |
| 8. Terminal.app backend | 3, 5 | 13, 14 | 2 |
| 9. Warp backend | 3, 5 | 13, 14 | 2 |
| 10. Version command | 1 | — | 2 |
| 11. List command | 6 | — | 2 |
| 12. CI + goreleaser | 1, 7 | 17 | 2 |
| 13. Root command | 2, 4, 6, 8, 9 | 15, 16 | 3 |
| 14. Kill command | 6, 8, 9 | 15 | 3 |
| 15. E2E QA | 13, 14 | — | 4 |
| 16. README | 13 | — | 4 |
| 17. Homebrew formula | 12 | — | 4 |

### Agent Dispatch Summary

| Wave | # Parallel | Tasks → Agent Category |
|------|------------|----------------------|
| 1 | **7** | T1-T7 → `quick` |
| 2 | **5** | T8-T9 → `deep`, T10-T11 → `quick`, T12 → `quick` |
| 3 | **2** | T13 → `deep`, T14 → `unspecified-high` |
| 4 | **3** | T15 → `unspecified-high`, T16 → `writing`, T17 → `quick` |
| FINAL | **4** | F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep` |

---

## TODOs

- [x] 1. Project Scaffolding — Go Module, Directories, Entry Point

  **What to do**:
  - Initialize Go module: `go mod init github.com/riricardoMa/claude-grid`
  - Create directory structure per PRD section 6.2:
    ```
    cmd/           # cobra commands
    internal/
      grid/        # layout calculator
      screen/      # screen detection
      terminal/    # backend interface + implementations
      session/     # session tracking
    internal/script/  # AppleScript executor utility
    ```
  - Create `main.go` entry point with version vars (`version`, `commit`, `date`) and cobra root command execution
  - Create `cmd/root.go` with cobra root command skeleton: `Use: "claude-grid"`, `SilenceErrors: true`, `SilenceUsage: true`
  - Root command must accept `<count>` as first positional argument (via `Args` validation)
  - Add global persistent flags: `--verbose`, `--version`
  - Run `go mod tidy` to add cobra dependency
  - Verify: `go build ./...` succeeds

  **Must NOT do**:
  - Do not implement subcommands yet (list, kill, version come later)
  - Do not add any business logic — just the skeleton
  - Do not add config file parsing

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple scaffolding task — file creation, module init, no complex logic
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `git-master`: No git operations needed for this task

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4, 5, 6, 7)
  - **Blocks**: Tasks 10, 12, 13
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `prd.md:236-273` — Module architecture and directory structure from PRD
  - `prd.md:278-311` — Interface types that inform package layout

  **External References**:
  - cobra CLI framework: `github.com/spf13/cobra` — Use `cobra.Command` factory pattern (not `init()` globals). Root command should be created via `NewRootCmd()` function.
  - GitHub CLI root command pattern: Root cmd uses `SilenceErrors: true`, `SilenceUsage: true`, separates command creation from flag binding.

  **WHY Each Reference Matters**:
  - PRD directory structure ensures consistency with planned architecture
  - cobra factory pattern ensures commands are testable (can create command in test and verify flags/args)

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Project builds successfully
    Tool: Bash
    Preconditions: go 1.22+ installed
    Steps:
      1. Run `go build ./...` from project root
      2. Check exit code
    Expected Result: Exit code 0, no output to stderr
    Failure Indicators: Any compilation error
    Evidence: .sisyphus/evidence/task-1-build-success.txt

  Scenario: Root command shows help
    Tool: Bash
    Preconditions: Binary built successfully
    Steps:
      1. Run `go run . --help`
      2. Capture stdout
    Expected Result: Output contains "claude-grid" and "Usage"
    Failure Indicators: Panic, missing usage text
    Evidence: .sisyphus/evidence/task-1-help-output.txt

  Scenario: Directory structure exists
    Tool: Bash
    Preconditions: Scaffolding complete
    Steps:
      1. Run `ls -la cmd/ internal/grid/ internal/screen/ internal/terminal/ internal/session/ internal/script/`
      2. Verify all directories exist
    Expected Result: All 6 directories listed without errors
    Failure Indicators: "No such file or directory"
    Evidence: .sisyphus/evidence/task-1-dirs.txt
  ```

  **Commit**: YES
  - Message: `chore: scaffold project structure with go.mod and cobra`
  - Files: `go.mod, go.sum, main.go, cmd/root.go`
  - Pre-commit: `go build ./...`

---

- [x] 2. Grid Layout Calculator (TDD)

  **What to do**:
  - **RED**: Write `internal/grid/layout_test.go` FIRST with table-driven tests:
    ```go
    // Test cases (from PRD section 4.2 + 6.4):
    // count=1  → {Rows: 1, Cols: 1}
    // count=2  → {Rows: 1, Cols: 2}
    // count=3  → {Rows: 1, Cols: 3}
    // count=4  → {Rows: 2, Cols: 2}
    // count=5  → {Rows: 2, Cols: 3}  (uniform grid, 1 empty cell)
    // count=6  → {Rows: 2, Cols: 3}
    // count=7  → {Rows: 3, Cols: 3}  (uniform grid, 2 empty cells)
    // count=8  → {Rows: 2, Cols: 4}
    // count=9  → {Rows: 3, Cols: 3}
    // count=12 → {Rows: 3, Cols: 4}
    // count=16 → {Rows: 4, Cols: 4}
    ```
  - **RED**: Also test `CalculateWindowBounds(grid GridLayout, screen ScreenInfo) []WindowBounds`:
    ```go
    // Given 2×2 grid on 2560×1575 screen:
    // Window 0: {X: 0, Y: 0, Width: 1280, Height: 787}
    // Window 1: {X: 1280, Y: 0, Width: 1280, Height: 787}
    // Window 2: {X: 0, Y: 787, Width: 1280, Height: 788}
    // Window 3: {X: 1280, Y: 787, Width: 1280, Height: 788}
    ```
  - **RED**: Test edge case — count=5 in 2×3 grid produces 6 bounds (last one empty/unused)
  - **RED**: Test `ParseLayout("2x3")` for `--layout` flag parsing
  - **GREEN**: Implement `internal/grid/layout.go`:
    - `CalculateGrid(count int) GridLayout` — special cases for 1-3, sqrt-based for rest
    - `CalculateWindowBounds(grid GridLayout, screen ScreenInfo, count int) []WindowBounds` — pixel-accurate bounds per window
    - `ParseLayout(s string) (GridLayout, error)` — parse "RxC" or "RXC" format
  - **REFACTOR**: Ensure algorithm prefers wider windows (more cols than rows) since terminals are horizontal

  **Must NOT do**:
  - No uneven grid expansion (different cell sizes for last row)
  - No gap/spacing support (0px gap hardcoded)
  - No multi-monitor logic

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Pure algorithm with table-driven tests, no I/O or OS dependencies
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4, 5, 6, 7)
  - **Blocks**: Task 13
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `prd.md:84-93` — Expected grid mappings for each count
  - `prd.md:449-471` — CalculateGrid algorithm with special cases and sqrt formula

  **External References**:
  - Go table-driven test pattern: `func TestCalculateGrid(t *testing.T) { tests := []struct{...}; for _, tt := range tests { t.Run(tt.name, func(t *testing.T) {...}) } }`

  **WHY Each Reference Matters**:
  - PRD grid mappings are the authoritative spec — tests must match exactly
  - PRD algorithm is a starting point but the implementation should optimize for wider windows

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] Test file created: `internal/grid/layout_test.go`
  - [ ] `go test ./internal/grid/ -v` → PASS (all table-driven tests pass)

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Grid calculation matches PRD spec
    Tool: Bash (go test)
    Preconditions: Tests written and implementation complete
    Steps:
      1. Run `go test ./internal/grid/ -v -run TestCalculateGrid`
      2. Verify all test cases pass
    Expected Result: "PASS" with 0 failures. All count→grid mappings match PRD.
    Failure Indicators: Any "FAIL" output
    Evidence: .sisyphus/evidence/task-2-grid-calc.txt

  Scenario: Window bounds calculation is pixel-accurate
    Tool: Bash (go test)
    Preconditions: CalculateWindowBounds implemented
    Steps:
      1. Run `go test ./internal/grid/ -v -run TestCalculateWindowBounds`
      2. Verify bounds cover full screen area with no overlap and no gaps
    Expected Result: Sum of all window areas equals screen area. No bounds exceed screen dimensions.
    Failure Indicators: Off-by-one errors, bounds exceeding screen, gaps between windows
    Evidence: .sisyphus/evidence/task-2-window-bounds.txt

  Scenario: ParseLayout handles valid and invalid input
    Tool: Bash (go test)
    Preconditions: ParseLayout implemented
    Steps:
      1. Run `go test ./internal/grid/ -v -run TestParseLayout`
      2. Verify "2x3" → {2,3}, "3X2" → {3,2}, "abc" → error, "0x1" → error
    Expected Result: Valid layouts parsed, invalid layouts return descriptive error
    Failure Indicators: Panic on invalid input, wrong parsing
    Evidence: .sisyphus/evidence/task-2-parse-layout.txt
  ```

  **Commit**: YES
  - Message: `feat(grid): add layout calculator with TDD`
  - Files: `internal/grid/layout.go, internal/grid/layout_test.go`
  - Pre-commit: `go test ./internal/grid/`

---

- [x] 3. AppleScript Executor Utility (TDD)

  **What to do**:
  - **RED**: Write `internal/script/executor_test.go`:
    - Test `SanitizeForAppleScript(s string) string` — escapes `\` then `"` for safe interpolation
    - Test sanitization edge cases: paths with spaces, quotes, backslashes, dollar signs, semicolons
    - Test `Executor.RunAppleScript()` with mock (interface-based) — verify it calls `osascript -e <script>`
    - Test timeout behavior — verify context cancellation propagates
  - **GREEN**: Implement `internal/script/executor.go`:
    - `ScriptExecutor` interface: `RunAppleScript(ctx context.Context, script string) (string, error)`
    - `OSAExecutor` struct implementing `ScriptExecutor` — calls `osascript -e` via `exec.CommandContext`
    - `SanitizeForAppleScript(s string) string` — escapes backslashes then double quotes
    - Every `osascript` call gets `context.WithTimeout` (10s default)
    - Capture both stdout and stderr from osascript
    - Return descriptive errors including stderr content
  - **REFACTOR**: Ensure the interface is minimal and mockable for backend testing

  **Must NOT do**:
  - No JXA (JavaScript for Automation) support — AppleScript only
  - No file-based script execution — inline `-e` only for MVP

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small utility module, interface + one implementation, straightforward TDD
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4, 5, 6, 7)
  - **Blocks**: Tasks 8, 9
  - **Blocked By**: None

  **References**:

  **External References**:
  - Go `os/exec` with context: `exec.CommandContext(ctx, "osascript", "-e", script)` — always use context variant for timeout support
  - AppleScript string escaping: backslash first (`\` → `\\`), then double quote (`"` → `\"`)
  - Boba-CLI executor pattern: wraps osascript with `bytes.Buffer` for stdout/stderr capture

  **WHY Each Reference Matters**:
  - Context-based execution prevents AppleScript from hanging on permission dialogs
  - Proper escaping prevents injection attacks (Metis E3)
  - Interface allows mocking in backend tests without running real AppleScript

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] Test file created: `internal/script/executor_test.go`
  - [ ] `go test ./internal/script/ -v` → PASS

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: AppleScript string sanitization prevents injection
    Tool: Bash (go test)
    Preconditions: SanitizeForAppleScript implemented
    Steps:
      1. Run `go test ./internal/script/ -v -run TestSanitize`
      2. Test cases: `hello` → `hello`, `say "hi"` → `say \"hi\"`, `path\to` → `path\\to`, `/tmp/"; do shell script "bad"` → properly escaped
    Expected Result: All special characters escaped, no injection possible
    Failure Indicators: Unescaped quotes or backslashes
    Evidence: .sisyphus/evidence/task-3-sanitize.txt

  Scenario: Executor respects timeout
    Tool: Bash (go test)
    Preconditions: OSAExecutor with mock available
    Steps:
      1. Run `go test ./internal/script/ -v -run TestTimeout`
      2. Verify that a slow script gets cancelled by context
    Expected Result: Returns context.DeadlineExceeded error within timeout
    Failure Indicators: Test hangs or returns wrong error
    Evidence: .sisyphus/evidence/task-3-timeout.txt
  ```

  **Commit**: YES (groups with Task 4)
  - Message: `feat(script): add AppleScript executor with timeout and sanitization`
  - Files: `internal/script/executor.go, internal/script/executor_test.go`
  - Pre-commit: `go test ./internal/script/`

---

- [x] 4. macOS Screen Detection via AppleScript (TDD)

  **What to do**:
  - **RED**: Write `internal/screen/detect_test.go`:
    - Test `DetectScreen()` with mock ScriptExecutor:
      - Mock returns `"0, 25, 2560, 1575"` → `ScreenInfo{X: 0, Y: 25, Width: 2560, Height: 1550}`
      - Note: bounds format is `{left, top, right, bottom}`, so Width = right - left, Height = bottom - top
    - Test parsing of various Finder bounds formats
    - Test error handling when AppleScript fails
  - **GREEN**: Implement `internal/screen/detect.go` (with `//go:build darwin`):
    - `ScreenInfo` struct: `X, Y, Width, Height int`
    - `DetectScreen(executor script.ScriptExecutor) (ScreenInfo, error)`
    - Runs: `tell application "Finder" to get bounds of window of desktop`
    - Parses comma-separated response: `"left, top, right, bottom"`
    - Converts to ScreenInfo: Width = right - left, Height = bottom - top, X = left, Y = top
    - Returns descriptive error if parsing fails
  - **KEY VALIDATION**: The `Finder` bounds approach returns USABLE area (menu bar + Dock already subtracted). This eliminates CGO. If this doesn't work on the executor's machine, fall back to hardcoded defaults with a warning.
  - **REFACTOR**: Add `//go:build darwin` build tag

  **Must NOT do**:
  - No CGO, no CoreGraphics, no AppKit
  - No multi-monitor detection — main display only
  - No Dock position detection (Finder bounds already accounts for Dock)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small module, depends on script executor interface, straightforward parsing
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3, 5, 6, 7)
  - **Blocks**: Task 13
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `prd.md:477-484` — DetectScreenMacOS function signature and requirements

  **External References**:
  - AppleScript Finder bounds: `tell application "Finder" to get bounds of window of desktop` returns `{left, top, right, bottom}` representing usable screen area
  - This is the same technique Rectangle/Magnet use under the hood (Metis finding)

  **WHY Each Reference Matters**:
  - Finder bounds gives us usable area without CGO — the single most important architectural simplification in this project
  - PRD function signature guides the API design but we adapt to use AppleScript instead of CGO

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] Test file created: `internal/screen/detect_test.go`
  - [ ] `go test ./internal/screen/ -v` → PASS

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Screen detection parses Finder bounds correctly
    Tool: Bash (go test)
    Preconditions: Mock executor returning known bounds
    Steps:
      1. Run `go test ./internal/screen/ -v -run TestDetectScreen`
      2. Mock returns "0, 25, 2560, 1575"
    Expected Result: ScreenInfo{X: 0, Y: 25, Width: 2560, Height: 1550}
    Failure Indicators: Wrong width/height calculation, parsing error
    Evidence: .sisyphus/evidence/task-4-screen-detect.txt

  Scenario: Live screen detection returns valid dimensions
    Tool: Bash
    Preconditions: Running on macOS
    Steps:
      1. Run `osascript -e 'tell application "Finder" to get bounds of window of desktop'`
      2. Verify output is 4 comma-separated integers
      3. Verify width > 0 and height > 0
    Expected Result: Returns something like "0, 25, 2560, 1575" (actual values depend on machine)
    Failure Indicators: Error, empty output, non-numeric values
    Evidence: .sisyphus/evidence/task-4-live-screen.txt

  Scenario: Graceful error when AppleScript fails
    Tool: Bash (go test)
    Preconditions: Mock executor returning error
    Steps:
      1. Run `go test ./internal/screen/ -v -run TestDetectScreenError`
    Expected Result: Returns descriptive error, does not panic
    Failure Indicators: Panic, nil pointer dereference
    Evidence: .sisyphus/evidence/task-4-screen-error.txt
  ```

  **Commit**: YES (groups with Task 3)
  - Message: `feat(screen): add macOS screen detection via AppleScript Finder bounds`
  - Files: `internal/screen/detect.go, internal/screen/detect_test.go`
  - Pre-commit: `go test ./internal/screen/`

- [x] 5. TerminalBackend Interface + Types

  **What to do**:
  - Create `internal/terminal/backend.go` with `//go:build darwin`:
    ```go
    type TerminalBackend interface {
        Name() string
        Available() bool
        SpawnWindows(ctx context.Context, opts SpawnOptions) ([]WindowInfo, error)
        CloseSession(sessionID string) error
    }
    ```
  - Define `SpawnOptions` struct:
    ```go
    type SpawnOptions struct {
        Count      int
        Command    string       // "claude" by default
        Dir        string       // absolute working directory
        Grid       grid.GridLayout
        Screen     screen.ScreenInfo
        Bounds     []grid.WindowBounds  // pre-calculated per-window bounds
        SessionID  string       // for tracking
    }
    ```
  - Define `WindowInfo` struct:
    ```go
    type WindowInfo struct {
        ID       string  // Terminal.app window ID or Warp window index
        Index    int     // 0-based position in grid
        Backend  string  // "terminal" or "warp"
    }
    ```
  - Define `BackendRegistry` function: `DetectBackend(preferred string) (TerminalBackend, error)`
    - If `preferred` is set, use that backend (error if not available)
    - Auto-detection order: Warp (if `/Applications/Warp.app` exists) → Terminal.app (always available)
  - No implementation of backends — just interfaces and types

  **Must NOT do**:
  - No `SendKeys()` method — deferred to v0.2 with prompts
  - No separate `Tile()` method — tiling is part of `SpawnWindows`
  - No implementation of backends in this task

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Pure type definitions and interface — no logic to implement
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3, 4, 6, 7)
  - **Blocks**: Tasks 8, 9
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `prd.md:278-325` — TerminalBackend interface, SpawnOptions, GridLayout, ScreenInfo types from PRD. NOTE: Adapt these — PRD has `Tile()` and `SendKeys()` which we are NOT including. PRD has `CloseAll(sessionName)` which we rename to `CloseSession(sessionID)`.

  **External References**:
  - claude-squad pattern: Dependency injection via constructor (`NewTmuxSession(name, program)` returns interface impl)
  - Go interface best practice: Keep interfaces small (4 methods max), define at consumer site if possible

  **WHY Each Reference Matters**:
  - PRD types are the starting point but must be adapted per Metis guardrails (no SendKeys, no separate Tile)
  - Small interface = easy to implement, easy to test, easy to mock

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Types compile correctly
    Tool: Bash
    Preconditions: All type files created
    Steps:
      1. Run `go build ./internal/terminal/`
      2. Check exit code
    Expected Result: Exit code 0, types compile without errors
    Failure Indicators: Compilation errors, missing imports
    Evidence: .sisyphus/evidence/task-5-types-compile.txt

  Scenario: Interface is implementable
    Tool: Bash (go build)
    Preconditions: Interface defined
    Steps:
      1. Verify the interface has exactly 4 methods: Name, Available, SpawnWindows, CloseSession
      2. Run `go vet ./internal/terminal/`
    Expected Result: Clean compilation, no vet warnings
    Failure Indicators: Extra methods, circular imports
    Evidence: .sisyphus/evidence/task-5-interface.txt
  ```

  **Commit**: YES (groups with Task 6)
  - Message: `feat(core): add backend interface and session storage`
  - Files: `internal/terminal/backend.go`
  - Pre-commit: `go build ./internal/terminal/`

---

- [x] 6. Session Storage (TDD)

  **What to do**:
  - **RED**: Write `internal/session/store_test.go` FIRST:
    - Test `GenerateSessionName()` → returns `"grid-XXXX"` (4 random hex chars), no collision with existing sessions
    - Test `SaveSession(session Session) error` → creates `~/.claude-grid/sessions/<name>.json`
    - Test `LoadSession(name string) (Session, error)` → reads and unmarshals session file
    - Test `ListSessions() ([]Session, error)` → returns all sessions from directory
    - Test `DeleteSession(name string) error` → removes session file
    - Test directory auto-creation (first run, `~/.claude-grid/sessions/` doesn't exist)
    - Use `t.TempDir()` for test isolation (not real `~/.claude-grid/`)
  - **GREEN**: Implement `internal/session/store.go`:
    - `Session` struct:
      ```go
      type Session struct {
          Name      string       `json:"name"`
          Backend   string       `json:"backend"`  // "terminal" or "warp"
          Count     int          `json:"count"`
          Dir       string       `json:"dir"`
          CreatedAt time.Time    `json:"created_at"`
          Windows   []WindowRef  `json:"windows"`
      }
      type WindowRef struct {
          ID    string `json:"id"`     // Terminal.app window ID or Warp index
          Index int    `json:"index"`  // grid position
      }
      ```
    - `Store` struct with configurable base directory (default `~/.claude-grid/sessions/`)
    - Session name: `grid-<4 random hex chars>` with collision check
    - JSON serialization for session files
    - Auto-create directories on first write
  - **REFACTOR**: Ensure Store accepts base directory as parameter (for testability)

  **Must NOT do**:
  - No liveness checking in Store (that's the List command's job)
  - No session locking — keep it simple
  - No migration logic for schema changes

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard file-based CRUD with TDD, uses temp dirs for testing, no OS-specific logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3, 4, 5, 7)
  - **Blocks**: Tasks 11, 13, 14
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - claude-squad session storage: `~/.claude-squad/` with JSON DTO pattern — separate runtime struct from storage struct

  **External References**:
  - Go `t.TempDir()` for test isolation: creates per-test temp directory, auto-cleaned
  - `crypto/rand` for session name generation: `rand.Read(b)` + `hex.EncodeToString(b[:2])` for 4 hex chars

  **WHY Each Reference Matters**:
  - claude-squad pattern validates the approach (JSON files in home dir)
  - TempDir ensures tests don't pollute real file system

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] Test file created: `internal/session/store_test.go`
  - [ ] `go test ./internal/session/ -v` → PASS (all CRUD operations tested)

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Session CRUD lifecycle
    Tool: Bash (go test)
    Preconditions: Store implementation complete
    Steps:
      1. Run `go test ./internal/session/ -v -run TestSessionCRUD`
      2. Test: Create → Save → Load → verify fields match → List → verify appears → Delete → verify gone
    Expected Result: Full lifecycle works, all assertions pass
    Failure Indicators: File not created, JSON parse error, stale file after delete
    Evidence: .sisyphus/evidence/task-6-session-crud.txt

  Scenario: Session name generation is unique
    Tool: Bash (go test)
    Preconditions: GenerateSessionName implemented
    Steps:
      1. Run `go test ./internal/session/ -v -run TestGenerateSessionName`
      2. Generate 100 names, verify no duplicates
      3. Verify format matches `grid-[0-9a-f]{4}`
    Expected Result: 100 unique names, all matching expected pattern
    Failure Indicators: Duplicate names, wrong format
    Evidence: .sisyphus/evidence/task-6-session-name.txt

  Scenario: Auto-creates directory on first save
    Tool: Bash (go test)
    Preconditions: Empty temp directory
    Steps:
      1. Create Store pointing to non-existent directory
      2. Call SaveSession
      3. Verify directory was created and file exists
    Expected Result: Directory auto-created, session file written
    Failure Indicators: Error about missing directory
    Evidence: .sisyphus/evidence/task-6-auto-mkdir.txt
  ```

  **Commit**: YES (groups with Task 5)
  - Message: `feat(core): add backend interface and session storage`
  - Files: `internal/session/store.go, internal/session/store_test.go`
  - Pre-commit: `go test ./internal/session/`

---

- [x] 7. Makefile

  **What to do**:
  - Create `Makefile` with standard Go project targets:
    ```makefile
    BINARY_NAME=claude-grid
    VERSION?=dev
    COMMIT=$(shell git rev-parse --short HEAD)
    DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
    LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

    .PHONY: build test vet clean install

    build:
    	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

    test:
    	go test ./... -count=1 -v

    vet:
    	go vet ./...

    clean:
    	rm -rf bin/

    install:
    	go install $(LDFLAGS) .

    check: vet test build  # Full pre-commit check
    ```
  - Verify: `make build` produces binary at `bin/claude-grid`
  - Verify: `make test` runs all tests
  - Verify: `make vet` runs go vet

  **Must NOT do**:
  - No Docker targets
  - No release targets (goreleaser handles that)
  - No complex platform detection

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single file, standard Makefile targets, no complex logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3, 4, 5, 6)
  - **Blocks**: Task 12
  - **Blocked By**: None

  **References**:

  **External References**:
  - Standard Go Makefile pattern: ldflags for version injection, phony targets, separate build/test/vet
  - goreleaser uses same ldflags pattern for release builds

  **WHY Each Reference Matters**:
  - Consistent ldflags between Makefile and goreleaser ensures `claude-grid version` works in both dev and release

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Make build produces binary
    Tool: Bash
    Preconditions: Go source compiles (Task 1 done)
    Steps:
      1. Run `make build`
      2. Check `bin/claude-grid` exists
      3. Run `bin/claude-grid version` (or `--help` if version not yet wired)
    Expected Result: Binary exists and is executable
    Failure Indicators: Missing binary, permission denied
    Evidence: .sisyphus/evidence/task-7-make-build.txt

  Scenario: Make test runs all tests
    Tool: Bash
    Preconditions: At least one test file exists
    Steps:
      1. Run `make test`
      2. Verify output contains test results
    Expected Result: Exit code 0, shows PASS for all packages
    Failure Indicators: Test failures, missing packages
    Evidence: .sisyphus/evidence/task-7-make-test.txt
  ```

  **Commit**: YES
  - Message: `chore: add Makefile with build/test/install targets`
  - Files: `Makefile`
  - Pre-commit: `make build`

- [x] 8. Terminal.app Backend (TDD)

  **What to do**:
  - **RED**: Write `internal/terminal/terminal_app_test.go`:
    - Test `Available()` → returns true on macOS (Terminal.app always exists)
    - Test `SpawnWindows()` generates correct AppleScript:
      - For count=2: script should call `do script "claude"` twice in Terminal.app
      - Each window should get `set bounds of window id X to {left, top, right, bottom}`
      - Window ID should be captured via `id of front window` after each `do script`
    - Test AppleScript sanitization of `Dir` path (spaces, quotes)
    - Test `CloseSession()` generates correct `close window id X` AppleScript
    - Use MOCK ScriptExecutor (from Task 3's interface) — don't run real AppleScript in tests
  - **GREEN**: Implement `internal/terminal/terminal_app.go` (with `//go:build darwin`):
    - `TerminalAppBackend` struct with `script.ScriptExecutor` dependency
    - `Name()` → `"terminal"`
    - `Available()` → always true on macOS
    - `SpawnWindows()`:
      1. Build single AppleScript that creates all windows in one `tell application "Terminal"` block
      2. For each window: `do script "<command>"` then capture `id of front window`
      3. Immediately set bounds: `set bounds of window id <id> to {left, top, right, bottom}`
      4. Return `[]WindowInfo` with captured IDs
      5. **CRITICAL**: All `do script` + `set bounds` in ONE `tell` block to avoid window ordering issues
    - `CloseSession()`:
      1. Load session from store
      2. For each window ID: `close window id <id>` in Terminal.app
      3. Gracefully handle already-closed windows (try/catch in AppleScript)
  - **CRITICAL**: Terminal.app `bounds` format is `{left, top, right, bottom}` NOT `{x, y, width, height}`. Convert from WindowBounds: `{X, Y, X+Width, Y+Height}`.
  - **REFACTOR**: Extract AppleScript template strings as constants

  **Must NOT do**:
  - No `SendKeys` implementation
  - No tab creation (windows only)
  - No profile selection

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex AppleScript generation with edge cases (escaping, bounds conversion, window ID capture). Needs careful testing of script correctness.
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 9, 10, 11, 12)
  - **Blocks**: Tasks 13, 14
  - **Blocked By**: Tasks 3 (executor), 5 (interface)

  **References**:

  **Pattern References**:
  - `prd.md:278-301` — TerminalBackend interface (adapted per guardrails — no SendKeys/Tile)

  **External References**:
  - Terminal.app AppleScript: `do script "command"` creates new window, `id of front window` captures handle, `set bounds of window id X to {l, t, r, b}` positions it
  - Boba-CLI Terminal.app pattern: confirms `bounds` format is `{left, top, right, bottom}` (absolute coordinates, not width/height)
  - AppleScript try/on error: `try ... on error ... end try` for graceful handling of closed windows

  **WHY Each Reference Matters**:
  - Terminal.app bounds format is a CRITICAL detail — wrong format means windows pile up at origin
  - Boba-CLI is a production-tested reference for exactly this AppleScript pattern
  - Window ID capture in same tell block prevents race conditions (Metis E2)

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] Test file created: `internal/terminal/terminal_app_test.go`
  - [ ] `go test ./internal/terminal/ -v -run TestTerminalApp` → PASS

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Generated AppleScript is correct for 2 windows
    Tool: Bash (go test)
    Preconditions: Mock executor, 2-window spawn options with known bounds
    Steps:
      1. Run `go test ./internal/terminal/ -v -run TestTerminalAppSpawnScript`
      2. Capture the AppleScript string passed to mock executor
      3. Verify: contains 2x `do script`, 2x `set bounds`, bounds in {l,t,r,b} format
    Expected Result: Script contains correct Terminal.app AppleScript commands
    Failure Indicators: Wrong bounds format, missing window ID capture
    Evidence: .sisyphus/evidence/task-8-terminal-script.txt

  Scenario: Path with spaces is properly escaped
    Tool: Bash (go test)
    Preconditions: Dir set to `/Users/bob/my project`
    Steps:
      1. Run `go test ./internal/terminal/ -v -run TestTerminalAppEscaping`
      2. Verify generated script properly escapes the path
    Expected Result: Path appears as escaped string in AppleScript
    Failure Indicators: Unescaped quotes breaking AppleScript syntax
    Evidence: .sisyphus/evidence/task-8-terminal-escape.txt

  Scenario: CloseSession handles already-closed windows gracefully
    Tool: Bash (go test)
    Preconditions: Mock executor simulates AppleScript error for missing window
    Steps:
      1. Run `go test ./internal/terminal/ -v -run TestTerminalAppCloseGraceful`
      2. Mock returns error for one window, success for another
    Expected Result: Function completes without error, closes available windows
    Failure Indicators: Panic on missing window, early abort
    Evidence: .sisyphus/evidence/task-8-terminal-close.txt
  ```

  **Commit**: YES
  - Message: `feat(terminal): add Terminal.app backend with AppleScript`
  - Files: `internal/terminal/terminal_app.go, internal/terminal/terminal_app_test.go`
  - Pre-commit: `go test ./internal/terminal/ -run TestTerminalApp`

---

- [x] 9. Warp Backend (TDD)

  **What to do**:
  - **RED**: Write `internal/terminal/warp_test.go`:
    - Test `Available()` → checks `/Applications/Warp.app` exists
    - Test `SpawnWindows()` flow:
      1. Opens N windows via `open "warp://action/new_window?path=<abs_path>"` (test the URL construction)
      2. Waits for windows to appear (exponential backoff polling via System Events)
      3. Tiles windows via System Events `set position` + `set size`
    - Test URL encoding of paths with spaces
    - Test System Events AppleScript generation for tiling
    - Test `CloseSession()` via System Events `close window`
    - Use MOCK ScriptExecutor and MOCK for `exec.Command("open", ...)` — don't run real commands in tests
  - **GREEN**: Implement `internal/terminal/warp.go` (with `//go:build darwin`):
    - `WarpBackend` struct with `script.ScriptExecutor` and command runner dependencies
    - `Name()` → `"warp"`
    - `Available()` → checks `/Applications/Warp.app` exists via `os.Stat`
    - `SpawnWindows()`:
      1. For each window (0 to count-1):
         - Run `exec.Command("open", fmt.Sprintf("warp://action/new_window?path=%s", url.PathEscape(dir)))` 
         - Sleep 500ms between spawns (Warp needs time to create each window)
      2. Wait for all windows via exponential backoff:
         - Poll System Events: `tell application "System Events" to tell process "Warp" to count windows`
         - Start at 100ms, double up to 1s, timeout after 15s total
      3. Tile windows via System Events:
         - For each window by index: `set position of window <i> to {X, Y}` and `set size of window <i> to {W, H}`
         - **NOTE**: System Events window index is 1-based and ordered by z-order. Use reverse order (newest windows have highest index initially) or title matching.
      4. Return `[]WindowInfo` with index-based references (Warp has no stable window IDs)
    - `CloseSession()`:
      1. Close all Warp windows matching the session (by count, best-effort)
      2. Via System Events: `close window <i> of process "Warp"`
    - **Handle Warp not running**: If Warp isn't running, `open warp://` will launch it. Add extra initial delay (3-5s) if Warp process wasn't found before opening URI.
    - **Handle Accessibility permissions**: If System Events positioning fails, print clear error: `"⚠ Accessibility permission required. Go to System Settings → Privacy & Security → Accessibility and add your terminal app."`
  - **REFACTOR**: Extract System Events tiling into `internal/terminal/sysevents.go` helper (reusable for future backends)

  **Must NOT do**:
  - No Launch Config YAML files — individual URI scheme only
  - No panes mode
  - No `SendKeys`
  - No Warp Launch Configurations directory touching

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex multi-step flow (URI spawn → poll → System Events tile), timing-sensitive, accessibility permission handling, needs careful mock design
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8, 10, 11, 12)
  - **Blocks**: Tasks 13, 14
  - **Blocked By**: Tasks 3 (executor), 5 (interface)

  **References**:

  **Pattern References**:
  - `prd.md:328-447` — Warp backend deep dive, Strategy B (URI scheme), System Events tiling pattern
  - `prd.md:390-404` — System Events AppleScript example for setting position and size

  **External References**:
  - Warp URI scheme: `warp://action/new_window?path=<abs_path>` — opens new window at directory
  - System Events window positioning: `tell application "System Events" to tell process "Warp" to set position of window 1 to {X, Y}`
  - claude-squad exponential backoff: start 5ms → double → cap at 50ms, timeout 2s. Adapt to longer timeouts for Warp (slower window creation).
  - macOS Accessibility: System Events requires the calling terminal to have Accessibility permission. First use prompts a system dialog.

  **WHY Each Reference Matters**:
  - PRD Strategy B is the chosen approach (per Metis recommendation — simpler, no file system pollution)
  - System Events example is the exact AppleScript we need to generate
  - Exponential backoff is essential — Warp window creation is async and timing varies

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] Test file created: `internal/terminal/warp_test.go`
  - [ ] `go test ./internal/terminal/ -v -run TestWarp` → PASS

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Warp URI construction is correct
    Tool: Bash (go test)
    Preconditions: Mock command runner
    Steps:
      1. Run `go test ./internal/terminal/ -v -run TestWarpURIConstruction`
      2. Verify generated URI for path `/Users/bob/my project` is properly URL-encoded
    Expected Result: URI is `warp://action/new_window?path=/Users/bob/my%20project`
    Failure Indicators: Unencoded spaces, missing scheme
    Evidence: .sisyphus/evidence/task-9-warp-uri.txt

  Scenario: System Events tiling script is correct
    Tool: Bash (go test)
    Preconditions: Mock executor, 4 windows with known bounds
    Steps:
      1. Run `go test ./internal/terminal/ -v -run TestWarpTilingScript`
      2. Verify AppleScript sets position and size for each of 4 windows
    Expected Result: Script contains `set position of window X to {posX, posY}` and `set size of window X to {W, H}` for each window
    Failure Indicators: Missing windows, wrong bounds
    Evidence: .sisyphus/evidence/task-9-warp-tiling.txt

  Scenario: Exponential backoff waits for correct window count
    Tool: Bash (go test)
    Preconditions: Mock executor returns increasing window counts on successive polls
    Steps:
      1. Run `go test ./internal/terminal/ -v -run TestWarpBackoff`
      2. Mock returns 0, 1, 2, 4 on successive count queries
      3. Verify polling stops when count reaches target (4)
    Expected Result: Polling stops after mock returns 4, no timeout error
    Failure Indicators: Immediate timeout, infinite loop
    Evidence: .sisyphus/evidence/task-9-warp-backoff.txt

  Scenario: Accessibility error produces clear message
    Tool: Bash (go test)
    Preconditions: Mock executor returns System Events permission error
    Steps:
      1. Run `go test ./internal/terminal/ -v -run TestWarpAccessibilityError`
      2. Verify error message mentions "Accessibility permission"
    Expected Result: Error contains guidance about System Settings → Privacy → Accessibility
    Failure Indicators: Generic error without guidance
    Evidence: .sisyphus/evidence/task-9-warp-accessibility.txt
  ```

  **Commit**: YES
  - Message: `feat(terminal): add Warp backend with URI scheme and System Events tiling`
  - Files: `internal/terminal/warp.go, internal/terminal/warp_test.go, internal/terminal/sysevents.go`
  - Pre-commit: `go test ./internal/terminal/ -run TestWarp`

---

- [x] 10. Version Command

  **What to do**:
  - Create `cmd/version.go`:
    - Cobra subcommand: `claude-grid version`
    - Output format: `claude-grid <version> (<os>/<arch>) commit:<commit> built:<date>`
    - Version, commit, date injected via ldflags from `main.go` vars
    - Pass version info from root command to version subcommand via closure or struct
  - Register in root command: `rootCmd.AddCommand(NewVersionCmd(version, commit, date))`

  **Must NOT do**:
  - No update checking
  - No JSON output format

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Trivial single-file subcommand, no complex logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8, 9, 11, 12)
  - **Blocks**: None
  - **Blocked By**: Task 1 (scaffolding — root command must exist)

  **References**:

  **External References**:
  - Go `runtime.GOOS` + `runtime.GOARCH` for OS/arch info
  - Standard version output: `<name> <version> (<os>/<arch>)`

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Version command outputs correct format
    Tool: Bash
    Preconditions: Binary built with ldflags
    Steps:
      1. Run `go run -ldflags "-X main.version=v0.1.0 -X main.commit=abc123 -X main.date=2026-02-17" . version`
      2. Capture stdout
    Expected Result: Output contains "claude-grid v0.1.0" and "darwin" and "commit:abc123"
    Failure Indicators: Empty output, missing version info
    Evidence: .sisyphus/evidence/task-10-version.txt

  Scenario: Version command without ldflags shows dev
    Tool: Bash
    Preconditions: Binary built without ldflags
    Steps:
      1. Run `go run . version`
    Expected Result: Output contains "dev" as version
    Failure Indicators: Empty string or panic
    Evidence: .sisyphus/evidence/task-10-version-dev.txt
  ```

  **Commit**: YES (groups with Task 11)
  - Message: `feat(cli): add version and list commands`
  - Files: `cmd/version.go`
  - Pre-commit: `go build ./...`

---

- [x] 11. List Command

  **What to do**:
  - Create `cmd/list.go`:
    - Cobra subcommand: `claude-grid list`
    - Loads all sessions from store (`session.Store.ListSessions()`)
    - For each session, perform liveness check:
      - Terminal.app: `osascript -e 'tell application "Terminal" to get id of every window'` — check if session's window IDs still exist
      - Warp: `osascript -e 'tell application "System Events" to tell process "Warp" to count windows'` — basic check if Warp has any windows
    - Output table format:
      ```
      SESSION     BACKEND    WINDOWS  DIR                    CREATED
      grid-a3f2   terminal   4        ~/projects/my-app      2026-02-17 10:30
      grid-b1c4   warp       2        ~/projects/api         2026-02-17 11:15
      ```
    - If no sessions: `"No active sessions."`
    - Mark stale sessions (windows no longer exist) with `(stale)` indicator
  - Create helper to format table output (simple column alignment with `fmt.Fprintf` and `text/tabwriter`)

  **Must NOT do**:
  - No JSON output format
  - No auto-cleanup of stale sessions (just mark them)
  - No filtering or sorting options

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Straightforward command — load data, format table, print. Liveness check is simple AppleScript.
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8, 9, 10, 12)
  - **Blocks**: None
  - **Blocked By**: Task 6 (session storage)

  **References**:

  **Pattern References**:
  - `prd.md:137-138` — `claude-grid list` shows active sessions

  **External References**:
  - Go `text/tabwriter` for aligned table output: `w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)`

  **WHY Each Reference Matters**:
  - tabwriter ensures consistent column alignment regardless of data length

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: List shows active sessions in table format
    Tool: Bash
    Preconditions: At least one session file exists in test store
    Steps:
      1. Create test session file via Store.SaveSession
      2. Run list command (or test the formatting function)
      3. Verify output has header row and data row
    Expected Result: Table with SESSION, BACKEND, WINDOWS, DIR, CREATED columns
    Failure Indicators: Missing columns, misaligned output
    Evidence: .sisyphus/evidence/task-11-list-table.txt

  Scenario: List with no sessions shows empty message
    Tool: Bash
    Preconditions: Empty session directory
    Steps:
      1. Run list command with empty store
    Expected Result: Output is "No active sessions."
    Failure Indicators: Error, empty output, crash
    Evidence: .sisyphus/evidence/task-11-list-empty.txt
  ```

  **Commit**: YES (groups with Task 10)
  - Message: `feat(cli): add version and list commands`
  - Files: `cmd/list.go`
  - Pre-commit: `go build ./...`

---

- [x] 12. CI + goreleaser Config

  **What to do**:
  - Create `.github/workflows/ci.yml`:
    - Trigger: push to main, pull requests
    - Jobs:
      1. `test`: Run on `macos-latest` (need macOS for build tags), `go test ./... -count=1`
      2. `build`: Run `go build ./...`
      3. `vet`: Run `go vet ./...`
    - Use `actions/checkout@v4` + `actions/setup-go@v5` with Go 1.22+
    - Cache go modules via setup-go built-in caching
  - Create `.github/workflows/release.yml`:
    - Trigger: push tags `v*`
    - Job: Run goreleaser on `macos-latest`
    - Permissions: `contents: write`
  - Create `.goreleaser.yml`:
    - `version: 2`
    - Builds: macOS only (darwin), arm64 + amd64
    - `CGO_ENABLED=0` (no CGO — we use AppleScript for screen detection)
    - ldflags: `-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.CommitDate}}`
    - Universal binary: `universal_binaries` section for single fat binary
    - Archives: tar.gz with README + LICENSE
    - Checksum file
    - Changelog: auto from git, exclude docs/test/chore commits
    - Homebrew tap placeholder (separate repo `riricardoMa/homebrew-tap`)
  - Verify: `goreleaser check` passes (install goreleaser locally or just validate YAML structure)

  **Must NOT do**:
  - No Docker builds
  - No Linux or Windows targets
  - No NPM publishing
  - No code signing

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard CI/CD config files, well-documented patterns, no custom logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8, 9, 10, 11)
  - **Blocks**: Task 17
  - **Blocked By**: Tasks 1 (module path), 7 (Makefile)

  **References**:

  **External References**:
  - goreleaser v2 config: `version: 2`, `builds:` with `goos: [darwin]`, `goarch: [amd64, arm64]`
  - goreleaser universal binaries: `universal_binaries: [{id: claude-grid, replace: true}]`
  - GitHub Actions for Go: `actions/setup-go@v5` with `go-version: '1.22'`

  **WHY Each Reference Matters**:
  - goreleaser v2 syntax is required (v1 is deprecated)
  - Universal binary means one download for all Mac users (Intel + Apple Silicon)
  - macOS runner is required because `//go:build darwin` files won't compile on Linux

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: goreleaser config is valid
    Tool: Bash
    Preconditions: .goreleaser.yml created
    Steps:
      1. Run `goreleaser check` (or validate YAML structure manually)
      2. Verify no errors
    Expected Result: "config is valid" or equivalent success message
    Failure Indicators: Schema validation errors
    Evidence: .sisyphus/evidence/task-12-goreleaser-check.txt

  Scenario: CI workflow YAML is valid
    Tool: Bash
    Preconditions: .github/workflows/ files created
    Steps:
      1. Validate YAML syntax: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"`
      2. Verify trigger, jobs, steps structure
    Expected Result: YAML parses without error, contains expected job names
    Failure Indicators: YAML parse error, missing jobs
    Evidence: .sisyphus/evidence/task-12-ci-yaml.txt
  ```

  **Commit**: YES
  - Message: `ci: add GitHub Actions workflows and goreleaser config`
  - Files: `.github/workflows/ci.yml, .github/workflows/release.yml, .goreleaser.yml`
  - Pre-commit: `goreleaser check` (if available)

- [x] 13. Root Command — Wire Grid Spawning End-to-End

  **What to do**:
  - Rewrite `cmd/root.go` to wire all modules together:
    1. Parse `<count>` positional argument (validate: 1-16 range, error if 0 or >16)
    2. Parse flags:
       - `--terminal terminal|warp` (default: auto-detect)
       - `--dir <path>` (default: `$PWD`, resolve to absolute path)
       - `--name <name>` (default: auto-generated via session store)
       - `--layout <RxC>` (default: auto-calculate)
       - `--verbose` (default: false)
    3. Validate `claude` exists in PATH via `exec.LookPath("claude")`. On error: print `"'claude' not found in PATH. Install: npm install -g @anthropic-ai/claude-code\nOr specify a different location (v0.2)."` and exit 1.
    4. Detect screen via `screen.DetectScreen()`
    5. Calculate grid via `grid.CalculateGrid(count)` (or parse `--layout`)
    6. Calculate window bounds via `grid.CalculateWindowBounds(gridLayout, screenInfo, count)`
    7. Detect/select backend via `terminal.DetectBackend(preferred)`
    8. Build `SpawnOptions` from all the above
    9. Call `backend.SpawnWindows(ctx, opts)`
    10. Save session to store with window info
    11. Print summary:
        ```
        Detected: macOS, <backend> <version>, screen <W>x<H>
        Layout: <R>x<C> grid (<winW>x<winH> per window)
        Directory: <dir>
        Spawning <count> Claude Code instances...
        Session "<name>" created. Use `claude-grid kill <name>` to close all.
        ```
    12. Exit 0 on success
  - Error handling:
    - `claude` not found → exit 1 with install instructions
    - Backend not available → exit 1 with suggestion
    - Screen detection fails → warn and use fallback (1920×1080)
    - SpawnWindows fails → exit 1, attempt cleanup, print error
    - Count >16 → warn about readability, confirm or exit (for MVP: just warn and proceed)
  - **Minimum window size guard**: If calculated window size < 400×200, print warning but proceed

  **Must NOT do**:
  - No `--prompt` flag handling
  - No config file loading
  - No emoji in output — plain text only
  - No ASCII art grid visualization (PRD §8.1 shows one, but defer)
  - No interactive confirmation prompts (just warnings to stderr)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: This is the central orchestration — wires all modules together, needs careful error handling, flag validation, and correct flow. Critical path task.
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 14)
  - **Parallel Group**: Wave 3
  - **Blocks**: Tasks 15, 16
  - **Blocked By**: Tasks 2 (grid), 4 (screen), 6 (session), 8 (Terminal.app), 9 (Warp)

  **References**:

  **Pattern References**:
  - `prd.md:66-77` — One-command launch behavior description
  - `prd.md:525-539` — First-run UX example with detection output
  - `prd.md:555-567` — Error state examples (claude not found, backend not found, too many windows)
  - `prd.md:573-604` — CLI reference with all flags and their defaults

  **External References**:
  - Go `exec.LookPath("claude")` — returns full path or error if not in PATH
  - Go `os.Getwd()` for default working directory
  - Go `filepath.Abs(path)` to resolve relative paths
  - cobra `Args: cobra.ExactArgs(1)` or custom validator for count argument
  - cobra flag binding: `cmd.Flags().StringVarP(&terminal, "terminal", "t", "", "terminal backend")`

  **WHY Each Reference Matters**:
  - PRD examples define the exact UX contract — output messages must match or closely follow
  - PRD CLI reference is the canonical flag spec
  - LookPath for claude detection follows claude-squad's approach (simpler than shell-aware detection for MVP)

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] `go build -o /tmp/claude-grid .` succeeds
  - [ ] All flags parse correctly

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Spawn 2 Terminal.app windows tiled correctly
    Tool: Bash
    Preconditions: macOS with Terminal.app, claude in PATH
    Steps:
      1. Run `/tmp/claude-grid 2 --terminal terminal --name test-spawn`
      2. Wait 5 seconds
      3. Run `osascript -e 'tell application "Terminal" to get id of every window'`
      4. Verify at least 2 new windows exist
      5. Run `cat ~/.claude-grid/sessions/test-spawn.json`
      6. Verify session file exists with backend="terminal" and 2 windows
    Expected Result: 2 Terminal.app windows visible, session file saved
    Failure Indicators: No windows, wrong window count, missing session file
    Evidence: .sisyphus/evidence/task-13-spawn-terminal.txt

  Scenario: Error when claude not in PATH
    Tool: Bash
    Preconditions: Temporarily hide claude from PATH
    Steps:
      1. Run `PATH=/usr/bin /tmp/claude-grid 2 2>&1; echo "exit:$?"`
      2. Capture stderr and exit code
    Expected Result: Exit code 1, stderr contains "claude" and "not found" and install instruction
    Failure Indicators: Exit 0, no error message, panic
    Evidence: .sisyphus/evidence/task-13-no-claude.txt

  Scenario: Invalid count produces error
    Tool: Bash
    Preconditions: Binary built
    Steps:
      1. Run `/tmp/claude-grid 0 2>&1; echo "exit:$?"`
      2. Run `/tmp/claude-grid -1 2>&1; echo "exit:$?"`
      3. Run `/tmp/claude-grid abc 2>&1; echo "exit:$?"`
    Expected Result: All exit code 1, descriptive error messages
    Failure Indicators: Exit 0, panic, no error
    Evidence: .sisyphus/evidence/task-13-invalid-count.txt

  Scenario: Auto-detection selects correct backend
    Tool: Bash (go test)
    Preconditions: DetectBackend implemented
    Steps:
      1. Run `go test ./internal/terminal/ -v -run TestDetectBackend`
      2. On machine with Warp: returns warp backend
      3. On machine without Warp: returns terminal backend
    Expected Result: Correct backend selected based on available apps
    Failure Indicators: Wrong backend, nil backend
    Evidence: .sisyphus/evidence/task-13-autodetect.txt
  ```

  **Commit**: YES
  - Message: `feat(cli): wire root command for grid spawning end-to-end`
  - Files: `cmd/root.go`
  - Pre-commit: `go build -o /tmp/claude-grid . && /tmp/claude-grid --help`

---

- [x] 14. Kill Command

  **What to do**:
  - Create `cmd/kill.go`:
    - Cobra subcommand: `claude-grid kill <session-name>`
    - Validate: session name provided (cobra `Args: cobra.ExactArgs(1)`)
    - Flow:
      1. Load session from store by name
      2. If not found: exit 1 with `"Session '<name>' not found. Run 'claude-grid list' to see active sessions."`
      3. Determine backend from session's `Backend` field
      4. Call `backend.CloseSession(sessionID)` to close windows
      5. Delete session file from store
      6. Print: `"Session '<name>' killed. <N> windows closed."`
    - Gracefully handle partial failures (some windows already closed):
      - Close what you can, delete session file regardless
      - Print warning for windows that couldn't be closed
  - Wire into root command: `rootCmd.AddCommand(NewKillCmd(store, backends))`
  - The kill command needs access to both the session store AND the backend implementations to close windows

  **Must NOT do**:
  - No `kill --all` flag (single session at a time)
  - No force kill
  - No confirmation prompt

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Moderate complexity — session lookup, backend dispatch, graceful error handling. Needs both store and backend dependencies.
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 13)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 6 (session), 8 (Terminal.app backend), 9 (Warp backend)

  **References**:

  **Pattern References**:
  - `prd.md:139-140` — `claude-grid kill <name>` kills all windows in a session

  **WHY Each Reference Matters**:
  - PRD defines the command signature and expected behavior

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Kill removes session and closes windows
    Tool: Bash
    Preconditions: Active session "test-kill" exists with 2 Terminal.app windows
    Steps:
      1. Verify `~/.claude-grid/sessions/test-kill.json` exists
      2. Run `/tmp/claude-grid kill test-kill`
      3. Verify session file is deleted: `ls ~/.claude-grid/sessions/test-kill.json 2>&1`
      4. Verify output mentions windows closed
    Expected Result: Session file removed, windows closed, success message printed
    Failure Indicators: Session file still exists, error on kill
    Evidence: .sisyphus/evidence/task-14-kill-success.txt

  Scenario: Kill non-existent session shows error
    Tool: Bash
    Preconditions: No session "does-not-exist"
    Steps:
      1. Run `/tmp/claude-grid kill does-not-exist 2>&1; echo "exit:$?"`
    Expected Result: Exit code 1, error mentions "not found" and suggests `claude-grid list`
    Failure Indicators: Exit 0, panic, wrong error message
    Evidence: .sisyphus/evidence/task-14-kill-notfound.txt

  Scenario: Kill handles already-closed windows gracefully
    Tool: Bash
    Preconditions: Session file exists but windows were manually closed
    Steps:
      1. Create session, manually close its windows
      2. Run `/tmp/claude-grid kill <session>`
    Expected Result: Session file removed, warning about windows already closed, exit 0
    Failure Indicators: Exit 1, crash on missing windows
    Evidence: .sisyphus/evidence/task-14-kill-stale.txt
  ```

  **Commit**: YES
  - Message: `feat(cli): add kill command for session cleanup`
  - Files: `cmd/kill.go`
  - Pre-commit: `go build ./...`

---

- [ ] 15. End-to-End Integration QA

  **What to do**:
  - This is a VERIFICATION-ONLY task. No new code.
  - Build final binary: `go build -o /tmp/claude-grid .`
  - Execute complete user journey for BOTH backends:
  - **Terminal.app flow**:
    1. `claude-grid 4 --terminal terminal --name e2e-terminal`
    2. Verify 4 windows spawned and tiled in 2×2 grid
    3. Verify each window is running claude (or shows shell if claude not installed)
    4. `claude-grid list` — verify session appears
    5. `claude-grid kill e2e-terminal` — verify cleanup
  - **Warp flow** (if Warp installed):
    1. `claude-grid 2 --terminal warp --name e2e-warp`
    2. Verify 2 windows spawned side-by-side
    3. `claude-grid list` — verify session appears
    4. `claude-grid kill e2e-warp`
  - **Edge cases**:
    1. `claude-grid 1` — single fullscreen window
    2. `claude-grid 9 --layout 3x3` — manual layout override
    3. `claude-grid 0` — error
    4. `claude-grid version` — version output
    5. `claude-grid --help` — help text
  - Capture screenshots/evidence for each scenario

  **Must NOT do**:
  - No code changes — verification only
  - No modifying test infrastructure

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Requires running real commands on macOS, managing terminal windows, capturing evidence
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 16, 17)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Tasks 13, 14

  **References**:

  **Pattern References**:
  - `prd.md:525-567` — Expected first-run UX, prompt UX, error states

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Full Terminal.app lifecycle
    Tool: Bash
    Preconditions: Binary built, macOS, claude in PATH (or mock)
    Steps:
      1. /tmp/claude-grid 4 --terminal terminal --name e2e-term
      2. sleep 5
      3. osascript -e 'tell app "Terminal" to count windows' (verify >= 4)
      4. /tmp/claude-grid list (verify e2e-term appears)
      5. /tmp/claude-grid kill e2e-term
      6. Verify session file gone
    Expected Result: Complete lifecycle succeeds
    Evidence: .sisyphus/evidence/task-15-e2e-terminal.txt

  Scenario: Full Warp lifecycle (if installed)
    Tool: Bash
    Preconditions: Warp installed
    Steps:
      1. /tmp/claude-grid 2 --terminal warp --name e2e-warp
      2. sleep 8 (Warp slower to spawn)
      3. /tmp/claude-grid list (verify e2e-warp appears)
      4. /tmp/claude-grid kill e2e-warp
    Expected Result: Complete lifecycle succeeds
    Evidence: .sisyphus/evidence/task-15-e2e-warp.txt

  Scenario: Edge cases all handled
    Tool: Bash
    Preconditions: Binary built
    Steps:
      1. /tmp/claude-grid 1 --terminal terminal (single fullscreen)
      2. /tmp/claude-grid 0 2>&1 (error)
      3. /tmp/claude-grid version (version info)
      4. /tmp/claude-grid --help (usage)
      5. Cleanup: /tmp/claude-grid kill any-active-sessions
    Expected Result: count=1 works, count=0 errors, version shows info, help shows usage
    Evidence: .sisyphus/evidence/task-15-edge-cases.txt
  ```

  **Commit**: NO (verification only — evidence files not committed)

---

- [ ] 16. README

  **What to do**:
  - Rewrite `README.md` with:
    - **Title + tagline**: `claude-grid` — "One command. N Claude instances. Perfectly tiled."
    - **Quick demo**: Show the core command and expected output (text, not screenshot)
    - **Install section**:
      ```bash
      # Homebrew
      brew install riricardoMa/tap/claude-grid

      # Go install
      go install github.com/riricardoMa/claude-grid@latest

      # From source
      git clone https://github.com/riricardoMa/claude-grid.git
      cd claude-grid && make install
      ```
    - **Prerequisites**: macOS 12+, `claude` CLI installed
    - **Usage section**: Core commands with examples
      - `claude-grid <count>` — main command
      - `claude-grid list` — show sessions
      - `claude-grid kill <name>` — cleanup
      - `claude-grid version` — show version
    - **Flags reference table**: All flags with defaults
    - **Supported terminals**: Terminal.app (default), Warp
    - **Accessibility permissions note**: System Events requires Accessibility access for Warp tiling
    - **Contributing section**: `make check` for pre-commit, `make test` for tests
    - **License**: MIT
  - Keep it concise — under 200 lines

  **Must NOT do**:
  - No comparison table with competitors (save for website)
  - No architecture documentation
  - No detailed contributing guide
  - No badges (add later when CI is green)

  **Recommended Agent Profile**:
  - **Category**: `writing`
    - Reason: Documentation task — clear technical writing, no code
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 15, 17)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 13 (root command — need to know exact CLI interface)

  **References**:

  **Pattern References**:
  - `prd.md:37-47` — Product vision with CLI examples
  - `prd.md:525-567` — UX examples for README demo section
  - `prd.md:573-604` — CLI reference for flags table
  - `prd.md:489-505` — Installation methods

  **WHY Each Reference Matters**:
  - PRD examples should be adapted directly into README for consistency
  - CLI reference is the canonical source for flags documentation

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: README covers all essential sections
    Tool: Bash (grep)
    Preconditions: README.md written
    Steps:
      1. grep for key sections: "Install", "Usage", "Prerequisites", "License"
      2. Verify install commands include brew, go install, source
      3. Verify usage shows claude-grid <count>, list, kill, version
    Expected Result: All sections present with correct content
    Failure Indicators: Missing sections, wrong install commands
    Evidence: .sisyphus/evidence/task-16-readme-check.txt

  Scenario: Install commands are correct
    Tool: Bash
    Preconditions: README.md written
    Steps:
      1. Extract brew command from README
      2. Extract go install command from README
      3. Verify module path matches go.mod
    Expected Result: Module path is github.com/riricardoMa/claude-grid
    Failure Indicators: Wrong module path, typo in brew formula name
    Evidence: .sisyphus/evidence/task-16-readme-install.txt
  ```

  **Commit**: YES
  - Message: `docs: add README with install and usage instructions`
  - Files: `README.md`
  - Pre-commit: none

---

- [ ] 17. Homebrew Formula in goreleaser

  **What to do**:
  - Update `.goreleaser.yml` to include Homebrew tap configuration:
    ```yaml
    brews:
      - repository:
          owner: riricardoMa
          name: homebrew-tap
        homepage: "https://github.com/riricardoMa/claude-grid"
        description: "Spawn and tile multiple Claude Code instances with a single command"
        license: "MIT"
        install: |
          bin.install "claude-grid"
        test: |
          system "#{bin}/claude-grid", "version"
    ```
  - NOTE: This requires a separate `riricardoMa/homebrew-tap` repo to exist on GitHub. The goreleaser config DEFINES the formula; goreleaser PUBLISHES it to the tap repo during release.
  - If the tap repo doesn't exist yet, create a note in the plan that it needs to be created manually (one-time setup, just an empty repo with a README).
  - Verify: `goreleaser check` still passes with the added brew section

  **Must NOT do**:
  - No creating the homebrew-tap repo (out of scope — just configure goreleaser)
  - No cask formula (it's a CLI binary, use formula)
  - No custom download strategy

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small YAML addition to existing goreleaser config
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 15, 16)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 12 (goreleaser config must exist)

  **References**:

  **External References**:
  - goreleaser Homebrew: `brews:` section in `.goreleaser.yml` — auto-publishes formula to tap repo on tag release
  - Homebrew formula pattern: `install` block with `bin.install`, `test` block with version check

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: goreleaser config still valid with brew section
    Tool: Bash
    Preconditions: .goreleaser.yml updated with brews section
    Steps:
      1. Run `goreleaser check`
    Expected Result: Config validates successfully
    Failure Indicators: Schema errors in brews section
    Evidence: .sisyphus/evidence/task-17-goreleaser-brew.txt

  Scenario: Brew formula has correct metadata
    Tool: Bash (grep)
    Preconditions: .goreleaser.yml has brews section
    Steps:
      1. grep .goreleaser.yml for "claude-grid" in description
      2. grep for "riricardoMa/homebrew-tap" as repository
      3. grep for "MIT" license
    Expected Result: All metadata present and correct
    Failure Indicators: Wrong repo, missing license
    Evidence: .sisyphus/evidence/task-17-brew-metadata.txt
  ```

  **Commit**: YES
  - Message: `chore: add Homebrew formula to goreleaser config`
  - Files: `.goreleaser.yml`
  - Pre-commit: `goreleaser check`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `go build ./...` + `go test ./...`. Review all `.go` files for: `interface{}` abuse, empty error checks (`if err != nil { }` with no action), `fmt.Println` in production code (use structured output), dead code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state (`go build -o /tmp/claude-grid .`). Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration: spawn 4 windows, verify tiling, list sessions, kill session. Test edge cases: count=1, count=9, invalid count, missing claude binary. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual code. Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| After Task(s) | Message | Key Files | Verification |
|---------------|---------|-----------|--------------|
| 1 | `chore: scaffold project structure with go.mod and cobra` | go.mod, main.go, cmd/root.go | `go build ./...` |
| 2 | `feat(grid): add layout calculator with TDD` | internal/grid/ | `go test ./internal/grid/` |
| 3 | `feat(script): add AppleScript executor with timeout` | internal/script/ | `go test ./internal/script/` |
| 4 | `feat(screen): add macOS screen detection via AppleScript` | internal/screen/ | `go test ./internal/screen/` |
| 5, 6 | `feat(core): add backend interface and session storage` | internal/terminal/, internal/session/ | `go test ./internal/session/` |
| 7 | `chore: add Makefile with build/test/install targets` | Makefile | `make build` |
| 8 | `feat(terminal): add Terminal.app backend` | internal/terminal/terminal_app.go | `go test ./internal/terminal/` |
| 9 | `feat(terminal): add Warp backend` | internal/terminal/warp.go | `go test ./internal/terminal/` |
| 10, 11 | `feat(cli): add version and list commands` | cmd/version.go, cmd/list.go | `go build ./...` |
| 12 | `ci: add GitHub Actions and goreleaser config` | .github/, .goreleaser.yml | `goreleaser check` |
| 13 | `feat(cli): wire root command for grid spawning` | cmd/root.go | `go build -o /tmp/claude-grid .` |
| 14 | `feat(cli): add kill command` | cmd/kill.go | `go build ./...` |
| 15 | `test: end-to-end integration verification` | (evidence only) | QA pass |
| 16 | `docs: add README with install and usage` | README.md | — |
| 17 | `chore: add Homebrew formula to goreleaser` | .goreleaser.yml | `goreleaser check` |

---

## Success Criteria

### Verification Commands
```bash
# Build
go build -o /tmp/claude-grid .        # Expected: binary at /tmp/claude-grid
go vet ./...                           # Expected: zero output
go test ./... -count=1                 # Expected: all PASS

# CLI
/tmp/claude-grid version              # Expected: "claude-grid v0.1.0 (darwin/arm64)"
/tmp/claude-grid 0 2>&1; echo $?      # Expected: error message, exit 1
/tmp/claude-grid --help                # Expected: usage text with flags

# End-to-end (Terminal.app)
/tmp/claude-grid 2 --terminal terminal --name e2e-test
# Expected: 2 Terminal.app windows tiled side-by-side, each running claude
/tmp/claude-grid list                  # Expected: shows "e2e-test" session
/tmp/claude-grid kill e2e-test         # Expected: windows close, session removed

# End-to-end (Warp)
/tmp/claude-grid 4 --terminal warp --name warp-test
# Expected: 4 Warp windows in 2×2 grid
/tmp/claude-grid kill warp-test        # Expected: windows close

# Release
goreleaser check                       # Expected: config valid
```

### Final Checklist
- [ ] All "Must Have" items present and verified
- [ ] All "Must NOT Have" items absent (searched codebase)
- [ ] All `go test` pass
- [ ] `go vet` clean
- [ ] Binary builds for darwin/arm64 and darwin/amd64
- [ ] goreleaser config validates
- [ ] README has install instructions + usage examples

# Git Worktree Integration for claude-grid

## TL;DR

> **Quick Summary**: Add git worktree support so each spawned terminal window gets its own isolated working copy, preventing file write conflicts between parallel Claude Code instances. `claude-grid 3 --worktrees` creates 3 worktree branches with fun random names (e.g. `brave-fox-1`), each window `cd`s into its worktree, and `claude-grid clean <session>` removes them later.
> 
> **Deliverables**:
> - `internal/git/` package: worktree create/remove/list/prune + fun name generator
> - `--worktrees` and `--branch-prefix` CLI flags
> - Per-window directory support in SpawnOptions + both backends
> - Session lifecycle (active → stopped → deleted) with worktree tracking
> - `claude-grid clean <session>` subcommand
> - Updated `list` output showing session status
> 
> **Estimated Effort**: Medium (~3-4 days)
> **Parallel Execution**: YES — 4 waves
> **Critical Path**: Git module → SpawnOptions refactor → Root command wiring → Clean command + QA

---

## Context

### Original Request
Add git worktree integration (PRD section 5.4) to the existing claude-grid MVP. Each spawned instance gets its own worktree branch so multiple Claude Code agents don't conflict on file writes.

### Interview Summary
**Key Discussions**:
- **Cleanup on kill**: Preserve worktrees — `kill` only closes windows. Separate `claude-grid clean <session>` command removes worktrees.
- **Worktree location**: `~/.claude-grid/worktrees/{branch}_{timestamp_hex}`
- **Branch naming**: `{prefix}-{N}` (dash-separated, e.g. `sprint-42-1`, `sprint-42-2`)
- **Default prefix**: Fun adjective-noun with randomness (like Claude Code plan names: `brave-fox`, `swift-elk`)
- **Non-git repo**: Hard error and exit
- **Session lifecycle**: active → stopped (killed, worktrees preserved) → deleted (after clean)

**Research Findings**:
- claude-squad stores worktrees at `~/.claude-squad/worktrees/{branch}_{timestamp}`, creates from HEAD SHA using `git -C`
- `git worktree add -b <branch> <path> <HEAD-SHA>` ensures clean start without inheriting uncommitted changes
- `git worktree list --porcelain` is the right format for programmatic parsing
- Error handling: "already exists" vs "already checked out" are distinct errors requiring different messages
- Always `git worktree prune` after removal to clean `.git/worktrees/` metadata
- Never use `sh -c` with user input — always separate args to `exec.Command`

### Metis Review
**Identified Gaps** (all addressed):
1. **Branch "already checked out" error**: Distinct from "branch already exists" — another worktree is using the branch. Must detect and suggest `clean` command.
2. **Submodule warning**: Repos with submodules won't auto-initialize in worktrees. Must detect and warn.
3. **Dirty worktree on clean**: `git worktree remove` fails without `--force` if uncommitted changes exist. Must use `--force` and warn.
4. **Manually deleted worktree dirs**: `git worktree remove` fails if directory is gone. Must check existence first, fall back to `prune`.
5. **Error aggregation during cleanup**: Don't short-circuit — collect all errors and report together.
6. **Atomic rollback on spawn failure**: If window spawning fails after worktree creation, must clean up all worktrees via `defer`.
7. **Branch name sanitization**: Must validate `--branch-prefix` against git ref-name rules.

---

## Work Objectives

### Core Objective
Enable isolated git worktrees per spawned terminal window so parallel Claude Code instances don't conflict on file writes, with clean lifecycle management (create → use → preserve → clean).

### Concrete Deliverables
- `internal/git/worktree.go` — Worktree CRUD operations (create, remove, list, prune)
- `internal/git/names.go` — Fun adjective-noun name generator for default branch prefixes
- `internal/git/validate.go` — Branch name validation against git ref-name rules
- Updated `internal/terminal/backend.go` — `Dirs []string` in SpawnOptions for per-window dirs
- Updated `internal/terminal/terminal_app.go` — Per-window dir in AppleScript spawn
- Updated `internal/terminal/warp.go` — Per-window dir in URI construction
- Updated `internal/session/store.go` — Worktree refs + Status field in Session struct
- Updated `cmd/root.go` — `--worktrees` and `--branch-prefix` flags, worktree creation flow
- Updated `cmd/kill.go` — Preserve session when worktrees exist
- New `cmd/clean.go` — Remove worktrees + delete session
- Updated `cmd/list.go` — Show session status (active/stopped)

### Definition of Done
- [ ] `claude-grid 3 --worktrees` creates 3 worktrees with random names, each window opens in its worktree
- [ ] `claude-grid 3 --worktrees --branch-prefix sprint-42` creates sprint-42-1, sprint-42-2, sprint-42-3
- [ ] `claude-grid kill <session>` closes windows but preserves worktrees
- [ ] `claude-grid clean <session>` removes worktrees + deletes session
- [ ] `claude-grid list` shows active vs stopped sessions
- [ ] `go test ./...` passes with new worktree tests
- [ ] `go vet ./...` clean
- [ ] `go build ./...` succeeds

### Must Have
- Git repository detection (`git rev-parse --show-toplevel`)
- Worktree creation from HEAD SHA (clean start)
- Per-window directory isolation in both Terminal.app and Warp backends
- Session lifecycle: active → stopped → deleted
- Atomic rollback if spawn fails after worktree creation
- Error aggregation during cleanup (no short-circuit)
- Fun random name generator with adjective-noun pattern
- Branch name validation

### Must NOT Have (Guardrails)
- **No submodule auto-initialization** — detect and warn only
- **No go-git library** — pure os/exec + `git -C` (zero external deps beyond cobra)
- **No auto-commit on clean** — force-remove, warn about uncommitted changes
- **No worktree locking** — unnecessary complexity
- **No resume into existing worktrees** — deferred to future version
- **No changes to grid layout, screen detection, or tiling logic** — only touch dir handling
- **No changes to existing tests** unless directly affected by SpawnOptions changes

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (existing tests: layout_test.go, store_test.go, executor_test.go, terminal_app_test.go, warp_test.go)
- **Automated tests**: TDD (test-first for all new modules)
- **Framework**: `go test` (standard library)

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

| Deliverable Type | Verification Tool | Method |
|------------------|-------------------|--------|
| Git operations | Bash | Create temp repo, run worktree commands, verify branches/dirs |
| CLI flags | Bash | Run claude-grid with --worktrees flags, verify output |
| Session lifecycle | Bash | Create session → kill → verify preserved → clean → verify deleted |
| Backend integration | Bash (dry-run) | Verify SpawnOptions.Dirs populated correctly |

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — new modules + name generator):
├── Task 1: internal/git/worktree.go — Worktree CRUD operations [deep]
├── Task 2: internal/git/names.go — Fun adjective-noun name generator [quick]
├── Task 3: internal/git/validate.go — Branch name validation [quick]
├── Task 4: internal/session/store.go — Add Worktree refs + Status to Session [quick]

Wave 2 (After Wave 1 — backend refactoring):
├── Task 5: SpawnOptions + Terminal.app per-window dirs [unspecified-high]
├── Task 6: Warp backend per-window dirs [unspecified-high]

Wave 3 (After Waves 1+2 — command wiring):
├── Task 7: cmd/root.go — --worktrees and --branch-prefix flags + spawn flow [deep]
├── Task 8: cmd/kill.go — Preserve session when worktrees exist [quick]
├── Task 9: cmd/clean.go — New clean subcommand [unspecified-high]
├── Task 10: cmd/list.go — Show session status [quick]

Wave 4 (After Wave 3 — verification):
├── Task 11: Integration test — full lifecycle QA [deep]

Wave FINAL (After ALL — independent review):
├── Task F1: Plan compliance audit [oracle]
├── Task F2: Code quality review [unspecified-high]
├── Task F3: Real QA [unspecified-high]
├── Task F4: Scope fidelity check [deep]

Critical Path: Task 1 → Task 5 → Task 7 → Task 11 → F1-F4
Parallel Speedup: ~55% faster than sequential
Max Concurrent: 4 (Wave 1)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|------------|--------|------|
| 1 | — | 5, 6, 7, 9 | 1 |
| 2 | — | 7 | 1 |
| 3 | — | 7 | 1 |
| 4 | — | 7, 8, 9, 10 | 1 |
| 5 | 1 | 7 | 2 |
| 6 | 1 | 7 | 2 |
| 7 | 1, 2, 3, 4, 5, 6 | 11 | 3 |
| 8 | 4 | 11 | 3 |
| 9 | 1, 4 | 11 | 3 |
| 10 | 4 | 11 | 3 |
| 11 | 7, 8, 9, 10 | F1-F4 | 4 |

### Agent Dispatch Summary

| Wave | # Parallel | Tasks → Agent Category |
|------|------------|----------------------|
| 1 | **4** | T1 → `deep`, T2 → `quick`, T3 → `quick`, T4 → `quick` |
| 2 | **2** | T5 → `unspecified-high`, T6 → `unspecified-high` |
| 3 | **4** | T7 → `deep`, T8 → `quick`, T9 → `unspecified-high`, T10 → `quick` |
| 4 | **1** | T11 → `deep` |
| FINAL | **4** | F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep` |

---

## TODOs

- [x] 1. internal/git/worktree.go — Worktree CRUD Operations (TDD)

  **What to do**:
  - Create `internal/git/worktree.go` with a `Manager` struct wrapping git worktree operations
  - `Manager` fields: `repoPath string` (absolute path to git repo root), `worktreeBase string` (defaults to `~/.claude-grid/worktrees/`)
  - `NewManager(dir string) (*Manager, error)` — validates `dir` is inside a git repo via `git -C <dir> rev-parse --show-toplevel`, stores absolute repo root. Returns error `"not a git repository"` if not in a repo.
  - `func (m *Manager) CreateWorktree(branchName string) (worktreePath string, err error)` — runs `git -C <repoPath> rev-parse HEAD` to get HEAD SHA, then `git -C <repoPath> worktree add -b <branchName> <worktreePath> <headSHA>`. Worktree path is `~/.claude-grid/worktrees/<branchName>_<unix-nano-hex>`. Returns absolute worktree path.
  - Handle "already checked out" error: detect `"is already checked out"` in stderr, return descriptive error suggesting `claude-grid clean`.
  - Handle "branch already exists" error: detect `"already exists"` in stderr, return descriptive error.
  - `func (m *Manager) RemoveWorktree(worktreePath string) error` — checks `os.Stat(worktreePath)` first. If dir exists, runs `git -C <repoPath> worktree remove --force <worktreePath>`. If dir is gone, skips to prune. Calls `Prune()` after.
  - `func (m *Manager) Prune() error` — runs `git -C <repoPath> worktree prune`.
  - `func (m *Manager) DetectSubmodules() bool` — runs `git -C <repoPath> submodule status`, returns true if output is non-empty.
  - All git commands use `exec.Command("git", "-C", path, args...)` pattern — never `sh -c`, never `cmd.Dir`.
  - All commands use `cmd.CombinedOutput()` and include output in error messages.
  - Write `internal/git/worktree_test.go` using TDD:
    - Test `NewManager` with a temp git repo (use `git init` in `t.TempDir()`)
    - Test `NewManager` with a non-git directory → expect error
    - Test `CreateWorktree` → verify branch exists, worktree dir exists, worktree dir contains files from HEAD
    - Test `RemoveWorktree` → verify dir removed, `git worktree list` no longer shows it
    - Test `RemoveWorktree` with already-deleted dir → verify no error, prune called
    - Test `DetectSubmodules` with repo that has no submodules → false

  **Must NOT do**:
  - Do NOT use the go-git library — pure os/exec only
  - Do NOT auto-initialize submodules
  - Do NOT use `cmd.Dir` for git commands — always `git -C`
  - Do NOT handle resume/reattach to existing worktrees

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Core module with complex git interaction, error handling edge cases, and comprehensive TDD
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `git-master`: This is about programmatic git via Go exec, not interactive git operations

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: Tasks 5, 6, 7, 9
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References** (existing code to follow):
  - `internal/script/executor.go:17-41` — Pattern for wrapping external command execution with interface + concrete implementation. Follow the same testability pattern with exec functions.
  - `internal/session/store.go:36-48` — Pattern for `~/.claude-grid/` directory creation with `os.MkdirAll`. Use identical path resolution for worktree base.

  **API/Type References** (contracts to implement against):
  - `internal/terminal/backend.go:32-55` — SpawnOptions struct that will consume worktree paths. Manager.CreateWorktree must return paths compatible with SpawnOptions.Dir.

  **External References**:
  - claude-squad worktree implementation: `github.com/smtg-ai/claude-squad/session/git/worktree_ops.go` — CreateWorktree from HEAD SHA, error handling, cleanup with prune
  - Git worktree porcelain format: `git worktree list --porcelain` returns `worktree <path>\nHEAD <sha>\nbranch refs/heads/<name>`

  **WHY Each Reference Matters**:
  - executor.go shows how this codebase wraps external tools — follow same pattern for git commands
  - store.go shows how `~/.claude-grid/` paths are resolved — reuse the same home dir logic
  - backend.go shows the SpawnOptions contract — worktree paths feed into this

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test file created: `internal/git/worktree_test.go`
  - [ ] `go test ./internal/git/...` → PASS (6+ tests, 0 failures)

  **QA Scenarios:**

  ```
  Scenario: Create worktree from temp git repo
    Tool: Bash
    Preconditions: Create temp git repo with `git init` + `git commit --allow-empty -m "init"`
    Steps:
      1. Call NewManager(tempDir) — expect no error
      2. Call CreateWorktree("test-branch-1") — expect no error
      3. Run `git -C <tempDir> worktree list --porcelain` — expect output contains "test-branch-1"
      4. Verify returned path exists as directory: `test -d <returned-path>`
      5. Run `git -C <returned-path> branch --show-current` — expect "test-branch-1"
    Expected Result: Worktree created at ~/.claude-grid/worktrees/test-branch-1_<hex>, branch exists
    Failure Indicators: git worktree list doesn't show branch, directory doesn't exist
    Evidence: .sisyphus/evidence/task-1-create-worktree.txt

  Scenario: NewManager fails outside git repo
    Tool: Bash
    Preconditions: Create temp dir that is NOT a git repo
    Steps:
      1. Call NewManager(nonGitDir) — expect error
      2. Error message contains "not a git repository"
    Expected Result: Error returned, no panics
    Evidence: .sisyphus/evidence/task-1-non-git-error.txt

  Scenario: RemoveWorktree with deleted directory
    Tool: Bash
    Preconditions: Create worktree, then manually rm -rf the directory
    Steps:
      1. Create worktree with CreateWorktree("doomed-branch")
      2. rm -rf the returned path
      3. Call RemoveWorktree(path) — expect no error
      4. Run `git -C <repoPath> worktree list` — expect no stale entries
    Expected Result: No error, prune cleans up metadata
    Evidence: .sisyphus/evidence/task-1-remove-deleted-worktree.txt
  ```

  **Commit**: YES (groups with Tasks 2, 3)
  - Message: `feat(git): add worktree manager, name generator, and branch validation`
  - Files: `internal/git/worktree.go`, `internal/git/worktree_test.go`
  - Pre-commit: `go test ./internal/git/...`

- [x] 2. internal/git/names.go — Fun Adjective-Noun Name Generator (TDD)

  **What to do**:
  - Create `internal/git/names.go` with a `GenerateBranchPrefix() string` function
  - Returns a random adjective-noun pair like `brave-fox`, `swift-elk`, `calm-owl`, `bold-lynx`
  - Use two word lists: ~30 adjectives and ~30 nouns (animals). Keep words short (3-6 chars) for clean branch names.
  - Adjective examples: brave, swift, calm, bold, keen, warm, cool, neat, fair, wise, glad, soft, pure, wild, free, true, deep, high, rich, slim, rare, fast, safe, firm, mild, dark, pale, loud, sly, dry
  - Noun examples (animals): fox, elk, owl, lynx, wolf, bear, hawk, deer, crow, dove, hare, seal, wren, mink, frog, moth, newt, pike, colt, swan, lark, crab, mole, toad, wasp, goat, lamb, puma, orca, ibis
  - Use `crypto/rand` for randomness (not math/rand) to avoid seed issues
  - `GenerateBranchPrefix` returns a single adjective-noun string, e.g. `brave-fox`
  - The caller (root.go) will append `-N` for each window
  - Write `internal/git/names_test.go`:
    - Test that GenerateBranchPrefix returns non-empty string
    - Test that result matches pattern `^[a-z]+-[a-z]+$`
    - Test that calling it twice is likely to produce different results (call 10 times, expect ≥2 unique)

  **Must NOT do**:
  - Do NOT use math/rand (non-cryptographic, needs seeding)
  - Do NOT make the word lists configurable (overkill)
  - Do NOT add more than ~30 words per list (900 combinations is plenty)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple, self-contained function with two word lists and random selection
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4)
  - **Blocks**: Task 7
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/session/store.go:52-65` — GenerateSessionName pattern using `crypto/rand`. Follow identical randomness approach.

  **WHY Each Reference Matters**:
  - store.go shows the project's established pattern for random name generation with crypto/rand — match it exactly

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test file created: `internal/git/names_test.go`
  - [ ] `go test ./internal/git/...` → PASS (3+ tests for names)

  **QA Scenarios:**

  ```
  Scenario: Name generator produces valid git branch prefix
    Tool: Bash
    Preconditions: Build and run test
    Steps:
      1. Run `go test ./internal/git/... -run TestGenerateBranchPrefix -v`
      2. Verify test passes
      3. Verify output shows pattern like "brave-fox"
    Expected Result: All tests pass, names match ^[a-z]+-[a-z]+$ pattern
    Evidence: .sisyphus/evidence/task-2-name-generator.txt
  ```

  **Commit**: YES (groups with Tasks 1, 3)
  - Message: `feat(git): add worktree manager, name generator, and branch validation`
  - Files: `internal/git/names.go`, `internal/git/names_test.go`
  - Pre-commit: `go test ./internal/git/...`

- [x] 3. internal/git/validate.go — Branch Name Validation (TDD)

  **What to do**:
  - Create `internal/git/validate.go` with `ValidateBranchPrefix(prefix string) error`
  - Validate against git ref-name rules (see `git check-ref-format`):
    - No spaces
    - No `~`, `^`, `:`, `\`, `?`, `*`, `[`
    - No `..` (double dot)
    - No leading or trailing `.` or `-`
    - No leading `/`
    - No consecutive `/`
    - ASCII printable characters only
    - Non-empty
  - Return `nil` if valid, descriptive error listing invalid characters/patterns if not
  - Keep it simple — regex-based check is fine, no need to shell out to `git check-ref-format`
  - Write `internal/git/validate_test.go`:
    - Valid: `sprint-42`, `feature/auth`, `user-name`, `abc123`, `a`
    - Invalid: `bad name` (space), `bad~1` (tilde), `bad..name` (double dot), `.hidden` (leading dot), `` (empty), `bad[1]` (bracket)

  **Must NOT do**:
  - Do NOT shell out to `git check-ref-format` — pure Go validation
  - Do NOT over-engineer — regex-based check is sufficient

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single validation function with regex, straightforward TDD
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4)
  - **Blocks**: Task 7
  - **Blocked By**: None

  **References**:

  **External References**:
  - Git ref-name format rules: `git check-ref-format --help` — lists all forbidden patterns

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test file created: `internal/git/validate_test.go`
  - [ ] `go test ./internal/git/...` → PASS (8+ tests for validation)

  **QA Scenarios:**

  ```
  Scenario: Valid and invalid branch prefixes
    Tool: Bash
    Steps:
      1. Run `go test ./internal/git/... -run TestValidateBranchPrefix -v`
      2. Verify tests for valid prefixes pass (sprint-42, feature/auth, abc123)
      3. Verify tests for invalid prefixes return errors (spaces, tildes, double dots, empty)
    Expected Result: All tests pass
    Evidence: .sisyphus/evidence/task-3-branch-validation.txt
  ```

  **Commit**: YES (groups with Tasks 1, 2)
  - Message: `feat(git): add worktree manager, name generator, and branch validation`
  - Files: `internal/git/validate.go`, `internal/git/validate_test.go`
  - Pre-commit: `go test ./internal/git/...`

- [x] 4. internal/session/store.go — Add Worktree Refs + Status to Session (TDD)

  **What to do**:
  - Add `WorktreeRef` struct: `{ Path string, Branch string }` with JSON tags
  - Add fields to `Session` struct:
    - `Worktrees []WorktreeRef `json:"worktrees,omitempty"``
    - `Status string `json:"status"`` — values: `"active"` (default), `"stopped"` (killed with worktrees preserved)
    - `RepoPath string `json:"repo_path,omitempty"`` — original git repo path (needed for worktree cleanup)
  - Session status logic:
    - New sessions default to `Status: "active"`
    - When killed with worktrees: set to `Status: "stopped"`, keep session file
    - When cleaned: delete session file
  - Add `func (s *Store) UpdateSession(session Session) error` — overwrites existing session file (same as SaveSession but semantically distinct for updates)
  - Ensure backward compatibility: existing session files without `worktrees`, `status`, or `repo_path` fields load fine (JSON omitempty + zero values)
  - Update `store_test.go`:
    - Test saving/loading session with worktree refs
    - Test loading old-format session (no worktrees/status) → defaults work
    - Test UpdateSession overwrites correctly

  **Must NOT do**:
  - Do NOT change the session file location or naming scheme
  - Do NOT break existing session files (backward compat)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small additions to existing struct + one new method, straightforward
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3)
  - **Blocks**: Tasks 7, 8, 9, 10
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/session/store.go:14-21` — Existing Session struct to extend. Add new fields at the end.
  - `internal/session/store.go:67-85` — SaveSession pattern to follow for UpdateSession.
  - `internal/session/store_test.go` — Existing test patterns to follow.

  **WHY Each Reference Matters**:
  - store.go is the exact file being modified — existing struct and methods show the conventions
  - store_test.go shows test patterns (temp dirs, assertions) to match

  **Acceptance Criteria**:

  **TDD:**
  - [ ] `internal/session/store_test.go` updated with 3+ new tests
  - [ ] `go test ./internal/session/...` → PASS

  **QA Scenarios:**

  ```
  Scenario: Save and load session with worktree refs
    Tool: Bash
    Steps:
      1. Run `go test ./internal/session/... -run TestSessionWithWorktrees -v`
      2. Verify session round-trips with worktree refs intact
    Expected Result: Worktree paths and branches preserved after save/load
    Evidence: .sisyphus/evidence/task-4-session-worktrees.txt

  Scenario: Backward compatibility with old session format
    Tool: Bash
    Steps:
      1. Run `go test ./internal/session/... -run TestBackwardCompat -v`
      2. Verify old-format JSON (no worktrees/status fields) loads without error
      3. Verify Worktrees defaults to nil/empty, Status defaults to ""
    Expected Result: No errors loading old format sessions
    Evidence: .sisyphus/evidence/task-4-backward-compat.txt
  ```

  **Commit**: YES (solo)
  - Message: `feat(session): add worktree refs and status lifecycle to session store`
  - Files: `internal/session/store.go`, `internal/session/store_test.go`
  - Pre-commit: `go test ./internal/session/...`

- [x] 5. SpawnOptions + Terminal.app Backend — Per-Window Directories (TDD)

  **What to do**:
  - In `internal/terminal/backend.go`: Add `Dirs []string` field to `SpawnOptions` struct. Comment: `// Dirs is an optional list of per-window directories. If set, Dirs[i] is used for window i. If empty, Dir is used for all windows.`
  - In `internal/terminal/terminal_app.go`:
    - Change `buildSpawnScript` signature from `func buildSpawnScript(count int, dir string, command string, bounds []grid.WindowBounds) string` to `func buildSpawnScript(count int, dirs []string, command string, bounds []grid.WindowBounds) string`
    - Inside the loop: use `dirs[i]` instead of single `dir` for each window's `cd` command
    - Update `SpawnWindows` to prepare `dirs []string` from opts: if `opts.Dirs` is non-empty, use it; otherwise fill all entries with `opts.Dir`
  - Update `internal/terminal/terminal_app_test.go`:
    - Add test for `buildSpawnScript` with different dirs per window
    - Verify each AppleScript `do script` command uses the correct per-window dir
    - Existing tests should still pass (they use single Dir which maps to identical entries in Dirs)

  **Must NOT do**:
  - Do NOT change tiling/bounds logic — only dir handling
  - Do NOT change CloseSession logic
  - Do NOT break existing test cases

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Touches interface struct + backend implementation + tests, but well-scoped refactor
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Task 6)
  - **Blocks**: Task 7
  - **Blocked By**: Task 1 (needs git module to exist, though not directly imported — wave ordering ensures correctness)

  **References**:

  **Pattern References**:
  - `internal/terminal/backend.go:32-55` — SpawnOptions struct being extended. Add `Dirs` field after `Dir`.
  - `internal/terminal/terminal_app.go:96-128` — `buildSpawnScript` function being refactored. Line 105 is where `cd "<dir>"` is constructed per window — this becomes `cd "<dirs[i]>"`.
  - `internal/terminal/terminal_app.go:45-80` — `SpawnWindows` method where the dirs slice is prepared from opts.
  - `internal/terminal/terminal_app_test.go` — Existing tests to preserve. The test for `buildSpawnScript` likely passes a single dir — adapt to pass `[]string` with same dir repeated.

  **WHY Each Reference Matters**:
  - backend.go defines the shared SpawnOptions contract — Warp (Task 6) also reads this
  - terminal_app.go:96-128 is the exact function being refactored — per-window dir substitution happens in the loop on line 105
  - Tests must be preserved to avoid regressions

  **Acceptance Criteria**:

  **TDD:**
  - [ ] `internal/terminal/terminal_app_test.go` updated with per-window dir tests
  - [ ] `go test ./internal/terminal/...` → PASS (all existing + new tests)

  **QA Scenarios:**

  ```
  Scenario: Terminal.app AppleScript uses per-window directories
    Tool: Bash
    Steps:
      1. Run `go test ./internal/terminal/... -run TestBuildSpawnScriptPerWindowDirs -v`
      2. Verify generated AppleScript contains different `cd` paths for each window
      3. E.g. window 0: `cd "/path/worktree-1"`, window 1: `cd "/path/worktree-2"`
    Expected Result: Each window's AppleScript `do script` uses its specific directory
    Evidence: .sisyphus/evidence/task-5-terminal-per-window-dirs.txt

  Scenario: Backward compat — Dirs empty falls back to Dir
    Tool: Bash
    Steps:
      1. Run `go test ./internal/terminal/... -run TestBuildSpawnScript -v` (existing tests)
      2. Verify all existing tests still pass without modification
    Expected Result: Existing behavior unchanged when Dirs is nil/empty
    Evidence: .sisyphus/evidence/task-5-backward-compat.txt
  ```

  **Commit**: YES (groups with Task 6)
  - Message: `feat(terminal): support per-window directories in both backends`
  - Files: `internal/terminal/backend.go`, `internal/terminal/terminal_app.go`, `internal/terminal/terminal_app_test.go`
  - Pre-commit: `go test ./internal/terminal/...`

- [x] 6. Warp Backend — Per-Window Directories (TDD)

  **What to do**:
  - In `internal/terminal/warp.go`, modify `SpawnWindows`:
    - Prepare `dirs []string` from opts (same logic as Terminal.app: if `opts.Dirs` set, use it; otherwise fill with `opts.Dir`)
    - Change the spawn loop (lines 79-88) to construct a per-window URI: `encodedPath := strings.ReplaceAll(url.PathEscape(dirs[i]), "%2F", "/")` and `uri := fmt.Sprintf("warp://action/new_window?path=%s", encodedPath)` inside the loop
  - Update `internal/terminal/warp_test.go`:
    - Add test that verifies per-window URIs have different path parameters
    - Existing tests should still pass (single Dir fills all entries)

  **Must NOT do**:
  - Do NOT change tiling, window count polling, or sendCommandToWindows logic
  - Do NOT change CloseSession logic
  - Do NOT break existing test cases

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Similar scope to Task 5, touches Warp spawn loop + tests
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Task 5)
  - **Blocks**: Task 7
  - **Blocked By**: Task 1

  **References**:

  **Pattern References**:
  - `internal/terminal/warp.go:66-120` — `SpawnWindows` method. Lines 79-80 construct URI from single `opts.Dir`. Lines 81-88 loop opening that same URI. Change to per-window URI inside loop.
  - `internal/terminal/warp_test.go` — Existing tests to preserve.

  **WHY Each Reference Matters**:
  - warp.go:79-88 is the exact code being refactored — URI construction moves inside the loop with `dirs[i]`

  **Acceptance Criteria**:

  **TDD:**
  - [ ] `internal/terminal/warp_test.go` updated with per-window dir tests
  - [ ] `go test ./internal/terminal/...` → PASS

  **QA Scenarios:**

  ```
  Scenario: Warp URIs use per-window directories
    Tool: Bash
    Steps:
      1. Run `go test ./internal/terminal/... -run TestWarpPerWindowDirs -v`
      2. Verify the mock runOpen was called with different URIs for each window
      3. E.g. window 0: `warp://action/new_window?path=/worktree-1`, window 1: `warp://action/new_window?path=/worktree-2`
    Expected Result: Each Warp window opens with its specific directory in URI
    Evidence: .sisyphus/evidence/task-6-warp-per-window-dirs.txt
  ```

  **Commit**: YES (groups with Task 5)
  - Message: `feat(terminal): support per-window directories in both backends`
  - Files: `internal/terminal/warp.go`, `internal/terminal/warp_test.go`
  - Pre-commit: `go test ./internal/terminal/...`

- [x] 7. cmd/root.go — --worktrees and --branch-prefix Flags + Spawn Flow Wiring

  **What to do**:
  - Add two new flag variables: `worktreesFlag bool`, `branchPrefixFlag string`
  - Register flags: `--worktrees, -w` (bool), `--branch-prefix, -b` (string)
  - In `RunE`, after `resolvedDir` is set and before backend detection (around line 85):
    1. If `worktreesFlag` is set:
       a. Create `git.NewManager(resolvedDir)` — if error, print `"Error: --worktrees requires a git repository. %v"` and exit
       b. If `manager.DetectSubmodules()`, print warning to stderr: `"Warning: this repo uses submodules. Worktrees may be incomplete. Run 'git -C <worktree-path> submodule update --init' if needed."`
       c. Determine branch prefix: if `branchPrefixFlag` is set, validate it with `git.ValidateBranchPrefix(branchPrefixFlag)` → error on invalid. If not set, use `git.GenerateBranchPrefix()`.
       d. Create N worktrees in a loop: `manager.CreateWorktree(fmt.Sprintf("%s-%d", prefix, i+1))` for i in 0..count-1
       e. Collect worktree paths into `worktreeDirs []string`
       f. Set up `defer` for atomic rollback: if any subsequent step fails, remove all created worktrees
    2. Build SpawnOptions:
       - If worktrees created: set `Dirs: worktreeDirs` (per-window dirs)
       - If no worktrees: leave Dirs nil (existing behavior, uses Dir for all)
    3. After successful spawn + session save:
       - If worktrees: add `Worktrees: worktreeRefs`, `Status: "active"`, `RepoPath: manager.RepoPath()` to Session before saving
  - Add a public getter `func (m *Manager) RepoPath() string` to the git Manager if not already present

  **Must NOT do**:
  - Do NOT change grid calculation, screen detection, or tiling logic
  - Do NOT add worktree logic to the backend code — keep it in root.go only
  - Do NOT change how `--dir` flag works (worktrees are additive, not replacing)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Central wiring task touching multiple modules, needs careful error handling and defer rollback
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 8, 9, 10)
  - **Blocks**: Task 11
  - **Blocked By**: Tasks 1, 2, 3, 4, 5, 6 (all foundational work)

  **References**:

  **Pattern References**:
  - `cmd/root.go:22-28` — Existing flag variable declarations. Add `worktreesFlag` and `branchPrefixFlag` here.
  - `cmd/root.go:37-184` — RunE function where the entire spawn flow lives. Worktree logic inserts after line ~83 (resolvedDir set) and before line ~85 (screen detection).
  - `cmd/root.go:146-154` — SpawnOptions construction. Add `Dirs: worktreeDirs` here.
  - `cmd/root.go:168-175` — Session save. Add Worktrees, Status, RepoPath fields here.
  - `cmd/root.go:157-161` — Error handling after spawn failure. This is where the defer rollback should also trigger.
  - `cmd/root.go:187-192` — Existing flag registrations. Add `--worktrees` and `--branch-prefix` here.

  **API/Type References**:
  - `internal/git/worktree.go` — Manager.CreateWorktree, Manager.RemoveWorktree, Manager.DetectSubmodules, Manager.RepoPath (from Task 1)
  - `internal/git/names.go` — GenerateBranchPrefix (from Task 2)
  - `internal/git/validate.go` — ValidateBranchPrefix (from Task 3)
  - `internal/session/store.go` — Session struct with Worktrees, Status, RepoPath fields (from Task 4)
  - `internal/terminal/backend.go` — SpawnOptions.Dirs field (from Task 5)

  **WHY Each Reference Matters**:
  - root.go is the file being modified — exact line numbers show insertion points
  - git module provides the worktree operations this task wires together
  - session store provides the persistence fields this task populates

  **Acceptance Criteria**:

  - [x] `go build ./...` succeeds
  - [x] `go vet ./...` clean
  - [x] Flags `--worktrees` and `--branch-prefix` visible in `claude-grid --help`

  **QA Scenarios:**

  ```
  Scenario: --worktrees flag creates worktrees and passes per-window dirs
    Tool: Bash
    Preconditions: Have a git repo with at least one commit
    Steps:
      1. cd into git repo
      2. Run `claude-grid 2 --worktrees --branch-prefix test-wt --terminal terminal 2>&1` (may fail on window spawn in CI, but should get past worktree creation)
      3. Check `~/.claude-grid/worktrees/` for test-wt-1_* and test-wt-2_* directories
      4. Check `git worktree list` shows 2 new worktrees
    Expected Result: Worktrees created with correct branch names
    Failure Indicators: No worktree directories, git worktree list empty
    Evidence: .sisyphus/evidence/task-7-worktree-flag.txt

  Scenario: --worktrees outside git repo fails with clear error
    Tool: Bash
    Preconditions: cd into a non-git directory
    Steps:
      1. Run `claude-grid 2 --worktrees 2>&1`
      2. Assert exit code != 0
      3. Assert stderr contains "git repository"
    Expected Result: Clean error, no partial state
    Evidence: .sisyphus/evidence/task-7-non-git-error.txt

  Scenario: Invalid --branch-prefix rejected
    Tool: Bash
    Steps:
      1. Run `claude-grid 2 --worktrees --branch-prefix "bad name~1" 2>&1`
      2. Assert exit code != 0
      3. Assert stderr mentions invalid characters
    Expected Result: Error before any worktrees created
    Evidence: .sisyphus/evidence/task-7-invalid-prefix.txt
  ```

  **Commit**: YES (groups with Tasks 8, 9, 10)
  - Message: `feat(cli): add --worktrees flag, clean command, and session lifecycle`
  - Files: `cmd/root.go`
  - Pre-commit: `go build ./... && go vet ./...`

- [x] 8. cmd/kill.go — Preserve Session When Worktrees Exist

  **What to do**:
  - After closing windows (line 39-41), check if `sess.Worktrees` is non-empty:
    - If worktrees exist: instead of deleting session file, update session with `Status: "stopped"` and save back via `store.UpdateSession(sess)` (or `store.SaveSession` — same effect). Print: `"Session '%s' stopped. %d windows closed. Worktrees preserved.\nRun 'claude-grid clean %s' to remove worktrees."`
    - If no worktrees: existing behavior (delete session file)
  - This is a minimal change — just an if/else around the existing `store.DeleteSession` call

  **Must NOT do**:
  - Do NOT touch worktree removal logic — that's `clean`'s job
  - Do NOT change how windows are closed

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 10-line if/else addition to existing function
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 7, 9, 10)
  - **Blocks**: Task 11
  - **Blocked By**: Task 4

  **References**:

  **Pattern References**:
  - `cmd/kill.go:19-51` — The entire kill command. Lines 39-47 are the cleanup block being modified.
  - `internal/session/store.go` — Session struct with Worktrees and Status fields (from Task 4), SaveSession/UpdateSession methods.

  **WHY Each Reference Matters**:
  - kill.go is the exact file being modified — lines 39-47 show the existing cleanup flow

  **Acceptance Criteria**:

  - [x] `go build ./...` succeeds

  **QA Scenarios:**

  ```
  Scenario: Kill preserves worktrees and updates session status
    Tool: Bash
    Steps:
      1. Manually create a session JSON with Worktrees field populated
      2. Run `claude-grid kill <session>` (windows may already be closed, that's fine)
      3. Verify session file still exists at ~/.claude-grid/sessions/<session>.json
      4. Load session JSON and verify Status == "stopped"
    Expected Result: Session preserved with "stopped" status, worktree info intact
    Evidence: .sisyphus/evidence/task-8-kill-preserves-worktrees.txt

  Scenario: Kill without worktrees deletes session (backward compat)
    Tool: Bash
    Steps:
      1. Create session JSON without Worktrees field
      2. Run `claude-grid kill <session>`
      3. Verify session file is deleted
    Expected Result: Existing kill behavior unchanged for non-worktree sessions
    Evidence: .sisyphus/evidence/task-8-kill-backward-compat.txt
  ```

  **Commit**: YES (groups with Tasks 7, 9, 10)
  - Message: `feat(cli): add --worktrees flag, clean command, and session lifecycle`
  - Files: `cmd/kill.go`
  - Pre-commit: `go build ./...`

- [x] 9. cmd/clean.go — New Clean Subcommand

  **What to do**:
  - Create `cmd/clean.go` with a `NewCleanCmd(storePath string) *cobra.Command`
  - Usage: `claude-grid clean <session-name>`
  - Flow:
    1. Load session from store
    2. Verify session has worktrees (if not, error: `"Session '%s' has no worktrees to clean."`)
    3. Create `git.NewManager` using `sess.RepoPath`
    4. Loop through `sess.Worktrees`, call `manager.RemoveWorktree(wt.Path)` for each
    5. Use error aggregation: `var errs []error`. Continue on individual failures.
    6. For each dirty worktree (check `git -C <path> status --porcelain`), print warning: `"Warning: worktree '%s' had uncommitted changes (discarded)."`
    7. After all removals, call `manager.Prune()`
    8. Delete session file via `store.DeleteSession`
    9. Print summary: `"Session '%s' cleaned. %d worktrees removed."`
    10. If any errors, print them at end: `"Warnings during cleanup:\n  - %v"`
  - Add build tag `//go:build darwin` (matching other cmd files)
  - Register in `cmd/root.go` via `cmd.AddCommand(NewCleanCmd(""))` alongside other subcommands

  **Must NOT do**:
  - Do NOT auto-commit changes before removing worktrees
  - Do NOT delete branches — only remove worktrees (branches may be useful for review)
  - Do NOT close windows — that's kill's job. Clean assumes windows already closed.

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: New command file with git integration, error aggregation, multiple edge cases
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 7, 8, 10)
  - **Blocks**: Task 11
  - **Blocked By**: Tasks 1, 4

  **References**:

  **Pattern References**:
  - `cmd/kill.go:14-51` — Existing command pattern to follow. Same structure: NewXCmd, cobra.Command, Args ExactArgs(1), RunE.
  - `internal/git/worktree.go` — Manager.RemoveWorktree, Manager.Prune (from Task 1)
  - `internal/session/store.go` — Store.LoadSession, Store.DeleteSession, Session.Worktrees

  **WHY Each Reference Matters**:
  - kill.go is the template for new commands — follow identical patterns for consistency
  - git module provides the removal operations
  - session store provides the data source for what to clean

  **Acceptance Criteria**:

  - [ ] `go build ./...` succeeds
  - [ ] `claude-grid clean --help` shows usage

  **QA Scenarios:**

  ```
  Scenario: Clean removes worktrees and deletes session
    Tool: Bash
    Preconditions: Create temp git repo, create worktrees, save session with worktree refs
    Steps:
      1. Create worktree dirs and session JSON manually
      2. Run `claude-grid clean <session>`
      3. Verify worktree dirs removed
      4. Verify `git worktree list` shows no extra worktrees
      5. Verify session file deleted from ~/.claude-grid/sessions/
    Expected Result: Clean exit, all worktrees removed, session gone
    Evidence: .sisyphus/evidence/task-9-clean-command.txt

  Scenario: Clean with already-deleted worktree directory
    Tool: Bash
    Steps:
      1. Create session with worktree ref pointing to non-existent directory
      2. Run `claude-grid clean <session>`
      3. Verify no crash, session still cleaned up
    Expected Result: Graceful handling, prune called, session deleted
    Evidence: .sisyphus/evidence/task-9-clean-missing-dir.txt
  ```

  **Commit**: YES (groups with Tasks 7, 8, 10)
  - Message: `feat(cli): add --worktrees flag, clean command, and session lifecycle`
  - Files: `cmd/clean.go`, `cmd/root.go` (AddCommand line)
  - Pre-commit: `go build ./...`

- [x] 10. cmd/list.go — Show Session Status

  **What to do**:
  - In the `list` command's output, add a STATUS column showing `active` or `stopped`
  - Update the table header and row formatting to include status
  - For sessions without a Status field (backward compat), default to `"active"`
  - Format: `SESSION  STATUS  BACKEND  WINDOWS  DIR  CREATED`
  - Stopped sessions should be visually distinct — show `stopped` in the status column

  **Must NOT do**:
  - Do NOT add filtering (e.g. `--status active`) — keep it simple
  - Do NOT change the list loading logic

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single column addition to existing table output
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 7, 8, 9)
  - **Blocks**: Task 11
  - **Blocked By**: Task 4

  **References**:

  **Pattern References**:
  - `cmd/list.go` — The file being modified. Find the table formatting/printing code and add STATUS column.

  **Acceptance Criteria**:

  - [x] `go build ./...` succeeds

  **QA Scenarios:**

  ```
  Scenario: List shows status column
    Tool: Bash
    Steps:
      1. Create two session files: one with Status "active", one with Status "stopped"
      2. Run `claude-grid list`
      3. Verify output contains STATUS column
      4. Verify "active" and "stopped" appear in correct rows
    Expected Result: Both statuses displayed correctly
    Evidence: .sisyphus/evidence/task-10-list-status.txt
  ```

  **Commit**: YES (groups with Tasks 7, 8, 9)
  - Message: `feat(cli): add --worktrees flag, clean command, and session lifecycle`
  - Files: `cmd/list.go`
  - Pre-commit: `go build ./...`

- [x] 11. Integration Test — Full Worktree Lifecycle QA

  **What to do**:
  - Create `internal/git/integration_test.go` with build tag `//go:build integration`
  - Test the complete lifecycle programmatically (no actual terminal windows needed):
    1. Create temp git repo with initial commit
    2. Create Manager from temp repo
    3. Generate branch prefix via GenerateBranchPrefix
    4. Create 3 worktrees: `{prefix}-1`, `{prefix}-2`, `{prefix}-3`
    5. Verify each worktree dir exists, each branch exists in `git branch`, each worktree appears in `git worktree list`
    6. Make a change in one worktree (create a file), verify other worktrees don't see it (isolation)
    7. Remove all worktrees via RemoveWorktree
    8. Verify all dirs gone, `git worktree list` only shows main
    9. Verify branches still exist (we don't delete branches on clean)
  - Also test error paths:
    - CreateWorktree with branch that's already checked out → expect error
    - ValidateBranchPrefix with invalid input → expect error
    - NewManager on non-git dir → expect error
  - Run with `go test -tags integration ./internal/git/...`

  **Must NOT do**:
  - Do NOT spawn actual terminal windows — this is a programmatic integration test
  - Do NOT require any external setup beyond `git` being in PATH

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Comprehensive integration test covering full lifecycle + error paths
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (solo)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 7, 8, 9, 10

  **References**:

  **Pattern References**:
  - `internal/grid/layout_test.go` — Test patterns in this project (table-driven tests, t.TempDir)
  - `internal/session/store_test.go` — Test patterns with temp dirs and file I/O
  - All `internal/git/*.go` files — The code being tested

  **WHY Each Reference Matters**:
  - Existing test files show project conventions for assertions, temp dirs, error checking

  **Acceptance Criteria**:

  - [x] `go test -tags integration ./internal/git/...` → PASS (10+ tests)

  **QA Scenarios:**

  ```
  Scenario: Full worktree lifecycle
    Tool: Bash
    Steps:
      1. Run `go test -tags integration ./internal/git/... -v -run TestFullLifecycle`
      2. Verify creates 3 worktrees, verifies isolation, removes all, prunes
    Expected Result: All assertions pass, clean state at end
    Evidence: .sisyphus/evidence/task-11-integration-lifecycle.txt

  Scenario: Error path coverage
    Tool: Bash
    Steps:
      1. Run `go test -tags integration ./internal/git/... -v -run TestErrorPaths`
      2. Verify non-git dir, duplicate branch, invalid prefix all error correctly
    Expected Result: All error paths return expected errors without panics
    Evidence: .sisyphus/evidence/task-11-integration-errors.txt
  ```

  **Commit**: YES (solo)
  - Message: `test: add worktree integration tests`
  - Files: `internal/git/integration_test.go`
  - Pre-commit: `go test -tags integration ./internal/git/...`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `go test ./...` + `go build ./...`. Review all changed/new files for: empty catches, unused imports, hardcoded paths. Check AI slop: excessive comments, over-abstraction, generic variable names (data/result/item/temp). Verify git commands use `-C` pattern, not `cmd.Dir`. Verify no `sh -c` with user input.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real QA** — `unspecified-high`
  Start from clean state. Create a temp git repo. Run `claude-grid 2 --worktrees` → verify 2 worktree branches created, windows spawn in worktree dirs. Run `claude-grid kill <session>` → verify windows close, worktrees preserved, session file shows "stopped". Run `claude-grid clean <session>` → verify worktrees removed, session deleted. Run `claude-grid list` throughout to verify status display. Test edge cases: non-git repo with --worktrees, invalid branch prefix.
  Output: `Scenarios [N/N pass] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff. Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes. Verify no grid/screen/tiling logic was touched.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | VERDICT`

---

## Commit Strategy

| After Task(s) | Message | Key Files | Verification |
|----------------|---------|-----------|--------------|
| 1, 2, 3 | `feat(git): add worktree manager, name generator, and branch validation` | internal/git/*.go | go test ./internal/git/... |
| 4 | `feat(session): add worktree refs and status lifecycle to session store` | internal/session/store.go | go test ./internal/session/... |
| 5, 6 | `feat(terminal): support per-window directories in both backends` | internal/terminal/*.go | go test ./internal/terminal/... |
| 7, 8, 9, 10 | `feat(cli): add --worktrees flag, clean command, and session lifecycle` | cmd/*.go | go test ./... && go build ./... |
| 11 | `test: add worktree integration tests` | internal/git/*_test.go | go test ./... |

---

## Success Criteria

### Verification Commands
```bash
go test ./...                    # Expected: all PASS
go vet ./...                     # Expected: clean
go build ./...                   # Expected: success
claude-grid 2 --worktrees       # Expected: 2 worktree branches, 2 windows in worktree dirs
claude-grid list                 # Expected: shows session as "active"
claude-grid kill <session>       # Expected: windows close, worktrees preserved
claude-grid list                 # Expected: shows session as "stopped"
claude-grid clean <session>      # Expected: worktrees removed, session deleted
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Both backends (Terminal.app + Warp) support per-window dirs
- [ ] Session lifecycle works: active → stopped → deleted
- [ ] Fun name generator produces readable, unique prefixes

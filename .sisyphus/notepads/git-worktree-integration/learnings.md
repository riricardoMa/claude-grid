# Learnings - git-worktree-integration

Session started: 2026-02-19T05:05:51.153Z

## Conventions & Patterns
(Subagents append findings here)

- `NewManager` should trust `git rev-parse --show-toplevel` and normalize with `filepath.Abs`; tests should compare paths via `os.SameFile` because macOS temp paths can differ (`/var` vs `/private/var`).
- `CreateWorktree` needs a pre-check using `git worktree list --porcelain` for `branch refs/heads/<name>` to produce a clear "already checked out" + `claude-grid clean` hint before attempting `worktree add -b`.
- For deterministic tests, override `manager.worktreeBase` to a `t.TempDir()` path so tests never write to `~/.claude-grid/worktrees`.
- `RemoveWorktree` is safest when it skips missing directories and always calls `Prune`, which clears stale metadata entries after manual directory deletion.

## Task 2: Branch Name Generator (names.go)

### Implementation Pattern
- **crypto/rand usage**: Followed store.go pattern (lines 52-65)
  - `rand.Read(b)` for byte generation
  - Modulo operation for index selection: `int(b[0]) % max`
  - Single byte sufficient for 30-item lists (256 % 30 = 0-29 range)

### Word List Design
- **Adjectives**: 30 words, all lowercase, single syllable/short (brave, swift, calm, bold, etc.)
- **Nouns**: 30 animals, all lowercase, single syllable/short (fox, elk, owl, lynx, etc.)
- **Rationale**: Memorable, pronounceable, consistent length for clean branch names

### Testing Strategy
1. **Format validation**: Regex pattern ^[a-z]+-[a-z]+$ across 20 iterations
2. **Variety verification**: 10 calls must produce ≥2 unique values (confirms randomness)
3. **Word validation**: 50 iterations verify all words from approved lists (no mutations)

### Key Decisions
- Single byte randomness sufficient (256 values >> 30 items)
- No collision handling needed (unlike session names) - duplicates acceptable
- Exported function `GenerateBranchPrefix()` returns single pair (caller appends -N suffix)
- Helper `randomIndex()` unexported (internal implementation detail)

### Testing Insights
- All 3 tests pass consistently
- Variety test confirms crypto/rand produces different values across calls
- Word validation test ensures no accidental mutations in word lists

## Task 3: Branch Validation (validate.go)

### Validation Rules Implementation
- **Pure Go regex-based validation**: No shell out to `git check-ref-format`
- **Check ordering matters**: Consecutive slashes check must come before leading slash check
  - Pattern `//` matches both `/badname` and `//badname`
  - Must check `//` first to give correct error message for `//badname`

### Validation Rules Enforced
1. Non-empty string
2. No spaces (any whitespace)
3. No forbidden characters: ~, ^, :, \, ?, *, [
4. No double dots (..)
5. No leading/trailing dots (.)
6. No leading/trailing hyphens (-)
7. No leading forward slash (/)
8. No consecutive forward slashes (//)
9. ASCII printable characters only (32-126 range)

### Testing Strategy
- **29 test cases**: 8 valid + 21 invalid
- **Organized by rule**: Comments group related test cases
- **Comprehensive coverage**: Each rule has positive and negative tests
- **Error message validation**: Tests verify descriptive error messages

### Key Implementation Details
- `regexp.MustCompile()` for pattern matching (compiled at runtime, acceptable for validation)
- `unicode.IsPrint()` for character validation
- Early returns for efficiency (fail fast on first violation)
- Helper function `contains()` for test assertion (simple substring matching)

### Test Results
- All 29 tests pass
- No external dependencies
- Ready for integration with Task 7 (branch prefix enforcement in worktree operations)

## Task 4: Session Worktrees Integration (store.go)

### Struct Design Pattern
- **WorktreeRef struct**: Minimal design with 2 fields
  - `Path string` - absolute path to worktree directory
  - `Branch string` - branch name checked out in worktree
  - JSON tags: `json:"path"` and `json:"branch"` (no omitempty needed for required fields)

- **Session struct extension**: Added 3 new fields at END of struct
  - `Worktrees []WorktreeRef` - slice of worktree references
  - `Status string` - session lifecycle state (active/stopped/deleted)
  - `RepoPath string` - path to main repository
  - All use `json:"...,omitempty"` for backward compatibility

### Backward Compatibility Strategy
- **omitempty tags**: Missing fields in old JSON → zero values in struct
  - Slice fields default to empty slice (not nil)
  - String fields default to empty string ("")
  - No errors raised during unmarshal
- **Field ordering**: New fields added at END preserves existing field positions
- **Zero value semantics**: Application can check for empty values to detect old sessions

### Method Implementation
- **UpdateSession method**: Identical to SaveSession but with semantic distinction
  - Used for updating existing sessions vs creating new ones
  - Both overwrite session files completely
  - Follows existing pattern from SaveSession (lines 67-85)

### Testing Strategy
1. **TestSaveSessionWithWorktrees**: Verify new fields persist correctly
   - Creates session with 2 worktrees, status, and repo path
   - Loads and validates all fields present and correct
   
2. **TestBackwardCompatibilityOldSessionFormat**: Verify old sessions load
   - Creates old format JSON (no new fields)
   - Loads and verifies zero values for new fields
   - Confirms no errors during unmarshal
   
3. **TestUpdateSession**: Verify update functionality
   - Saves initial session, then updates with new data
   - Confirms UpdateSession overwrites existing file
   - Validates new fields persist after update

### Test Results
- All 14 tests pass (11 existing + 3 new)
- Execution time: 0.280s
- No breaking changes to existing API
- Backward compatibility verified with old session format

### Key Insights
- Go's json.Unmarshal handles missing optional fields gracefully
- omitempty tag prevents null values in JSON output
- Slice zero value is empty slice, not nil (important for range loops)
- UpdateSession provides semantic clarity without code duplication
- Status field supports lifecycle: active → stopped → deleted

## Task 5: Terminal Per-Window Dirs

### Refactoring Pattern
- `buildSpawnScript` signature changed from `dir string` to `dirs []string`
- Sanitization moved inside loop: each `dirs[i]` sanitized independently
- SpawnWindows prepares dirs slice: `opts.Dirs` if set, otherwise fills from `opts.Dir`

### Backward Compatibility
- Existing callers using `opts.Dir` (single dir) work unchanged
- `len(opts.Dirs) > 0` check distinguishes per-window vs uniform mode
- All 7 existing tests pass without modification

### Testing
- TestBuildSpawnScriptPerWindowDirs: verifies different dirs per window + mixed empty dirs
- TestBuildSpawnScriptPerWindowDirsBackwardCompat: verifies single Dir fills all windows
- Total: 16 tests pass in terminal package

## Task 6: Warp Per-Window Dirs

### Refactoring Pattern
- Same dirs preparation pattern as Terminal.app (Task 5): `opts.Dirs` if set, otherwise fill from `opts.Dir`
- URI construction moved inside spawn loop: each `dirs[i]` produces its own encoded URI
- Original code had URI built once outside loop; now per-iteration

### Backward Compatibility
- Existing callers using `opts.Dir` get identical behavior (all windows same URI)
- All 8 existing warp tests pass without modification
- `TestWarpURIConstruction` (existing) validates single-dir encoding still works

### Testing
- TestWarpPerWindowDirs: 3 subtests covering per-window URIs, Dir fallback, and space encoding
- Captures URIs via `runOpen` mock function — same pattern as TestWarpURIConstruction

## Task 7: Root Command Worktree Wiring

### Integration Pattern
- Insert worktree setup immediately after `resolvedDir` resolution and before backend/screen orchestration so git failures happen early and don't affect existing grid logic.
- Keep backward compatibility by only setting `SpawnOptions.Dirs` when worktrees are enabled; default `Dir` path remains unchanged for all existing call paths.

### Rollback Safety
- Use a `defer` rollback guard tied to a `spawnSucceeded` boolean for atomic cleanup of created worktrees on spawn-path failures.
- Also invoke explicit cleanup in the spawn error branch and nil out the cleanup function to avoid double-removal attempts.

### Session Persistence
- Save worktree metadata only when worktrees are actually created: `Worktrees`, `Status: "active"`, and `RepoPath`.
- Preserve existing session schema behavior for non-worktree sessions by leaving new fields empty.

## Task 8: Kill Command Worktree Preservation (kill.go)

### Implementation Pattern
- **Conditional session lifecycle**: After closing windows, check `len(sess.Worktrees) > 0`
  - If worktrees exist: set `sess.Status = "stopped"`, call `store.UpdateSession(sess)`, print preservation message
  - If no worktrees: existing behavior (delete session file)

### Code Structure
- **Lines 43-54**: New if/else block after window closure
  - Worktree branch (lines 43-48): Update session status and save
  - No-worktree branch (lines 49-54): Delete session file (original behavior)
  - Both branches print appropriate user messages

### User Messaging
- **With worktrees**: "Session 'X' stopped. N windows closed. Worktrees preserved.\nRun 'claude-grid clean X' to remove worktrees."
- **Without worktrees**: "Session 'X' killed. N windows closed." (original message)
- Error handling: Warnings printed to stderr if update/delete fails, but command succeeds

### Key Design Decisions
- **Status field semantics**: "stopped" indicates session preserved due to worktrees
- **UpdateSession usage**: Semantic distinction from SaveSession (both overwrite, but UpdateSession used for existing sessions)
- **Graceful error handling**: Warnings don't fail the command (user sees windows closed regardless)
- **User guidance**: Message includes hint to run `claude-grid clean` for worktree removal

### Testing Considerations
- Session with worktrees: verify status set to "stopped" and file persists
- Session without worktrees: verify file deleted (original behavior)
- Error cases: verify warnings printed but command succeeds

## Task 10: List Command STATUS Column

### Implementation Pattern
- **Header update**: Added STATUS column after SESSION in table header (line 32)
- **Status column logic**: 
  - Default to "active" if `sess.Status` is empty (backward compatibility)
  - Append " (stale)" if session windows no longer exist
  - Format: `statusCol := sess.Status; if statusCol == "" { statusCol = "active" }`

### Code Changes
- Line 32: Header changed from `SESSION\tBACKEND\tWINDOWS\tDIR\tCREATED` to `SESSION\tSTATUS\tBACKEND\tWINDOWS\tDIR\tCREATED`
- Lines 36-42: New status column logic with backward compatibility
- Line 50-52: Updated fprintf format string to include status column

### Backward Compatibility
- Sessions without Status field (old format) default to "active"
- Stale detection still works (appends " (stale)" suffix)
- Output format: `SESSION  STATUS  BACKEND  WINDOWS  DIR  CREATED`

### Testing
- `go build ./...` succeeds
- No LSP diagnostics
- Ready for integration with Task 11 (clean command)

## Task 9: Clean Command (clean.go)

### Implementation Pattern
- **Follows kill.go pattern**: `NewCleanCmd(storePath string) *cobra.Command`, build tag, ExactArgs(1)
- **Simpler signature than kill.go**: No `ScriptExecutor` dependency — clean doesn't interact with terminal windows
- **Error aggregation**: `var errs []error` collects failures without stopping; final joined error returned if any

### Flow
1. Load session → verify `len(sess.Worktrees) > 0` (early error if none)
2. Create `git.NewManager(sess.RepoPath)` for worktree operations
3. Loop worktrees: dirty check via `git -C <path> status --porcelain`, then `RemoveWorktree`
4. Final `manager.Prune()` (idempotent — `RemoveWorktree` also prunes internally)
5. Delete session via `store.DeleteSession`
6. Print warnings (dirty worktrees) + summary (N/M removed)

### Key Decisions
- **Dirty worktree handling**: Warn but still remove (--force is used by RemoveWorktree)
- **Error aggregation over fail-fast**: Removes as many worktrees as possible before reporting
- **Session deletion always attempted**: Even if some worktree removals fail
- **No branch deletion**: Clean only removes worktrees, not branches (explicit MUST NOT)

### Registration
- Added `cmd.AddCommand(NewCleanCmd(""))` in root.go after kill command line

# Multi-Directory & Multi-Repo Mode

## TL;DR

> **Quick Summary**: Add repeatable `--dir` and `--prompt` CLI flags plus `--manifest` YAML support to claude-grid, enabling multi-repo orchestration where each spawned instance targets a different directory with an optional prompt. The core architecture (`SpawnOptions.Dirs`) already supports per-window directories; this work wires it to CLI flags, adds prompt injection to both backends, and introduces YAML manifest parsing.
> 
> **Deliverables**:
> - Repeatable `--dir` flag (infers count, replaces single-string flag)
> - Repeatable `--prompt` flag (paired by index with dirs)
> - `--manifest` flag with YAML file parsing (`internal/manifest/` package)
> - Tilde expansion utility (`internal/pathutil/` package)
> - Hardened AppleScript escaping for prompt content
> - Per-window prompt injection in both Terminal.app and Warp backends
> - Extended session model with `Dirs`, `Prompts`, `ManifestPath` fields
> - Optional count arg (inferred from dirs/manifest)
> - Baseline `cmd/root_test.go` + TDD tests for all new code
> 
> **Estimated Effort**: Medium
> **Parallel Execution**: YES — 5 waves
> **Critical Path**: Task 3 → Task 5 → Task 8 → Task 9 → Task 10 → F1-F4

---

## Context

### Original Request
User wants to implement PRD §4.5 — Multi-Directory & Multi-Repo Mode. This enables spawning Claude Code instances across different repos (frontend, backend, infra, docs) from a single command, which is the key differentiator from claude-squad and ntm (single-repo-centric tools).

### Interview Summary
**Key Discussions**:
- **Prompt delivery**: CLI positional arg — `claude "prompt text"`. Claude Code starts and immediately processes the prompt in interactive mode.
- **Branch checkout**: Include now. Simple `git checkout <branch>` in each dir before spawning. Only checkout existing branches; error on missing branch.
- **Manifest conflicts**: Error out with clear message when `--manifest` combined with `--dir`, `--prompt`, `--worktrees`, or count arg. Allow `--name`, `--layout`, `--terminal` (orthogonal).
- **Test strategy**: TDD (tests first). Table-driven tests with mock executors matching existing patterns.

**Research Findings**:
- `SpawnOptions.Dirs []string` already exists at `backend.go:44` — both backends consume it
- `buildSpawnScript` in `terminal_app.go:106-138` constructs per-window `do script` — easy to extend with prompt
- Warp's `sendCommandToWindows` at `warp.go:211-234` sends same command to ALL windows — needs refactoring to per-window
- `SanitizeForAppleScript` at `executor.go:63-71` only handles `\` and `"` — insufficient for prompt content
- No `cmd/` tests exist — adding new flag logic to untested code is risky
- `filepath.Abs("~/foo")` produces `/cwd/~/foo` — explicit tilde expansion needed

### Metis Review
**Identified Gaps** (addressed):
- Prompt escaping is triple-nested (shell → AppleScript → shell) — resolved: harden sanitization as Task 5
- Warp backend sends same command to all windows — resolved: refactor to per-window commands in Task 7
- No cmd/ tests — resolved: add baseline tests as Task 2 before any modifications
- Need to verify `claude "prompt"` syntax actually works — resolved: research task (Task 1)
- Prompt+dir count mismatch undefined — resolved: dirs pad with last-dir/$PWD, missing prompts → no prompt for that window
- Manifest dir resolution — resolved: resolve relative to manifest file location (standard config file behavior)

---

## Work Objectives

### Core Objective
Enable multi-repo orchestration by making `--dir` and `--prompt` repeatable CLI flags and adding `--manifest` YAML support, so users can spawn Claude Code instances across different directories and repos from a single command.

### Concrete Deliverables
- `internal/pathutil/expand.go` + `expand_test.go` — tilde expansion utility
- `internal/manifest/manifest.go` + `manifest_test.go` — YAML manifest parser
- Extended `internal/script/executor.go` — hardened sanitization for prompt content
- Extended `internal/terminal/terminal_app.go` — per-window prompt injection
- Extended `internal/terminal/warp.go` — refactored per-window command sending
- Extended `internal/terminal/backend.go` — `Prompts []string` in `SpawnOptions`
- Extended `internal/session/store.go` — new `Dirs`, `Prompts`, `ManifestPath` fields
- Modified `cmd/root.go` — repeatable flags, manifest, count inference, validation
- New `cmd/root_test.go` — comprehensive CLI layer tests

### Definition of Done
- [ ] `make check` passes (go vet + go test + go build)
- [ ] `claude-grid 3 --dir ~/a --dir ~/b --dir ~/c` spawns 3 instances in different dirs
- [ ] `claude-grid --dir ~/a --dir ~/b --prompt "fix X" --prompt "do Y"` works with count inference
- [ ] `claude-grid --manifest sprint.yaml` parses YAML and spawns instances per manifest
- [ ] `claude-grid --manifest sprint.yaml --dir ~/x` errors with conflict message
- [ ] Session JSON includes `dirs` and `prompts` fields; old sessions still load correctly

### Must Have
- Repeatable `--dir` flag using cobra `StringArrayVarP`
- Repeatable `--prompt` flag using cobra `StringArrayVarP`
- `--manifest` flag for YAML file
- Count arg becomes optional when dirs/manifest provide instance count
- Tilde expansion (`~` → home dir) for all path inputs
- Directory existence validation before spawning
- Per-window prompt injection in Terminal.app backend
- Per-window prompt injection in Warp backend (refactored `sendCommandToWindows`)
- Hardened AppleScript escaping for prompt content (newlines, backticks, `$()`)
- Session model backward compatibility (new fields use `omitempty`)
- Manifest `branch` field with `git checkout` before spawning
- TDD tests for all new code

### Must NOT Have (Guardrails)
- `--prompt-all` flag (defer to follow-up)
- `--prompts-file` flag (defer to follow-up)
- Branch creation (`git checkout -b`) — only checkout existing branches
- Manifest schema versioning or migration logic
- Manifest inheritance, includes, or templating
- Environment variable substitution in manifests
- Manifest validation subcommand (`claude-grid validate`)
- Changes to `list`, `kill`, or `clean` commands for multi-dir display
- Config file support (`~/.claude-grid.toml`)
- Any new subcommands
- Excessive JSDoc/comments — follow existing codebase comment density
- Over-abstraction — no generic "resolver" interfaces for path handling

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (go test, `make test`, `make check`)
- **Automated tests**: TDD (tests first)
- **Framework**: `go test` with `testing` package (standard library)
- **TDD flow**: RED (write failing test) → GREEN (minimal implementation) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Unit tests**: `go test ./internal/{package}/ -v -count=1` — assert pass/fail
- **Build verification**: `go build ./...` — assert compiles
- **Lint/vet**: `go vet ./...` — assert clean

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — all independent, 4 parallel):
├── Task 1: Verify claude CLI prompt syntax [quick]
├── Task 2: Baseline cmd/root_test.go [unspecified-low]
├── Task 3: Create internal/pathutil/ (tilde expansion) [quick]
└── Task 4: Harden SanitizeForAppleScript [quick]

Wave 2 (After Wave 1 — core modules, 4 parallel):
├── Task 5: Create internal/manifest/ parser (depends: 3) [unspecified-low]
├── Task 6: Terminal.app per-window prompts (depends: 1, 4) [unspecified-low]
├── Task 7: Warp per-window prompts (depends: 1, 4) [unspecified-low]
└── Task 8: Extend Session model (depends: 3) [quick]

Wave 3 (After Wave 2 — integration):
└── Task 9: Wire everything in cmd/root.go (depends: 2,3,5,6,7,8) [unspecified-high]

Wave 4 (After Wave 3 — full verification):
└── Task 10: Full test suite + build verification (depends: 9) [quick]

Wave FINAL (After ALL tasks — independent review, 4 parallel):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: Task 3 → Task 5 → Task 9 → Task 10 → F1-F4
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 4 (Waves 1 & 2)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | — | 6, 7 | 1 |
| 2 | — | 9 | 1 |
| 3 | — | 5, 8, 9 | 1 |
| 4 | — | 6, 7 | 1 |
| 5 | 3 | 9 | 2 |
| 6 | 1, 4 | 9 | 2 |
| 7 | 1, 4 | 9 | 2 |
| 8 | 3 | 9 | 2 |
| 9 | 2, 3, 5, 6, 7, 8 | 10 | 3 |
| 10 | 9 | F1-F4 | 4 |
| F1-F4 | 10 | — | FINAL |

### Agent Dispatch Summary

- **Wave 1**: **4** — T1 → `quick`, T2 → `unspecified-low`, T3 → `quick`, T4 → `quick`
- **Wave 2**: **4** — T5 → `unspecified-low`, T6 → `unspecified-low`, T7 → `unspecified-low`, T8 → `quick`
- **Wave 3**: **1** — T9 → `unspecified-high`
- **Wave 4**: **1** — T10 → `quick`
- **FINAL**: **4** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [ ] 1. Verify Claude CLI prompt syntax

  **What to do**:
  - Research the exact Claude Code CLI syntax for passing an initial prompt. Check `claude --help` output and/or official documentation.
  - Confirm whether `claude "prompt text"` (positional arg), `claude --prompt "text"`, `claude -p "text"`, or another syntax starts interactive mode with that prompt.
  - Document the confirmed syntax in a brief output (the exact command string format).
  - This is a research-only task — no code changes.

  **Must NOT do**:
  - Do not write any code or modify files
  - Do not install or uninstall anything

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Pure research task, no code changes, just verify a CLI interface
  - **Skills**: []
    - No special skills needed — use `claude --help` via Bash or web search

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: Tasks 6, 7 (backends need confirmed syntax for prompt injection)
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `internal/terminal/terminal_app.go:113-116` — Current command construction: `cd dir && claude`. Shows where prompt would be appended.
  - `internal/terminal/warp.go:112-118` — Warp sends command via keystrokes. Shows the command string that would include prompt.

  **External References**:
  - Claude Code CLI documentation — verify exact prompt argument syntax

  **WHY Each Reference Matters**:
  - `terminal_app.go:113-116`: The confirmed syntax determines the format of the `windowCommand` string (e.g., `cd dir && claude "prompt"` vs `cd dir && claude --prompt "text"`)
  - `warp.go:112-118`: Same syntax must work when sent as keystrokes to Warp windows

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Verify claude CLI accepts prompt argument
    Tool: Bash
    Preconditions: claude CLI is installed (`which claude` returns a path)
    Steps:
      1. Run `claude --help` and capture output
      2. Search output for prompt-related flags/arguments
      3. Document the exact syntax for passing a prompt that starts interactive mode
    Expected Result: Confirmed syntax documented (e.g., `claude -p "text"` for print or `claude "text"` for interactive)
    Failure Indicators: `claude --help` returns no prompt-related info; claude not installed
    Evidence: .sisyphus/evidence/task-1-claude-cli-syntax.txt
  ```

  **Commit**: NO (research only, no code changes)

- [ ] 2. Add cmd/root_test.go — baseline tests for existing behavior

  **What to do**:
  - Create `cmd/root_test.go` with table-driven tests covering the CURRENT behavior before any modifications.
  - Test cases: valid count arg (`claude-grid 4`), missing count (no args → error), count out of range (0, 17 → error), invalid count ("abc" → error), `--dir` flag with value, `--layout` flag with valid/invalid values, `--version` flag.
  - Use `cmd.SetArgs()` and `cmd.Execute()` pattern. Capture stderr/stdout via `bytes.Buffer` using `cmd.SetOut()` and `cmd.SetErr()`.
  - Follow table-driven test style from `terminal_app_test.go:46-131`.
  - These tests must PASS against the current code (no modifications to root.go).

  **Must NOT do**:
  - Do not modify `cmd/root.go` or any existing files
  - Do not add tests for features that don't exist yet (--prompt, --manifest)
  - Do not test actual window spawning (mock is not needed — test flag parsing and validation only)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
    - Reason: Single-file test creation following established patterns. Moderate effort — needs understanding of cobra test patterns.
  - **Skills**: []
    - No special skills needed — standard Go testing

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4)
  - **Blocks**: Task 9 (CLI integration must not regress existing behavior)
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `internal/terminal/terminal_app_test.go:46-131` — Table-driven test structure with subtests, error checking pattern
  - `internal/session/store_test.go:64-111` — Simple setup-execute-verify pattern
  - `cmd/root.go:23-288` — The function under test. All flag definitions (lines 260-267), argument validation (lines 50-63), error formatting.

  **API/Type References**:
  - `cmd/root.go:23` — `NewRootCommand(version, commit, date string) *cobra.Command` — the constructor to test

  **External References**:
  - Cobra testing patterns: `cmd.SetArgs([]string{"4"})`, `cmd.SetOut(&buf)`, `cmd.SetErr(&buf)`, `cmd.Execute()`

  **WHY Each Reference Matters**:
  - `terminal_app_test.go:46-131`: Copy this exact table-driven pattern — struct with `name`, `args`, `wantErr`, `wantOutput` fields
  - `root.go:23-288`: Must understand every validation branch to write comprehensive tests. Lines 50-63 are the count validation, 66-69 claude path check, 78-86 dir resolution, 151-158 layout parsing.
  - Note: Tests will fail at the `exec.LookPath("claude")` check (line 65-69) unless claude is installed. Consider testing that the error message is correct when claude is not found, or skip those tests with `t.Skip` when claude is absent.

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] Test file created: `cmd/root_test.go`
  - [ ] `go test ./cmd/ -v -count=1` → PASS (all subtests pass against current code)
  - [ ] Tests cover: valid count, missing count, out-of-range count, invalid count, --version flag, --layout valid/invalid

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: All baseline tests pass against unmodified root.go
    Tool: Bash
    Preconditions: No modifications to cmd/root.go
    Steps:
      1. Run `go test ./cmd/ -v -count=1`
      2. Verify exit code is 0
      3. Count number of test functions in output
    Expected Result: Exit code 0, at least 5 test subtests pass (count validation, version, layout, etc.)
    Failure Indicators: Any test fails; compilation error; import cycle
    Evidence: .sisyphus/evidence/task-2-baseline-tests.txt

  Scenario: Tests detect regressions when validation is broken
    Tool: Bash
    Preconditions: Temporarily break count validation in root.go (change max from 16 to 8)
    Steps:
      1. Temporarily modify root.go line 60: change `count > 16` to `count > 8`
      2. Run `go test ./cmd/ -v -count=1`
      3. Verify at least one test fails for count=16 or similar edge case
      4. Revert the temporary change
    Expected Result: At least one test fails, proving the baseline catches regressions
    Failure Indicators: All tests pass even with broken validation (tests are too weak)
    Evidence: .sisyphus/evidence/task-2-regression-detection.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `test(cmd): add baseline root command tests before multi-repo changes`
  - Files: `cmd/root_test.go`
  - Pre-commit: `go test ./cmd/ -count=1`

- [ ] 3. Create internal/pathutil/ — tilde expansion utility

  **What to do**:
  - **RED**: Create `internal/pathutil/expand_test.go` first with table-driven tests:
    - `"~/foo"` → `"{homedir}/foo"`
    - `"~"` → `"{homedir}"`
    - `"/absolute/path"` → `"/absolute/path"` (passthrough)
    - `"relative/path"` → `"relative/path"` (passthrough)
    - `""` → `""` (empty passthrough)
    - `"~user/foo"` → error (unsupported `~user` syntax)
  - **GREEN**: Create `internal/pathutil/expand.go` with:
    - `func ExpandTilde(path string) (string, error)` — replaces leading `~` with `os.UserHomeDir()`. Returns error for `~user` syntax (unsupported). Passes through non-tilde paths unchanged.
    - `func ExpandTildeAll(paths []string) ([]string, error)` — applies ExpandTilde to each element, returns on first error.
  - **REFACTOR**: Clean up if needed.

  **Must NOT do**:
  - Do not support `~user` syntax (only `~` for current user)
  - Do not resolve relative paths to absolute — only handle tilde
  - Do not add any dependencies (use standard library only)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small, self-contained utility with clear spec. ~2 files, ~50 lines each.
  - **Skills**: []
    - No special skills needed — pure Go standard library

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4)
  - **Blocks**: Tasks 5 (manifest parser uses tilde expansion), 8 (session model alignment), 9 (CLI uses it)
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `internal/session/store_test.go:12-21` — Test file structure, imports, table-driven pattern in this codebase
  - `internal/git/validate.go` — Similar small utility pattern (validation function, simple input/output)
  - `internal/git/validate_test.go` — Test structure for utility functions

  **API/Type References**:
  - `os.UserHomeDir()` — Go standard library, returns home directory

  **WHY Each Reference Matters**:
  - `store_test.go`: Shows the exact import style and test function naming convention used in this project
  - `validate.go` / `validate_test.go`: Closest pattern to what we're building — small utility with validation, table-driven tests

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] Test file created: `internal/pathutil/expand_test.go`
  - [ ] Source file created: `internal/pathutil/expand.go`
  - [ ] `go test ./internal/pathutil/ -v -count=1` → PASS (all 6+ subtests pass)

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Tilde expansion works correctly
    Tool: Bash
    Preconditions: Go project compiles
    Steps:
      1. Run `go test ./internal/pathutil/ -v -count=1`
      2. Verify all subtests pass: tilde-slash, tilde-alone, absolute, relative, empty, tilde-user-error
    Expected Result: All 6+ tests pass, exit code 0
    Failure Indicators: Any subtest fails; compilation error
    Evidence: .sisyphus/evidence/task-3-pathutil-tests.txt

  Scenario: ExpandTildeAll stops on first error
    Tool: Bash
    Preconditions: pathutil package exists
    Steps:
      1. Verify test exists for `ExpandTildeAll([]string{"~/valid", "~otheruser/bad", "~/alsovalid"})`
      2. Run that test
      3. Confirm error returned, and only first path processed
    Expected Result: Error returned for `~otheruser` syntax, function aborts early
    Failure Indicators: No error; all paths processed despite invalid entry
    Evidence: .sisyphus/evidence/task-3-expandall-error.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(pathutil): add tilde expansion utility for multi-dir path handling`
  - Files: `internal/pathutil/expand.go`, `internal/pathutil/expand_test.go`
  - Pre-commit: `go test ./internal/pathutil/ -count=1`

- [ ] 4. Harden SanitizeForAppleScript for prompt content

  **What to do**:
  - **RED**: Add new test cases to `internal/script/executor_test.go` for prompt-like content:
    - Prompt with double quotes: `"fix the \"login\" page"` — verify quotes escaped
    - Prompt with single quotes: `"don't break"` — verify single quotes handled
    - Prompt with newlines: `"line1\nline2"` — verify newlines escaped
    - Prompt with backticks: `` "run `whoami`" `` — verify backticks escaped
    - Prompt with shell expansion: `"fix $(pwd) issues"` — verify `$(` escaped
    - Verify ALL existing tests still pass (no regressions)
  - **GREEN**: Extend `SanitizeForAppleScript` in `internal/script/executor.go` to additionally handle:
    - Newlines: `\n` → literal `\\n` in AppleScript context
    - Backticks: `` ` `` → `` \` ``
    - Dollar-paren: `$(` → `\$(` (prevent shell expansion in spawned terminal)
    - Single quotes don't need escaping in AppleScript double-quoted strings, but verify this
  - **REFACTOR**: Ensure the escaping order is correct (backslashes first, then everything else — existing comment at executor.go:62 explains why).

  **Must NOT do**:
  - Do not change the function signature (still `func SanitizeForAppleScript(s string) string`)
  - Do not break existing behavior (existing tests must continue to pass)
  - Do not add a separate "SanitizePrompt" function — extend the existing one

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small, focused change to one function + test additions. ~20 lines of code changes.
  - **Skills**: []
    - No special skills needed — Go string manipulation

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3)
  - **Blocks**: Tasks 6, 7 (backends need safe escaping before injecting prompts)
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `internal/script/executor.go:61-71` — Current `SanitizeForAppleScript` implementation. CRITICAL: read the comment about escaping order (backslashes first).
  - `internal/script/executor_test.go` — Existing tests for this function (if any; check file for `TestSanitize` patterns)

  **API/Type References**:
  - `internal/script/executor.go:63` — `func SanitizeForAppleScript(s string) string` — the function to extend

  **External References**:
  - AppleScript string escaping rules: In double-quoted AppleScript strings, `\` and `"` must be escaped. Newlines can be embedded via `\n` but that creates actual newlines in the script.

  **WHY Each Reference Matters**:
  - `executor.go:61-71`: MUST understand the existing escaping order before adding new escapes. The comment explains why backslashes MUST be escaped first.
  - The triple-nesting context: User input → Go string → AppleScript `do script "..."` → spawned shell. Each layer has its own escaping needs. The `SanitizeForAppleScript` function handles the Go→AppleScript boundary.

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] New test cases added to `internal/script/executor_test.go`
  - [ ] `go test ./internal/script/ -v -count=1` → PASS (all old + new tests pass)

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Existing escaping still works
    Tool: Bash
    Preconditions: executor_test.go has existing tests
    Steps:
      1. Run `go test ./internal/script/ -v -count=1`
      2. Verify all pre-existing tests pass
    Expected Result: Zero regressions in existing tests
    Failure Indicators: Any existing test fails
    Evidence: .sisyphus/evidence/task-4-existing-tests.txt

  Scenario: Prompt-specific escaping works
    Tool: Bash
    Preconditions: New test cases added
    Steps:
      1. Run `go test ./internal/script/ -run TestSanitize -v -count=1`
      2. Verify prompt-specific subtests pass: double quotes, newlines, backticks, dollar-paren
    Expected Result: All prompt escaping subtests pass
    Failure Indicators: Any escaping test fails; shell expansion characters not escaped
    Evidence: .sisyphus/evidence/task-4-prompt-escaping.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `fix(script): harden AppleScript sanitization for prompt content (newlines, backticks, shell expansion)`
  - Files: `internal/script/executor.go`, `internal/script/executor_test.go`
  - Pre-commit: `go test ./internal/script/ -count=1`

- [ ] 5. Create internal/manifest/ — YAML manifest parser

  **What to do**:
  - **Add dependency**: `go get gopkg.in/yaml.v3` then `go mod tidy`
  - **RED**: Create `internal/manifest/manifest_test.go` first with table-driven tests:
    - Valid manifest with all fields (name, instances with dir/prompt/branch)
    - Valid manifest with minimal fields (instances with only dir)
    - Invalid YAML syntax → parse error
    - Missing `instances` field → error
    - Empty `instances` list → error
    - Instance missing `dir` field → error
    - More than 16 instances → error
    - Dir with tilde (`~/projects/foo`) → expanded
    - Relative dir in manifest → resolved relative to manifest file location
    - Branch field present → stored (no checkout here, that's CLI layer)
  - **GREEN**: Create `internal/manifest/manifest.go`:
    - Types: `Manifest struct { Name string; Instances []Instance }`, `Instance struct { Dir string; Prompt string; Branch string }`
    - YAML tags: `yaml:"name"`, `yaml:"instances"`, `yaml:"dir"`, `yaml:"prompt"`, `yaml:"branch"`
    - `func Parse(manifestPath string) (Manifest, error)` — reads file, unmarshals YAML, validates, applies tilde expansion to `Dir` fields, resolves relative dirs relative to manifest file's parent directory.
    - Validation: non-empty instances, ≤16 instances, non-empty Dir per instance.
  - **REFACTOR**: Clean up.

  **Must NOT do**:
  - Do not add manifest schema versioning
  - Do not support environment variable substitution in manifests
  - Do not support manifest inheritance or includes
  - Do not validate that dirs exist (that's the CLI layer's job)
  - Do not perform branch checkout (that's the CLI layer's job)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
    - Reason: New package with YAML parsing, moderate complexity. Needs dependency addition, struct definitions, file I/O, validation.
  - **Skills**: []
    - No special skills needed — standard Go + yaml.v3

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 7, 8)
  - **Blocks**: Task 9 (CLI integration needs manifest parser)
  - **Blocked By**: Task 3 (uses pathutil for tilde expansion)

  **References**:

  **Pattern References**:
  - `internal/session/store.go:14-24` — Struct definition pattern with JSON tags. Follow same style for YAML tags.
  - `internal/session/store_test.go:64-111` — Test pattern: create temp file, call function, verify fields.
  - `internal/pathutil/expand.go` — (from Task 3) Tilde expansion function to call for Dir fields.

  **API/Type References**:
  - `gopkg.in/yaml.v3` — `yaml.Unmarshal(data, &manifest)` — standard YAML parsing
  - `internal/pathutil.ExpandTilde(path)` — tilde expansion for manifest Dir fields

  **External References**:
  - PRD `prd.md:148-163` — Manifest file format spec (name, instances with dir/prompt/branch)

  **WHY Each Reference Matters**:
  - `store.go:14-24`: Struct tag style to match. The project uses `json:"field_name,omitempty"` — use equivalent `yaml:"field"` tags.
  - `store_test.go:64-111`: Test file creation pattern. Use `os.WriteFile` to create temp YAML files, then parse them.
  - `prd.md:148-163`: THE authoritative spec for manifest format. Struct must match this exactly.

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] Test file created: `internal/manifest/manifest_test.go`
  - [ ] Source file created: `internal/manifest/manifest.go`
  - [ ] `go get gopkg.in/yaml.v3` added to go.mod
  - [ ] `go test ./internal/manifest/ -v -count=1` → PASS (all 10+ subtests pass)

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Valid manifest parses correctly
    Tool: Bash
    Preconditions: manifest package exists
    Steps:
      1. Run `go test ./internal/manifest/ -run TestParse -v -count=1`
      2. Verify subtests pass: valid-full, valid-minimal, tilde-expansion, relative-dirs
    Expected Result: All valid manifest tests pass, fields correctly populated
    Failure Indicators: Any subtest fails; Dir not expanded; relative paths not resolved
    Evidence: .sisyphus/evidence/task-5-manifest-valid.txt

  Scenario: Invalid manifests produce clear errors
    Tool: Bash
    Preconditions: manifest package exists
    Steps:
      1. Run `go test ./internal/manifest/ -run TestParse -v -count=1`
      2. Verify subtests pass: invalid-yaml, missing-instances, empty-instances, missing-dir, too-many-instances
    Expected Result: Each invalid case returns descriptive error (not generic "unmarshal failed")
    Failure Indicators: Invalid manifests accepted without error; generic/unhelpful error messages
    Evidence: .sisyphus/evidence/task-5-manifest-invalid.txt

  Scenario: Dependency addition is clean
    Tool: Bash
    Preconditions: yaml.v3 dependency added
    Steps:
      1. Run `go mod tidy`
      2. Run `go mod verify`
      3. Verify go.mod contains `gopkg.in/yaml.v3`
    Expected Result: go.mod clean, no extraneous deps, yaml.v3 present
    Failure Indicators: go mod tidy adds/removes unexpected dependencies
    Evidence: .sisyphus/evidence/task-5-gomod.txt
  ```

  **Commit**: YES (groups with Wave 2)
  - Message: `feat(manifest): add YAML manifest parser for multi-repo sprint configuration`
  - Files: `internal/manifest/manifest.go`, `internal/manifest/manifest_test.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./internal/manifest/ -count=1`

- [ ] 6. Extend Terminal.app backend — per-window prompts

  **What to do**:
  - **First**: Add `Prompts []string` field to `SpawnOptions` in `internal/terminal/backend.go:32-58`.
  - **RED**: Add tests to `internal/terminal/terminal_app_test.go`:
    - Spawn with prompts: 3 windows, 3 different prompts → each `do script` contains `claude "prompt N"`
    - Spawn with prompt containing special chars: prompt with quotes → properly escaped in AppleScript
    - Spawn with fewer prompts than windows: 3 windows, 1 prompt → first window has prompt, others just `claude`
    - Spawn with no prompts (backward compat): existing tests still pass, `do script` contains `claude` without prompt
    - Follow `TestBuildSpawnScriptPerWindowDirs` pattern at `terminal_app_test.go:134-192`.
  - **GREEN**: Modify `buildSpawnScript` in `terminal_app.go`:
    - Accept `prompts []string` parameter (or read from opts passed through)
    - When `prompts[i]` is non-empty, construct: `cd dir && claude "prompt"` (using confirmed syntax from Task 1)
    - When `prompts[i]` is empty or index out of range, construct: `cd dir && claude` (current behavior)
    - Sanitize prompt text using `SanitizeForAppleScript` before embedding in script
  - **REFACTOR**: Update `SpawnWindows` to pass prompts through to `buildSpawnScript`.

  **Must NOT do**:
  - Do not change the `TerminalBackend` interface
  - Do not break backward compatibility — zero-prompt case must work identically to current behavior
  - Do not handle prompt delivery via keystroke injection — use command-line argument only

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
    - Reason: Modifying existing backend with new functionality. Needs careful AppleScript string construction.
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 7, 8)
  - **Blocks**: Task 9 (CLI needs backend prompt support)
  - **Blocked By**: Task 1 (prompt syntax), Task 4 (sanitization hardening)

  **References**:

  **Pattern References**:
  - `internal/terminal/terminal_app.go:106-138` — Current `buildSpawnScript`. Lines 111-116 construct the per-window command. This is where prompt gets appended.
  - `internal/terminal/terminal_app.go:59-67` — How `Dirs` fallback works (if no per-window dirs, use single dir). Same pattern for prompts.
  - `internal/terminal/terminal_app_test.go:134-192` — `TestBuildSpawnScriptPerWindowDirs` — exact test pattern to follow for per-window prompts.

  **API/Type References**:
  - `internal/terminal/backend.go:32-58` — `SpawnOptions` struct. Add `Prompts []string` here.
  - `internal/script/executor.go:63` — `SanitizeForAppleScript` — use for prompt text escaping.

  **WHY Each Reference Matters**:
  - `terminal_app.go:111-116`: The exact line where `windowCommand` is constructed. Prompt appends here: `windowCommand = fmt.Sprintf("cd \\\"%s\\\" && %s \\\"%s\\\"", dir, command, prompt)`.
  - `terminal_app.go:59-67`: Shows the fallback pattern for Dirs. Copy this pattern for Prompts (if empty, no prompt for that window).
  - `terminal_app_test.go:134-192`: The existing per-window-dirs test. Copy structure, change dirs to prompts.

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] `Prompts []string` added to `SpawnOptions` in `backend.go`
  - [ ] New test cases in `terminal_app_test.go`
  - [ ] `go test ./internal/terminal/ -run TestTerminalApp -v -count=1` → PASS
  - [ ] Existing tests still pass (backward compat)

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Per-window prompts in Terminal.app
    Tool: Bash
    Preconditions: terminal_app.go modified, backend.go has Prompts field
    Steps:
      1. Run `go test ./internal/terminal/ -run TestTerminalApp -v -count=1`
      2. Verify subtest for per-window prompts passes
      3. Verify the generated AppleScript contains `claude "prompt text"` for prompted windows
    Expected Result: Tests pass, generated script has per-window prompt in `do script` commands
    Failure Indicators: Prompt not appearing in generated script; escaping broken; backward compat test fails
    Evidence: .sisyphus/evidence/task-6-terminal-prompts.txt

  Scenario: Backward compatibility — no prompts
    Tool: Bash
    Preconditions: All changes applied
    Steps:
      1. Run `go test ./internal/terminal/ -run TestTerminalAppSpawnScript -v -count=1`
      2. Run `go test ./internal/terminal/ -run TestBuildSpawnScriptPerWindowDirs -v -count=1`
      3. Verify ALL existing tests pass unchanged
    Expected Result: Zero regressions in existing terminal backend tests
    Failure Indicators: Any pre-existing test fails
    Evidence: .sisyphus/evidence/task-6-backward-compat.txt
  ```

  **Commit**: YES (groups with Wave 2)
  - Message: `feat(terminal): add per-window prompt injection for Terminal.app backend`
  - Files: `internal/terminal/backend.go`, `internal/terminal/terminal_app.go`, `internal/terminal/terminal_app_test.go`
  - Pre-commit: `go test ./internal/terminal/ -count=1`

- [ ] 7. Extend Warp backend — per-window prompts

  **What to do**:
  - **RED**: Add tests to `internal/terminal/warp_test.go`:
    - Per-window prompts: 3 windows, 3 different prompts → each window gets keystroke for `claude "prompt N"` (not bare `claude`)
    - No prompts (backward compat): existing behavior preserved — bare `claude` sent to all
    - Partial prompts: 3 windows, 1 prompt → first window gets `claude "prompt"`, others get `claude`
    - Follow existing Warp test patterns in `warp_test.go`.
  - **GREEN**: Refactor `sendCommandToWindows` in `warp.go`:
    - Change signature from `sendCommandToWindows(ctx, count int, command string)` to `sendCommandsToWindows(ctx context.Context, commands []string)`.
    - Each `commands[i]` is the full command string for window `i` (e.g., `claude "prompt 1"`, `claude`, etc.).
    - Update `SpawnWindows` to construct per-window command strings from `opts.Command` + `opts.Prompts[i]`.
    - When `Prompts` is empty or `Prompts[i]` is empty, use bare `opts.Command`.
  - **REFACTOR**: Ensure the keystroke sending loop uses the per-window command correctly.

  **Must NOT do**:
  - Do not change the `TerminalBackend` interface
  - Do not break the Warp spawning flow (URI scheme + tiling + command sending)
  - Do not use prompt delivery via Warp Launch Config YAML (keep using keystroke approach for now)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
    - Reason: Backend refactoring with signature change. Needs careful handling of Warp's keystroke-based command injection.
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6, 8)
  - **Blocks**: Task 9 (CLI needs backend prompt support)
  - **Blocked By**: Task 1 (prompt syntax), Task 4 (sanitization hardening)

  **References**:

  **Pattern References**:
  - `internal/terminal/warp.go:211-234` — Current `sendCommandToWindows`. This is the function to refactor. Note: it uses `script.SanitizeForAppleScript` for the command, then sends keystrokes per window.
  - `internal/terminal/warp.go:66-130` — `SpawnWindows`. Lines 112-118 construct the command and call `sendCommandToWindows`. This is where per-window commands get assembled from `opts.Command` + `opts.Prompts[i]`.
  - `internal/terminal/warp.go:79-87` — How per-window dirs are resolved (same pattern for prompts).

  **API/Type References**:
  - `internal/terminal/backend.go:32-58` — `SpawnOptions.Prompts []string` (added in Task 6)
  - `internal/terminal/warp.go:211` — `func (b *WarpBackend) sendCommandToWindows(ctx context.Context, count int, command string) error` — signature to change

  **WHY Each Reference Matters**:
  - `warp.go:211-234`: The function sends the SAME command to ALL windows via keystrokes. Must refactor to send DIFFERENT commands per window. The loop at line 217 iterates `i from 1 to count` — change to iterate through `commands` slice.
  - `warp.go:79-87`: Shows how `dirs` fallback works in Warp's `SpawnWindows`. Copy this exact pattern for assembling per-window commands from `Command` + `Prompts`.

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] `sendCommandToWindows` renamed/refactored to `sendCommandsToWindows` accepting `[]string`
  - [ ] New test cases in `warp_test.go`
  - [ ] `go test ./internal/terminal/ -run TestWarp -v -count=1` → PASS
  - [ ] Existing Warp tests still pass

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Per-window commands in Warp
    Tool: Bash
    Preconditions: warp.go modified with per-window command support
    Steps:
      1. Run `go test ./internal/terminal/ -run TestWarp -v -count=1`
      2. Verify per-window prompt test passes
      3. Verify the AppleScript keystrokes contain different commands per window
    Expected Result: Tests pass, different keystroke commands sent per window
    Failure Indicators: Same command sent to all windows; test fails; compilation error
    Evidence: .sisyphus/evidence/task-7-warp-prompts.txt

  Scenario: Warp backward compatibility
    Tool: Bash
    Preconditions: All changes applied
    Steps:
      1. Run `go test ./internal/terminal/ -run TestWarp -v -count=1`
      2. Verify ALL existing Warp tests pass
    Expected Result: Zero regressions
    Failure Indicators: Any existing Warp test fails
    Evidence: .sisyphus/evidence/task-7-warp-compat.txt
  ```

  **Commit**: YES (groups with Wave 2)
  - Message: `feat(warp): refactor command sending for per-window prompt support`
  - Files: `internal/terminal/warp.go`, `internal/terminal/warp_test.go`
  - Pre-commit: `go test ./internal/terminal/ -count=1`

- [ ] 8. Extend Session model — new fields for multi-dir/prompt

  **What to do**:
  - **RED**: Add tests to `internal/session/store_test.go`:
    - Save session with `Dirs`, `Prompts`, `ManifestPath` fields → load back → verify fields match
    - Load old session JSON without new fields → verify `Dirs` is nil, `Prompts` is nil, `ManifestPath` is empty (backward compat)
    - Follow `TestSaveSessionWithWorktrees` pattern at `store_test.go:365-412`.
  - **GREEN**: Add to `Session` struct in `internal/session/store.go`:
    - `Dirs []string \`json:"dirs,omitempty"\`` — per-instance directories
    - `Prompts []string \`json:"prompts,omitempty"\`` — per-instance prompts
    - `ManifestPath string \`json:"manifest_path,omitempty"\`` — path to manifest file used
  - **REFACTOR**: Ensure JSON serialization is clean (omitempty means old sessions without these fields are unaffected).

  **Must NOT do**:
  - Do not remove or rename the existing `Dir string` field — keep it for backward compat
  - Do not add validation logic to the session store (validation belongs in the CLI layer)
  - Do not modify `SaveSession` or `LoadSession` signatures

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Adding 3 fields to a struct + 2 test functions. Very small, well-defined change.
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6, 7)
  - **Blocks**: Task 9 (CLI stores multi-dir session data)
  - **Blocked By**: Task 3 (field naming alignment with pathutil convention)

  **References**:

  **Pattern References**:
  - `internal/session/store.go:14-24` — Current `Session` struct. New fields go after `RepoPath` (line 23). Follow exact tag pattern: `json:"field_name,omitempty"`.
  - `internal/session/store_test.go:365-412` — `TestSaveSessionWithWorktrees`. Exact pattern to follow: create session with new fields, save, load, assert fields match.
  - `internal/session/store_test.go:414-453` — `TestBackwardCompatibilityOldSessionFormat`. Follow this pattern for backward compat test with old JSON.

  **API/Type References**:
  - `internal/session/store.go:14` — `type Session struct` — the struct to extend

  **WHY Each Reference Matters**:
  - `store.go:14-24`: Shows the exact struct field pattern. New fields MUST follow the `json:"field,omitempty"` convention. Fields like `Worktrees` (line 21) are the closest analog.
  - `store_test.go:365-412`: Copy-paste this test, rename to `TestSaveSessionWithMultiDir`, change fields to `Dirs`/`Prompts`.
  - `store_test.go:414-453`: The backward compat test creates raw JSON without new fields and verifies they default to zero values. Copy this for the new fields.

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] New fields added to `Session` struct: `Dirs`, `Prompts`, `ManifestPath`
  - [ ] New tests in `store_test.go`
  - [ ] `go test ./internal/session/ -v -count=1` → PASS (all old + new tests pass)

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Session with multi-dir fields saves and loads
    Tool: Bash
    Preconditions: store.go has new fields
    Steps:
      1. Run `go test ./internal/session/ -run TestSaveSessionWithMultiDir -v -count=1`
      2. Verify dirs, prompts, manifest_path fields round-trip correctly
    Expected Result: All new fields preserved after save/load cycle
    Failure Indicators: Fields nil after load; JSON serialization error
    Evidence: .sisyphus/evidence/task-8-session-multidir.txt

  Scenario: Old sessions still load without new fields
    Tool: Bash
    Preconditions: store.go has new fields
    Steps:
      1. Run `go test ./internal/session/ -run TestBackwardCompat -v -count=1`
      2. Verify old JSON without dirs/prompts/manifest_path loads correctly
      3. Verify new fields are nil/empty (not populated with garbage)
    Expected Result: Old format sessions load perfectly, new fields default to zero values
    Failure Indicators: Parse error on old format; new fields have unexpected values
    Evidence: .sisyphus/evidence/task-8-session-compat.txt
  ```

  **Commit**: YES (groups with Wave 2)
  - Message: `feat(session): extend session model with dirs, prompts, and manifest_path fields`
  - Files: `internal/session/store.go`, `internal/session/store_test.go`
  - Pre-commit: `go test ./internal/session/ -count=1`

- [ ] 9. Wire everything in cmd/root.go — CLI integration

  **What to do**:
  This is the integration task that brings all pieces together in `cmd/root.go`.

  **Step-by-step changes to `cmd/root.go`:**

  **(a) Change flag declarations** (around lines 24-31):
  - Change `dirFlag string` → `dirFlags []string`
  - Add `promptFlags []string`
  - Add `manifestFlag string`
  - Change `cmd.Flags().StringVarP(&dirFlag, "dir", "d", ...)` → `cmd.Flags().StringArrayVarP(&dirFlags, "dir", "d", nil, "Working directory (repeatable); infers count")`
  - Add `cmd.Flags().StringArrayVarP(&promptFlags, "prompt", "p", nil, "Per-instance prompt (repeatable; paired with --dir by index)")`
  - Add `cmd.Flags().StringVarP(&manifestFlag, "manifest", "M", "", "YAML manifest file defining instances")`

  **(b) Manifest conflict detection** (early in RunE, before count parsing):
  - If `manifestFlag` is set AND any of (`len(dirFlags) > 0`, `len(promptFlags) > 0`, `worktreesFlag`, `len(args) > 0`) → return error: `"--manifest cannot be combined with --dir, --prompt, --worktrees, or count argument"`

  **(c) Count inference logic** (replace current lines 50-63):
  - If `manifestFlag` is set: parse manifest via `manifest.Parse(manifestPath)`, apply tilde expansion to manifestPath first. Count = len(manifest.Instances).
  - If `len(args) == 1`: parse count from args (existing logic).
  - If `len(args) == 0` and `len(dirFlags) > 0`: count = len(dirFlags).
  - If `len(args) == 0` and `len(dirFlags) == 0` and no manifest: error "count argument required when --dir or --manifest not provided".
  - If `len(args) == 1` and `len(dirFlags) > 0`: ALLOW — count from arg, dirs padded/cycled as needed.

  **(d) Dir resolution** (replace current lines 78-86):
  - If manifest: dirs come from manifest instances (already expanded by manifest parser).
  - If `len(dirFlags) > 0`: apply `pathutil.ExpandTildeAll(dirFlags)`, then `filepath.Abs()` each, then validate each exists with `os.Stat`.
  - Build `resolvedDirs []string`:
    - If len(dirFlags) == count: one-to-one mapping.
    - If len(dirFlags) < count: pad with last dir to fill remaining. (PRD rule: "remaining instances use the last --dir")
    - If len(dirFlags) > count: error "more --dir flags than instances".
  - Keep `resolvedDir` (single) as `resolvedDirs[0]` for display purposes.

  **(e) Prompt resolution**:
  - If manifest: prompts come from manifest instances.
  - If `len(promptFlags) > 0`: build `resolvedPrompts []string` with length = count. Unmatched indices get empty string.
  - If `len(promptFlags) > count`: error "more --prompt flags than instances".

  **(f) Directory validation** (after resolution, before spawning):
  - For each dir in `resolvedDirs`: `os.Stat(dir)` → if error, return `"directory does not exist: %s"`.

  **(g) Branch checkout from manifest** (after dir validation, before spawning):
  - For each manifest instance with non-empty `Branch`: run `exec.CommandContext(ctx, "git", "-C", dir, "checkout", branch)`.
  - If checkout fails: abort entire spawn with error `"failed to checkout branch %q in %s: %v"`.

  **(h) Pass data to SpawnOptions** (around lines 204-212):
  - Set `spawnOptions.Dirs = resolvedDirs`
  - Set `spawnOptions.Prompts = resolvedPrompts`
  - Remove single `Dir` from SpawnOptions (replaced by `Dirs`)

  **(i) Save session** (around lines 234-246):
  - Set `sess.Dirs = resolvedDirs`
  - Set `sess.Prompts = resolvedPrompts`
  - If manifest: `sess.ManifestPath = manifestFlag`
  - Keep `sess.Dir = resolvedDirs[0]` for backward compat with `list` command display

  **(j) Update display output** (around lines 199-202):
  - If single dir: show `"Directory: /path"` (current behavior)
  - If multiple dirs: show `"Directories: /path1, /path2, ..."` or `"Directories: N different directories"`

  **Tests** (add to `cmd/root_test.go`):
  - Repeatable `--dir`: `["--dir", "/a", "--dir", "/b"]` → count inferred as 2
  - Repeatable `--prompt`: `["2", "--dir", "/a", "--dir", "/b", "--prompt", "fix X", "--prompt", "do Y"]` → prompts paired
  - Count inference: `["--dir", "/a", "--dir", "/b"]` (no count arg) → count = 2
  - Manifest: `["--manifest", "test.yaml"]` → instances from manifest
  - Manifest conflict: `["--manifest", "test.yaml", "--dir", "/a"]` → error
  - Dir validation: `["2", "--dir", "/nonexistent"]` → error "directory does not exist"
  - Prompt count mismatch: `["2", "--dir", "/tmp", "--prompt", "a", "--prompt", "b", "--prompt", "c"]` → error
  - Backward compat: `["4"]` → works as before, `["4", "--dir", "/tmp"]` → works as before

  **Must NOT do**:
  - Do not add `--prompt-all` or `--prompts-file` flags
  - Do not modify the `TerminalBackend` interface
  - Do not change how `list`, `kill`, or `clean` commands work
  - Do not add manifest validation subcommand
  - Do not over-engineer the flag parsing — keep it readable and linear

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Large integration task touching many concerns (flags, validation, manifest, paths, git checkout, session). Needs careful ordering and comprehensive testing.
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: NO (integration — must run after all dependencies)
  - **Parallel Group**: Wave 3 (sequential)
  - **Blocks**: Task 10 (full verification)
  - **Blocked By**: Tasks 2, 3, 5, 6, 7, 8

  **References**:

  **Pattern References**:
  - `cmd/root.go:23-288` — THE file being modified. Read entire file to understand structure.
  - `cmd/root.go:24-31` — Flag variable declarations. Where new vars go.
  - `cmd/root.go:50-63` — Count parsing and validation. Major refactoring area.
  - `cmd/root.go:78-86` — Dir resolution. Replaced by multi-dir logic.
  - `cmd/root.go:99-141` — Worktree setup. Shows pattern for pre-spawn setup (follow for branch checkout).
  - `cmd/root.go:204-212` — SpawnOptions construction. Where new fields get set.
  - `cmd/root.go:234-246` — Session construction. Where new session fields get set.
  - `cmd/root.go:260-267` — Flag registration. Where new flags get registered.

  **API/Type References**:
  - `internal/manifest.Parse(path string) (Manifest, error)` — from Task 5
  - `internal/pathutil.ExpandTilde(path string) (string, error)` — from Task 3
  - `internal/pathutil.ExpandTildeAll(paths []string) ([]string, error)` — from Task 3
  - `internal/terminal/backend.go:SpawnOptions` — `Dirs`, `Prompts` fields (from Tasks 6/7)
  - `internal/session/store.go:Session` — `Dirs`, `Prompts`, `ManifestPath` fields (from Task 8)

  **Test References**:
  - `cmd/root_test.go` — from Task 2 (baseline tests). Extend with new test cases.

  **External References**:
  - Cobra docs for `StringArrayVarP`: repeatable flag that collects values into a slice

  **WHY Each Reference Matters**:
  - `root.go:50-63`: This is the MOST complex change. Current logic requires exactly 1 arg. Must handle 0 args (when dirs/manifest provide count), 1 arg (explicit count), and interaction between the two.
  - `root.go:99-141`: The worktree setup pattern shows how to do pre-spawn operations (git commands, cleanup on failure). Branch checkout follows the same pattern but without cleanup (checkout is not destructive in the same way).
  - `root.go:260-267`: Must register new flags in the correct order (before persistent flags). Cobra's `StringArrayVarP` uses `-p` shorthand which must not collide with other shorthands.

  **Acceptance Criteria**:

  **If TDD:**
  - [ ] All new tests in `cmd/root_test.go` pass
  - [ ] `go test ./cmd/ -v -count=1` → PASS (all baseline + new tests)
  - [ ] `go vet ./...` → clean
  - [ ] `go build ./...` → compiles

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Repeatable --dir with count inference
    Tool: Bash
    Preconditions: root.go modified with repeatable --dir
    Steps:
      1. Run test that sets args to `["--dir", "/tmp", "--dir", "/var"]` (no count arg)
      2. Verify count is inferred as 2
      3. Verify resolvedDirs = ["/tmp", "/var"]
    Expected Result: Count inferred from dir flags, no error
    Failure Indicators: Error about missing count; dirs not resolved
    Evidence: .sisyphus/evidence/task-9-dir-inference.txt

  Scenario: Manifest parsing and instance spawning
    Tool: Bash
    Preconditions: root.go modified with --manifest support
    Steps:
      1. Create temp YAML manifest with 2 instances
      2. Run test that sets args to `["--manifest", "tempfile.yaml"]`
      3. Verify count = 2, dirs from manifest, prompts from manifest
    Expected Result: Manifest parsed, instances configured correctly
    Failure Indicators: Parse error; wrong count; dirs not from manifest
    Evidence: .sisyphus/evidence/task-9-manifest.txt

  Scenario: Manifest conflict detection
    Tool: Bash
    Preconditions: root.go modified
    Steps:
      1. Run test that sets args to `["--manifest", "file.yaml", "--dir", "/tmp"]`
      2. Verify error message contains "--manifest cannot be combined"
    Expected Result: Clear error about conflicting flags
    Failure Indicators: No error; manifest silently overrides; unhelpful error message
    Evidence: .sisyphus/evidence/task-9-manifest-conflict.txt

  Scenario: Directory validation — nonexistent dir
    Tool: Bash
    Preconditions: root.go modified with dir validation
    Steps:
      1. Run test that sets args to `["2", "--dir", "/nonexistent/path/that/does/not/exist"]`
      2. Verify error message contains "directory does not exist"
    Expected Result: Clear error before any spawning occurs
    Failure Indicators: No validation error; windows spawn despite invalid dir
    Evidence: .sisyphus/evidence/task-9-dir-validation.txt

  Scenario: Full backward compatibility
    Tool: Bash
    Preconditions: All changes applied
    Steps:
      1. Run ALL baseline tests from Task 2
      2. Verify every single one passes
    Expected Result: Zero regressions in existing CLI behavior
    Failure Indicators: Any baseline test fails
    Evidence: .sisyphus/evidence/task-9-backward-compat.txt
  ```

  **Commit**: YES
  - Message: `feat(cli): wire multi-dir, multi-prompt, and manifest support in root command`
  - Files: `cmd/root.go`, `cmd/root_test.go`
  - Pre-commit: `make check`

- [ ] 10. Full test suite + build verification

  **What to do**:
  - Run `make check` (which executes `go vet ./...`, `go test ./... -count=1 -v`, `go build -o bin/claude-grid .`)
  - Fix any issues found (test failures, vet warnings, build errors)
  - Verify the binary compiles and the help text shows new flags (`--dir` as repeatable, `--prompt`, `--manifest`)
  - Run `./bin/claude-grid --help` and verify new flags appear in help output

  **Must NOT do**:
  - Do not add new features
  - Do not change test expectations to "fix" tests — fix the code instead
  - Do not skip/ignore failing tests

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Verification-only task. Run commands, check output, fix small issues.
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on all prior tasks)
  - **Parallel Group**: Wave 4 (sequential)
  - **Blocks**: Final Verification Wave (F1-F4)
  - **Blocked By**: Task 9

  **References**:

  **Pattern References**:
  - `Makefile:9-24` — Build, test, vet, check targets. `make check` runs all three.

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Full build + test suite passes
    Tool: Bash
    Preconditions: All tasks 1-9 complete
    Steps:
      1. Run `make check`
      2. Verify exit code is 0
      3. Count total tests and verify all pass
    Expected Result: `make check` exits 0, all tests pass, build succeeds
    Failure Indicators: Any test fails; vet warnings; build error
    Evidence: .sisyphus/evidence/task-10-make-check.txt

  Scenario: Help text shows new flags
    Tool: Bash
    Preconditions: Binary built
    Steps:
      1. Run `./bin/claude-grid --help`
      2. Verify output contains `--dir` with description mentioning "repeatable"
      3. Verify output contains `--prompt` flag
      4. Verify output contains `--manifest` flag
    Expected Result: All three new flags visible in help text with correct descriptions
    Failure Indicators: Missing flags; wrong descriptions; help text broken
    Evidence: .sisyphus/evidence/task-10-help-text.txt
  ```

  **Commit**: NO (verification only — no code changes expected)

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `go test ./... -count=1`. Review all changed files for: `as any` equivalent (empty interface{} abuse), empty catches, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp). Verify consistent code style with existing codebase.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (features working together, not isolation). Test edge cases: empty manifest, manifest with missing fields, dirs that don't exist, mixed flags. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Wave 1**: `feat(core): add pathutil tilde expansion, harden AppleScript sanitization, add cmd baseline tests` — `internal/pathutil/*.go`, `internal/script/executor.go`, `internal/script/executor_test.go`, `cmd/root_test.go`
  - Pre-commit: `go test ./internal/pathutil/ ./internal/script/ ./cmd/ -count=1`
- **Wave 2**: `feat(backends): add manifest parser, extend backends with per-window prompts, extend session model` — `internal/manifest/*.go`, `internal/terminal/backend.go`, `internal/terminal/terminal_app.go`, `internal/terminal/terminal_app_test.go`, `internal/terminal/warp.go`, `internal/terminal/warp_test.go`, `internal/session/store.go`, `internal/session/store_test.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./... -count=1`
- **Wave 3**: `feat(cli): wire multi-dir, multi-prompt, and manifest support` — `cmd/root.go`, `cmd/root_test.go`
  - Pre-commit: `make check`
- **Wave 4**: `chore: verify full test suite passes` — no code changes expected
  - Pre-commit: `make check`

---

## Success Criteria

### Verification Commands
```bash
make check                    # Expected: exit 0 (vet + test + build all pass)
go test ./... -count=1 -v     # Expected: all tests PASS
go vet ./...                  # Expected: clean, no warnings
go build -o /dev/null .       # Expected: compiles successfully
```

### Final Checklist
- [ ] All "Must Have" present (repeatable --dir, --prompt, --manifest, tilde expansion, prompt injection, session model, TDD tests)
- [ ] All "Must NOT Have" absent (no --prompt-all, no --prompts-file, no branch creation, no manifest templating, no new subcommands)
- [ ] All tests pass (`go test ./... -count=1`)
- [ ] Build succeeds (`go build ./...`)
- [ ] Old session JSON files still load correctly (backward compat)

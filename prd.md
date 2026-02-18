# PRD: `claude-grid` — Multi-Instance Claude Code Launcher & Visual Tiler

**Version:** 0.1.0  
**Author:** Ricardo  
**Date:** February 17, 2026  
**Status:** Draft

---

## 1. Problem Statement

Developers using Claude Code increasingly need to run multiple instances in parallel — whether for working on different features simultaneously, comparing approaches, or running a "team of agents" across a codebase. Today, this requires:

1. Manually opening N terminal windows
2. Navigating each to the correct directory
3. Running `claude` in each one
4. Manually resizing and arranging windows to see them all at once
5. Repeating this ritual every time you start a new session

**Existing tools don't solve this:**

| Tool                              | What it does                                                                   | What's missing                                                                                                                                          |
| --------------------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **claude-squad** (6k⭐)           | Manages multiple agents with git worktrees, TUI for switching between sessions | Single-window TUI — you can't _see_ all instances simultaneously. No visual tiling. Requires git repo. Focused on orchestration, not visual monitoring. |
| **ntm**                           | tmux pane manager for AI agents                                                | Heavy setup, tmux-only, no native macOS window support, panes get cramped at scale                                                                      |
| **multi-agent-shogun**            | Samurai-themed orchestrator with hierarchy                                     | Complex YAML config, overkill for "just give me N Claude windows"                                                                                       |
| **Claude Agent Teams** (built-in) | Experimental team coordination                                                 | Requires tmux/iTerm2, experimental flag, limited to coordinated work — not independent sessions                                                         |
| **Rectangle/Magnet**              | Generic window tiling                                                          | Manual per-window snapping, no automation, no terminal spawning                                                                                         |

**The gap:** No tool lets you run a single command to spawn N Claude Code instances, visually tiled across your screen, all from the directory you're already in.

---

## 2. Product Vision

`claude-grid` is a lightweight CLI tool that spawns and visually tiles multiple Claude Code instances with a single command. Think `tmux` meets `Rectangle` meets `claude` — but zero config, one command, instant visual grid.

```bash
# That's it. 4 Claude Code windows, perfectly tiled.
claude-grid 4

# With prompts pre-loaded
claude-grid 3 --prompt "fix all TypeScript errors" "add unit tests" "update docs"

# Different layout
claude-grid 6 --layout 3x2
```

**One-liner philosophy:** If it takes more than one command, it's too many.

---

## 3. Target Users

1. **Solo developers** running multiple Claude Code instances on different tasks in the same repo
2. **Tech leads** who want to visually monitor multiple agents working in parallel
3. **Claude Code power users** who've outgrown single-session workflows but find claude-squad/ntm too heavy
4. **Content creators/educators** who want to demo multi-agent coding workflows visually

---

## 4. Core Features (MVP — v0.1)

### 4.1 One-Command Launch

```bash
claude-grid <count> [options]
```

Spawns `<count>` terminal windows, each running `claude` in the current working directory, and tiles them in an auto-calculated grid across the screen.

**Behavior:**

- Auto-detect screen resolution and calculate optimal grid (e.g., 4 → 2×2, 6 → 3×2, 9 → 3×3)
- Each window opens in `$PWD` (or specified `--dir`)
- Each window runs `claude` (or specified `--program`)
- Windows are positioned with zero overlap and minimal gaps
- Works immediately after install with no config file

### 4.2 Smart Grid Layout

Auto-layout algorithm that maximizes readability:

```
count=1  → 1×1 (fullscreen)
count=2  → 2×1 (side by side)
count=3  → 3×1 (three columns) or 2+1 stacked
count=4  → 2×2
count=5  → 3+2 stacked
count=6  → 3×2
count=8  → 4×2
count=9  → 3×3
```

Users can override with `--layout ROWSxCOLS` (e.g., `--layout 2x3`).

### 4.3 Terminal Backend Support

Support multiple terminal backends, auto-detected in order of preference:

1. **iTerm2** (macOS) — via AppleScript API, best native experience
2. **Warp** (macOS/Linux) — via Launch Configurations YAML + URI scheme
3. **Terminal.app** (macOS) — fallback via AppleScript
4. **tmux** (cross-platform) — split panes within a single tmux session
5. **Kitty** (cross-platform) — via `kitty @ launch` remote control
6. **Alacritty + tmux** — fallback combo

Override with `--terminal <backend>`.

### 4.4 Per-Instance Prompts

Optionally send an initial prompt to each Claude instance:

```bash
# Different task per window
claude-grid 3 \
  --prompt "refactor the auth module" \
  --prompt "write tests for api/routes" \
  --prompt "update README with new API docs"

# Same prompt to all
claude-grid 4 --prompt-all "review this codebase for security issues"

# From a file (one prompt per line)
claude-grid 4 --prompts-file tasks.txt
```

### 4.5 Session Naming & Persistence

```bash
# Named session
claude-grid 4 --name "feature-sprint"

# Resume a previous session (tmux backend)
claude-grid resume feature-sprint

# List active sessions
claude-grid list

# Kill all windows in a session
claude-grid kill feature-sprint
```

---

## 5. Extended Features (v0.2+)

### 5.1 Config File Support

`~/.claude-grid.toml`:

```toml
[defaults]
count = 4
terminal = "iterm"
layout = "auto"
gap = 4                    # pixels between windows
program = "claude"

[presets.frontend]
count = 3
dir = "~/projects/webapp"
prompts = [
  "work on React components in src/components/",
  "fix CSS issues in src/styles/",
  "write Playwright tests in tests/"
]

[presets.review]
count = 4
prompt_all = "review this file for bugs and suggest improvements"
```

Usage: `claude-grid preset frontend`

### 5.2 Watch Mode / Status Bar

A lightweight status overlay or companion pane that shows:

- Which instances are active/idle/waiting for input
- Token usage per instance (if detectable)
- Quick summary of what each instance is working on

### 5.3 Broadcast Mode

Send the same keystrokes/commands to all instances simultaneously:

```bash
# From separate terminal
claude-grid broadcast feature-sprint "/clear"
claude-grid broadcast feature-sprint "focus on error handling"
```

### 5.4 Git Worktree Integration (Optional)

```bash
# Each instance gets its own worktree branch
claude-grid 3 --worktrees --branch-prefix "sprint-42"
```

Creates `sprint-42-1`, `sprint-42-2`, `sprint-42-3` branches via git worktrees, so instances don't conflict on file writes.

### 5.5 Program Agnostic

While optimized for Claude Code, support any terminal program:

```bash
claude-grid 4 --program "aider"
claude-grid 2 --program "codex"
claude-grid 6 --program "bash"  # just 6 tiled terminals
```

---

## 6. Technical Architecture

### 6.1 Language & Stack

**Recommended: Rust or Go**

| Consideration                       | Rust                        | Go                   |
| ----------------------------------- | --------------------------- | -------------------- |
| Single binary distribution          | ✅                          | ✅                   |
| Homebrew formula ease               | ✅                          | ✅                   |
| AppleScript interop                 | via `std::process::Command` | via `exec.Command`   |
| tmux control                        | via tmux CLI                | via tmux CLI         |
| Cross-platform                      | ✅                          | ✅                   |
| Ecosystem for CLIs                  | `clap` crate                | `cobra`              |
| Community preference (Claude tools) | Less common                 | claude-squad uses Go |

**Recommendation: Go** — aligns with the Claude tooling ecosystem (claude-squad is Go), fast compilation, simple cross-compilation, and `cobra` CLI framework is battle-tested.

### 6.2 Module Architecture

```
claude-grid/
├── cmd/                    # CLI entry point (cobra)
│   ├── root.go             # Main command: claude-grid <count>
│   ├── list.go             # claude-grid list
│   ├── kill.go             # claude-grid kill <name>
│   ├── resume.go           # claude-grid resume <name>
│   ├── preset.go           # claude-grid preset <name>
│   └── broadcast.go        # claude-grid broadcast <name> <msg>
├── internal/
│   ├── grid/
│   │   └── layout.go       # Grid calculation (count → rows×cols)
│   ├── screen/
│   │   ├── detect.go       # Screen resolution detection
│   │   ├── macos.go        # macOS screen info via CGDisplay
│   │   └── linux.go        # Linux screen info via xrandr/wayland
│   ├── terminal/
│   │   ├── backend.go      # Terminal backend interface
│   │   ├── iterm.go        # iTerm2 AppleScript backend
│   │   ├── warp.go         # Warp Launch Config YAML + URI + System Events backend
│   │   ├── macos_term.go   # Terminal.app AppleScript backend
│   │   ├── tmux.go         # tmux backend
│   │   ├── kitty.go        # Kitty remote control backend
│   │   └── sysevents.go    # macOS System Events universal tiling (shared by Warp, etc.)
│   ├── session/
│   │   ├── manager.go      # Session tracking & persistence
│   │   └── store.go        # Session state storage (~/.claude-grid/)
│   └── config/
│       └── config.go       # TOML config parsing
├── scripts/
│   └── applescript/        # AppleScript templates
│       ├── iterm_spawn.scpt
│       └── terminal_spawn.scpt
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── LICENSE                 # MIT
└── .goreleaser.yml         # Release automation
```

### 6.3 Terminal Backend Interface

```go
type TerminalBackend interface {
    // Name returns the backend identifier
    Name() string

    // Available checks if this backend can be used
    Available() bool

    // SpawnWindows creates N windows, each running the given command
    // in the specified directory, and positions them according to the grid
    SpawnWindows(ctx context.Context, opts SpawnOptions) ([]WindowHandle, error)

    // Tile repositions existing windows according to a new grid layout
    Tile(handles []WindowHandle, grid GridLayout, screen ScreenInfo) error

    // SendKeys sends input to a specific window
    SendKeys(handle WindowHandle, text string) error

    // Close terminates specific windows
    Close(handles []WindowHandle) error

    // CloseAll terminates all windows in a session
    CloseAll(sessionName string) error
}

type SpawnOptions struct {
    Count      int
    Command    string        // "claude" by default
    Dir        string        // working directory
    Grid       GridLayout
    Screen     ScreenInfo
    Prompts    []string      // optional per-instance prompts
    Session    string        // session name for tracking
}

type GridLayout struct {
    Rows    int
    Cols    int
    Gap     int  // pixels between windows
}

type ScreenInfo struct {
    Width       int
    Height      int
    MenuBarH    int  // macOS menu bar
    DockH       int  // macOS dock (0 if hidden or on side)
    DockPos     string // "bottom", "left", "right"
}
```

### 6.3.1 Warp Backend — Deep Dive

Warp has **no AppleScript support** (open issue since 2022). Instead, we use two Warp-native mechanisms:

**Strategy A: Launch Configurations (primary)**

Generate a temporary YAML file at `~/.warp/launch_configurations/` that defines multiple windows, each running `claude` in the target directory:

```yaml
# Auto-generated by claude-grid
# ~/.warp/launch_configurations/_claude_grid_<session>.yaml
---
name: claude-grid-<session>
windows:
  - tabs:
      - title: "claude-1"
        layout:
          cwd: /Users/ricardo/projects/my-app
        commands:
          - exec: claude
  - tabs:
      - title: "claude-2"
        layout:
          cwd: /Users/ricardo/projects/my-app
        commands:
          - exec: claude
  - tabs:
      - title: "claude-3"
        layout:
          cwd: /Users/ricardo/projects/my-app
        commands:
          - exec: claude
  - tabs:
      - title: "claude-4"
        layout:
          cwd: /Users/ricardo/projects/my-app
        commands:
          - exec: claude
```

Then trigger it via URI scheme:

```go
// Open the generated launch config
exec.Command("open", "warp://launch/_claude_grid_"+session+".yaml").Run()
```

**Strategy B: URI scheme for individual windows (fallback)**

Open N windows via repeated URI calls, then tile with AppleScript `System Events`:

```go
for i := 0; i < count; i++ {
    exec.Command("open", fmt.Sprintf("warp://action/new_window?path=%s", dir)).Run()
    time.Sleep(300 * time.Millisecond)
}
// Then tile via System Events (works for any app)
tileWithSystemEvents("Warp", handles, grid, screen)
```

**Tiling with System Events (universal fallback for non-scriptable terminals):**

Since Warp doesn't expose window positioning via its own API, we use macOS `System Events` accessibility API to position windows of _any_ application:

```applescript
tell application "System Events"
    tell process "Warp"
        set allWindows to every window
        repeat with i from 1 to count of allWindows
            set win to item i of allWindows
            set position of win to {posX, posY}
            set size of win to {winW, winH}
        end repeat
    end tell
end tell
```

This is the same approach that Rectangle/Magnet use under the hood, and it works for Warp since it's a standard macOS window.

**Warp backend detection:**

```go
func (w *WarpBackend) Available() bool {
    // Check if Warp.app exists
    _, err := os.Stat("/Applications/Warp.app")
    return err == nil
}
```

**Limitations & workarounds:**

| Limitation                                        | Workaround                                                                                |
| ------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| No AppleScript session control                    | Use Launch Config YAML + URI scheme to spawn                                              |
| No programmatic `write text` to a specific window | Use `commands:` in Launch Config for initial command; use tmux inside Warp for `SendKeys` |
| Launch Config doesn't support window positioning  | Post-spawn tiling via `System Events` accessibility API                                   |
| URI scheme can't pass dynamic commands            | Generate temporary YAML, trigger via URI, clean up after                                  |
| No window identification API                      | Match windows by creation order + title pattern matching                                  |

**Split-pane alternative (single window mode):**

For users who prefer a single Warp window with split panes instead of multiple windows, we can generate a Launch Config with panes:

```yaml
name: claude-grid-<session>-panes
windows:
  - tabs:
      - title: "claude-grid"
        layout:
          split_direction: horizontal
          panes:
            - cwd: /path/to/project
              commands:
                - exec: claude
            - cwd: /path/to/project
              commands:
                - exec: claude
```

Exposed via flag: `claude-grid 4 --mode panes` (single Warp window with 4 split panes) vs default `--mode windows` (4 separate Warp windows tiled on screen).

### 6.4 Grid Layout Algorithm

```go
func CalculateGrid(count int) GridLayout {
    // Optimize for readability: prefer wider windows (more cols than rows)
    // since terminal content is horizontal
    cols := int(math.Ceil(math.Sqrt(float64(count))))
    rows := int(math.Ceil(float64(count) / float64(cols)))

    // Special cases for common counts
    switch count {
    case 2:
        return GridLayout{Rows: 1, Cols: 2}  // side by side
    case 3:
        return GridLayout{Rows: 1, Cols: 3}  // three columns
    case 5:
        // 3 on top, 2 on bottom (handled by uneven row logic)
        return GridLayout{Rows: 2, Cols: 3}
    }

    return GridLayout{Rows: rows, Cols: cols}
}
```

For uneven grids (e.g., 5 windows in a 3×2 grid), the last row's windows should expand to fill the remaining space evenly.

### 6.5 Screen Detection (macOS)

```go
func DetectScreenMacOS() (ScreenInfo, error) {
    // Use CGDisplayBounds via cgo or system_profiler
    // Also detect dock position and size
    // Account for menu bar (typically 25px, but varies with notch)
    // Support multi-monitor: use the display where the cursor is
}
```

---

## 7. Installation & Distribution

### 7.1 Install Methods

```bash
# Homebrew (primary)
brew install claude-grid

# Go install
go install github.com/<org>/claude-grid@latest

# Curl script
curl -fsSL https://raw.githubusercontent.com/<org>/claude-grid/main/install.sh | bash

# Manual download
# GitHub Releases with prebuilt binaries for macOS (arm64, amd64) and Linux
```

### 7.2 Dependencies

**Required:**

- macOS 12+ OR Linux with X11/Wayland
- One of: iTerm2, Terminal.app, tmux, Kitty

**Optional:**

- `claude` CLI (for Claude Code; any program works)
- `git` (for worktree feature)
- `tmux` (for cross-platform backend or persistence)

---

## 8. User Experience

### 8.1 First Run

```bash
$ claude-grid 4
🔍 Detected: macOS 15.2, iTerm2 3.5, screen 2560×1600
📐 Layout: 2×2 grid (1276×760 per window)
📁 Directory: ~/projects/my-app
🚀 Spawning 4 Claude Code instances...
   ┌──────────┬──────────┐
   │ claude 1 │ claude 2 │
   │          │          │
   ├──────────┼──────────┤
   │ claude 3 │ claude 4 │
   │          │          │
   └──────────┴──────────┘
✅ Session "grid-a3f2" created. Use `claude-grid kill grid-a3f2` to close all.
```

### 8.2 With Prompts

```bash
$ claude-grid 3 --prompt "fix auth bugs" "add tests" "update docs"
🚀 Spawning 3 Claude Code instances with tasks...
   ┌────────────┬────────────┬────────────┐
   │ 🔧 fix     │ 🧪 add    │ 📝 update  │
   │ auth bugs  │ tests      │ docs       │
   └────────────┴────────────┴────────────┘
✅ Prompts sent to each instance.
```

### 8.3 Error States

```bash
$ claude-grid 4
❌ `claude` not found in PATH. Install it: npm install -g @anthropic-ai/claude-code
   Or specify a different program: claude-grid 4 --program <cmd>

$ claude-grid 4 --terminal iterm
❌ iTerm2 not found. Available backends: terminal, tmux
   Install iTerm2: brew install --cask iterm2

$ claude-grid 20
⚠️  20 windows may be hard to read on a 2560×1600 screen.
   Each window would be ~510×310 pixels. Continue? [y/N]
```

---

## 9. CLI Reference

```
claude-grid — Spawn and tile multiple Claude Code instances

USAGE:
    claude-grid <count> [flags]
    claude-grid <command> [flags]

COMMANDS:
    list                     List active sessions
    kill <session>           Kill all windows in a session
    resume <session>         Resume a previous tmux session
    preset <name>            Launch a saved preset from config
    broadcast <session> <msg> Send text to all windows in a session

FLAGS:
    -d, --dir <path>         Working directory (default: $PWD)
    -l, --layout <RxC>       Grid layout, e.g., "2x3" (default: auto)
    -p, --prompt <text>      Per-instance prompt (repeat for each)
    -P, --prompt-all <text>  Same prompt for all instances
    -f, --prompts-file <path> File with one prompt per line
    -n, --name <name>        Session name (default: auto-generated)
    -t, --terminal <backend> Terminal: iterm, warp, terminal, tmux, kitty
    -m, --mode <mode>        Window mode: "windows" (default) or "panes" (single window, Warp/iTerm2/tmux only)
    -g, --gap <px>           Gap between windows in pixels (default: 4)
    -w, --worktrees          Create git worktree per instance
    -b, --branch-prefix <s>  Branch prefix for worktrees
        --program <cmd>      Program to run (default: "claude")
        --no-tile            Spawn without tiling (for custom arrangement)
    -v, --verbose            Verbose output
    -V, --version            Show version
    -h, --help               Show help
```

---

## 10. Success Metrics

| Metric                          | Target (3 months post-launch)          |
| ------------------------------- | -------------------------------------- |
| GitHub stars                    | 1,000+                                 |
| Homebrew installs               | 500+                                   |
| Time from install to first grid | < 60 seconds                           |
| Supported terminal backends     | 4+ (iTerm2, Terminal.app, tmux, Kitty) |
| Issues resolved within 1 week   | > 80%                                  |

---

## 11. Competitive Positioning

```
                    Visual Tiling ──────────►
                    │
          claude-grid ★
          (visual + simple)
                    │
    Simplicity      │
        │           │
        ▼           │
                    │     ntm
                    │     (visual + complex)
                    │
        claude-squad
        (single TUI + orchestration)
                    │
                    │     multi-agent-shogun
                    │     (complex orchestration)
                    │
                    ▼
              Orchestration Depth ──────────►
```

**claude-grid's moat:** Dead-simple UX. One command, instant visual result. No YAML, no git requirement, no TUI to learn. The "Rectangle for Claude Code."

---

## 12. Open Questions

1. **Should we support Windows?** WSL + Windows Terminal has split-pane support, but native Windows window management via PowerShell is fragile. Defer to v0.3?

2. **Token monitoring:** Should we parse Claude Code's `/usage` output and display aggregate stats? Adds complexity but high user value.

3. **Inter-instance communication:** Should instances be able to share context (like a shared clipboard)? Or keep it pure and leave orchestration to claude-squad/agent-teams?

4. **macOS Accessibility permissions:** AppleScript window positioning requires accessibility access. How do we guide users through this gracefully?

5. **Naming:** `claude-grid` vs `cgrid` vs `grid` vs `cg`? The binary name should be short for daily use. Ship as `claude-grid` with alias `cg`?

---

## 13. Implementation Roadmap

### Phase 1 — MVP (v0.1, ~2 weeks)

- [ ] CLI scaffolding with cobra
- [ ] Screen detection (macOS)
- [ ] Grid layout calculator
- [ ] macOS System Events universal tiler (shared by all macOS backends)
- [ ] iTerm2 backend (AppleScript)
- [ ] Warp backend (Launch Config YAML + URI scheme + System Events tiling)
- [ ] Terminal.app backend (AppleScript)
- [ ] Basic session tracking
- [ ] `claude-grid <count>` works end-to-end
- [ ] README, LICENSE (MIT), Homebrew formula
- [ ] GitHub Actions for CI + goreleaser

### Phase 2 — Power Features (v0.2, +2 weeks)

- [ ] Per-instance prompts (`--prompt`)
- [ ] tmux backend (cross-platform)
- [ ] Config file support (`~/.claude-grid.toml`)
- [ ] Presets
- [ ] `list`, `kill`, `resume` commands
- [ ] Kitty backend
- [ ] Uneven grid handling (e.g., 5 → 3+2)

### Phase 3 — Advanced (v0.3, +3 weeks)

- [ ] Broadcast mode
- [ ] Git worktree integration
- [ ] Status overlay / watch mode
- [ ] Linux support (X11 via `wmctrl`, Wayland via `swaymsg`)
- [ ] Windows/WSL support
- [ ] Token usage aggregation
- [ ] Plugin system for custom backends

---

## 14. Appendix: Naming & Branding

**Suggested name:** `claude-grid`  
**Binary:** `claude-grid` (alias `cg`)  
**Tagline:** "One command. N Claude instances. Perfectly tiled."  
**Logo concept:** A grid icon with the Claude ✦ sparkle in each cell.

**Alternative names considered:**

- `cgrid` — short but unclear
- `claude-tile` — "tile" is overloaded (window managers)
- `multicode` — generic
- `claude-panes` — tmux association limits it
- `gridcode` — clean, but less discoverable

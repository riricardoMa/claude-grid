# claude-grid

> Spawn and tile multiple Claude Code instances in a grid layout with a single command.

## Features

- 🚀 **One-command launch**: `claude-grid 4` spawns 4 tiled terminal windows
- 📐 **Auto-calculated grid layouts**: 1→1×1, 2→1×2, 4→2×2, 9→3×3, etc.
- 🖥️ **Multiple terminal backends**: Terminal.app (built-in) and Warp
- 💾 **Session tracking**: List and kill sessions with `list` and `kill` commands
- 🎯 **Smart screen detection**: Automatically accounts for menu bar and Dock
- ⚡ **Zero configuration**: Works out of the box

## Installation

### Homebrew (recommended)

```bash
brew install riricardoMa/tap/claude-grid
```

### Go Install

```bash
go install github.com/riricardoMa/claude-grid@latest
```

### From Source

```bash
git clone https://github.com/riricardoMa/claude-grid.git
cd claude-grid
make install
```

## Requirements

- **macOS** 12+ (darwin)
- **Claude Code CLI**: Install with `npm install -g @anthropic-ai/claude-code`
- **Terminal.app** (built-in) or **Warp** (optional)

## Quick Start

```bash
# Spawn 4 Claude instances in a 2×2 grid
claude-grid 4

# Use specific terminal backend
claude-grid 2 --terminal warp

# Specify working directory
claude-grid 3 --dir ~/projects/my-app

# Manual layout override
claude-grid 6 --layout 3x2

# Named session for easy reference
claude-grid 4 --name my-project
```

## Usage

### Spawn Windows

```bash
claude-grid <count> [flags]
```

**Arguments:**
- `<count>` — Number of windows to spawn (1-16)

**Flags:**
- `--terminal <backend>` — Terminal backend: `terminal` or `warp` (default: auto-detect)
- `--dir <path>` — Working directory (default: current directory)
- `--name <name>` — Session name (default: auto-generated as `grid-XXXX`)
- `--layout <RxC>` — Grid layout override, e.g., `2x3` or `3X2` (default: auto-calculated)
- `--verbose` — Enable verbose output

**Examples:**

```bash
# Auto-detected backend, 2×2 grid
claude-grid 4

# Custom layout: 3 rows × 2 columns
claude-grid 6 --layout 3x2

# Warp backend with specific directory
claude-grid 2 --terminal warp --dir ~/code/project

# Named session
claude-grid 3 --name my-dev-session
```

### List Sessions

```bash
claude-grid list
```

Shows all active sessions with:
- Session name
- Backend (terminal or warp)
- Window count
- Working directory
- Creation time
- Stale indicator (if windows no longer exist)

**Example output:**
```
SESSION     BACKEND    WINDOWS  DIR                    CREATED
grid-a3f2   terminal   4        ~/projects/my-app      2026-02-17 10:30
grid-b1c4   warp       2        ~/projects/api         2026-02-17 11:15
```

### Kill Session

```bash
claude-grid kill <session-name>
```

Closes all windows in the session and removes the session file.

**Example:**
```bash
claude-grid kill grid-a3f2
```

### Version

```bash
claude-grid version
```

Shows version, commit, and build date:
```
claude-grid v0.1.0 (darwin/arm64) commit:abc123 built:2026-02-17
```

## Supported Backends

### Terminal.app

- **Availability**: Built-in macOS terminal, always available
- **Method**: Spawns via AppleScript `do script` and tiles via `bounds` property
- **Pros**: No extra installation required, stable, fast

### Warp

- **Availability**: Requires [Warp](https://www.warp.dev) installation
- **Method**: Spawns via `warp://action/new_window` URI scheme, tiles via System Events
- **Pros**: Modern terminal with GPU acceleration, collaborative features
- **Note**: First use requires granting Accessibility permission (see Troubleshooting)

### Auto-Detection

When `--terminal` is not specified, `claude-grid` automatically selects:
1. **Warp** (if `/Applications/Warp.app` exists)
2. **Terminal.app** (fallback)

## Troubleshooting

### `'claude' not found in PATH`

**Error:**
```
'claude' not found in PATH. Install: npm install -g @anthropic-ai/claude-code
```

**Solution:**
Install Claude Code CLI:
```bash
npm install -g @anthropic-ai/claude-code
```

Verify installation:
```bash
which claude
```

### Warp windows not tiling

**Issue**: Windows spawn but don't tile into a grid.

**Cause**: Warp backend uses System Events for window positioning, which requires **Accessibility permission**.

**Solution**:
1. Open **System Settings**
2. Navigate to **Privacy & Security** → **Accessibility**
3. Add your terminal app (e.g., Terminal.app, iTerm2, or the app running `claude-grid`)
4. Grant permission
5. Retry spawning

### Invalid count error

**Error:**
```
invalid count X: must be between 1 and 16
```

**Solution**: Count must be in the range 1-16:
```bash
claude-grid 4   # ✅ Valid
claude-grid 0   # ❌ Invalid (too small)
claude-grid 20  # ❌ Invalid (too large)
```

### Small window warning

**Warning:**
```
warning: small windows detected (400x300 minimum). Readability may be reduced.
```

**Explanation**: High window counts (9+) on smaller screens may result in windows smaller than 400×200 pixels. The command still proceeds, but readability may be reduced.

**Solution**: Use a smaller count or specify a custom layout with fewer rows/columns.

## Session Storage

Sessions are stored as JSON files in `~/.claude-grid/sessions/<name>.json`.

**Session file format:**
```json
{
  "name": "grid-a3f2",
  "backend": "terminal",
  "count": 4,
  "dir": "/Users/bob/projects/my-app",
  "created_at": "2026-02-17T10:30:00Z",
  "windows": [
    {"id": "12345", "index": 0},
    {"id": "12346", "index": 1},
    ...
  ]
}
```

## Development

### Build

```bash
make build
```

Binary output: `bin/claude-grid`

### Test

```bash
make test
```

Runs all unit tests with coverage.

### Pre-commit Check

```bash
make check
```

Runs `go vet`, `go test`, and `go build`.

## License

MIT

---

**Made with ❤️ for the Claude Code community**

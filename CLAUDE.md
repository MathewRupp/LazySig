# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LazySig is a terminal-based UI for capturing and analyzing SPI/I2C bus data using sigrok-cli and fx2lafw-compatible logic analyzers. Built with Bubble Tea (Go TUI framework).

## Build and Run Commands

```bash
# Build the binary
go build -o lazysig

# Run directly
./lazysig

# Clean build (if dependencies change)
go mod tidy
go build -o lazysig
```

## Architecture

### Application Flow
The application follows a state-machine pattern using Bubble Tea's Elm architecture:

1. **Screen Flow**: `screenProtocol` → `screenPinConfig` → `screenCaptureSettings` → `screenCapturing` → `screenResults`
2. **Model-View-Update**: State in `model` struct, rendering in `View()` methods, updates via `Update()`

### Two-File Structure

**main.go** - TUI interface and user interaction
- Screen definitions and state management
- Bubble Tea model/view/update pattern
- User input handling (keyboard navigation, text editing)
- Five screen views: protocol selection, pin config, capture settings, capturing (with animated spinner), results

**capture.go** - Hardware interaction and data processing
- Executes `sigrok-cli` commands with configured parameters
- Decodes `.sr` files to CSV using sigrok protocol decoders
- SPI/I2C protocol-specific parsing
- Frame filtering logic (removes empty data frames)

### Key Architectural Patterns

**State Management**: Single `model` struct contains all application state including:
- Current screen and cursor position
- Protocol-specific configuration (SPI: CLK/MOSI/MISO/CS/CPOL/CPHA, I2C: SDA/SCL/Address)
- Capture settings (sample rate, duration, output file)
- UI state (editing mode, dropdown selection, spinner animation)

**Async Capture**: Uses Bubble Tea commands to run captures in background:
```go
startCapture(m) -> runCapture() -> captureCompleteMsg
```

**Channel Mapping**: Physical pins (D0-D7) mapped to protocol signals via `--channels` flag:
- Channels renamed in sigrok command (e.g., `D0=MISO,D1=MOSI`)
- Triggers reference renamed channels (e.g., `CS=f` for falling edge)

**Protocol Decoding**: Two-stage process:
1. Capture raw data to `.sr` file with `sigrok-cli -d fx2lafw`
2. Decode with protocol decoder: `sigrok-cli -i file.sr -P spi:... -A spi`

### sigrok-cli Integration

**Hardcoded Device**: Currently uses `fx2lafw:conn=1.43` (line 42 in capture.go)
- To support multiple analyzers: scan with `sigrok-cli --scan`, parse output, add device selection screen

**SPI Decoder Output Parsing** (lines 105-174 in capture.go):
- Parses `sigrok-cli -P spi -A spi -l 3` output
- Groups MISO/MOSI by sample range (e.g., "123-456")
- Uses sample timestamps divided by sample rate for timing

**I2C Decoder Output Parsing** (lines 199-245 in capture.go):
- Simpler than SPI - direct line-by-line parsing
- Extracts i2c-1 annotations with timestamps

### CSV Output Format

**SPI**: `time,mosi,miso` - synchronized byte pairs with timestamp
**I2C**: `time,scl,sda` - protocol events with timestamps

Filter frames feature (optional): Removes rows where both MOSI and MISO don't contain valid hex bytes (regex: `\b[0-9A-Fa-f]{2}\b`)

## Important Implementation Details

**Duration Dropdown**: Uses separate state (`selectingDuration`, `durationCursor`) to handle inline dropdown without modal
- "Custom..." option switches to text edit mode
- Presets: 2000ms, 1000ms, 500ms, 250ms

**Text Editing**: Modal editing system with `editing` flag and `editBuffer`
- Enter to save, Esc to cancel
- Cursor shows as █ character during edit

**Trigger Configuration**: SPI uses CS falling edge (`-t CS=f`), I2C has no trigger
- Trigger references the renamed channel name, not physical pin (important!)

**Spinner Animation**: 50ms tick updates during capture screen
- Uses Bubble Tea's `tea.Tick()` command pattern
- Only active when `currentScreen == screenCapturing`

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework (Elm architecture)
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `sigrok-cli` - External binary (not a Go dependency) - MUST be installed on system

## Common Pitfalls

1. **Channel naming**: Triggers and decoders use renamed channels (MISO/MOSI/CLK/CS), not physical pins (D0-D3)
2. **Device hardcoding**: Device connection ID is hardcoded; will fail if analyzer is on different USB port
3. **Error handling**: Capture runs async; errors only surface via `captureCompleteMsg`
4. **Empty trace data removed**: Previously had ASCII trace view feature - was removed due to performance issues with large datasets

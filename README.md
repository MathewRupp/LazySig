# LazySig

A terminal-based UI for quickly capturing and analyzing SPI/I2C bus data using sigrok and fx2lafw-compatible logic analyzers.

## Features

- Interactive TUI for configuring captures
- Support for SPI and I2C protocols
- Configurable pin mappings
- Hardware triggering (CS falling edge for SPI)
- CSV output with decoded protocol data
- Optional filtering of empty frames
- Preset and custom capture durations

## Requirements

### Hardware
- fx2lafw-compatible logic analyzer (e.g., Saleae Logic clone, DSLogic)
- Connection to target device's SPI or I2C bus

### Software
- **sigrok-cli** - Command-line interface for sigrok
  ```bash
  # Ubuntu/Debian
  sudo apt install sigrok-cli

  # Fedora
  sudo dnf install sigrok-cli

  # macOS
  brew install sigrok-cli
  ```

- **Go 1.21+** - For building from source
  ```bash
  # Ubuntu/Debian
  sudo apt install golang-go

  # Fedora
  sudo dnf install golang

  # macOS
  brew install go
  ```

### Logic Analyzer Setup
Ensure your logic analyzer is detected by sigrok:
```bash
sigrok-cli --scan
```

You should see output like:
```
The following devices were found:
fx2lafw:conn=1.43 - fx2lafw - fx2lafw
```

## Installation

```bash
cd /path/to/LASetup
go build
sudo mv lazysig /usr/local/bin/
```

Or run directly:
```bash
./lazysig
```

## Usage

1. **Start the application:**
   ```bash
   lazysig
   ```

2. **Select protocol:**
   - Choose between SPI or I2C

3. **Configure pins:**
   - For SPI: CLK, MOSI, MISO, CS, CPOL, CPHA
   - For I2C: SDA, SCL, Address
   - Default pins work with standard fx2lafw channel mappings (D0-D3)

4. **Set capture settings:**
   - Sample Rate: Default 24 MHz
   - Duration: Choose from presets (2000ms, 1000ms, 500ms, 250ms) or custom
   - Output File: CSV filename for decoded data
   - Filter Empty: Toggle to remove frames without valid data

5. **Start capture:**
   - For SPI: Waits for CS falling edge trigger
   - For I2C: Starts immediately
   - Animated progress bar shows capture status

6. **View results:**
   - Output saved to specified CSV file
   - Press q or Esc to exit

## Output Format

### SPI CSV
```csv
time,mosi,miso
0.000000042,88,00
0.000000125,00,E4
```

### I2C CSV
```csv
time,scl,sda
0.000000042,Start,
0.000000125,Address write: 50,
```

## Pin Naming

- **D0-D7**: Physical channel pins on fx2lafw device
- Pins are mapped to protocol signals (e.g., D2=CLK for SPI)
- Default mappings:
  - SPI: CLK=D2, MOSI=D1, MISO=D0, CS=D3
  - I2C: SDA=D0, SCL=D1

## Keyboard Controls

- **↑/↓ or k/j**: Navigate options
- **Enter**: Select/Edit field
- **Tab**: Next screen (where applicable)
- **Esc**: Cancel edit or quit
- **q**: Quit application

## Troubleshooting

**Logic analyzer not detected:**
```bash
# Check USB permissions
sudo usermod -aG plugdev $USER
# Log out and back in

# Verify device
lsusb | grep -i fx2
```

**No data captured:**
- Verify pin connections to target device
- Check trigger is appropriate (CS falling edge for SPI)
- Ensure target device is active during capture window
- Increase capture duration

**Permission errors:**
```bash
# Add udev rules for fx2lafw
sudo cp /usr/share/sigrok-firmware-fx2lafw/60-libsigrok.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules
sudo udevadm trigger
```

## License

MIT

## Author

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)

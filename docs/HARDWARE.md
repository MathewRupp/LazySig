# Hardware Setup Guide

## Supported Logic Analyzers

LazySig works with any **fx2lafw-compatible** logic analyzer, including:

- Saleae Logic clones
- DSLogic series
- Generic fx2lafw-based analyzers
- Original Saleae Logic (with fx2lafw firmware)

## Physical Connections

### Pin Layout
Most fx2lafw analyzers have 8 channels labeled D0-D7.

### Connection Best Practices

1. **Keep wires short** - Long wires can pick up noise and distort signals
2. **Use ground** - Always connect the analyzer's ground to your target device's ground
3. **Check voltage levels** - Ensure your target device runs at 3.3V or 5V (most fx2lafw analyzers support both)
4. **Probe placement** - Connect probes as close to the IC pins as possible

### Example: Connecting to SPI Flash Chip

```
Logic Analyzer    →    SPI Flash
───────────────────────────────
D2 (CLK)         →    Pin 6 (CLK)
D1 (MOSI)        →    Pin 5 (SI/MOSI)
D0 (MISO)        →    Pin 2 (SO/MISO)
D3 (CS)          →    Pin 1 (CS#)
GND              →    GND
```

### Example: Connecting to I2C Device

```
Logic Analyzer    →    I2C Device
───────────────────────────────
D0 (SDA)         →    SDA
D1 (SCL)         →    SCL
GND              →    GND
```

### Example: Connecting to UART

```
Logic Analyzer    →    UART Device
───────────────────────────────
D0 (TX)          →    TX (transmit from device)
D1 (RX)          →    RX (receive to device)
GND              →    GND
```

## USB Permissions (Linux)

If you get permission errors, add your user to the `plugdev` group:

```bash
sudo usermod -aG plugdev $USER
```

Then log out and back in.

### udev Rules

For persistent permissions, install sigrok's udev rules:

```bash
sudo cp /usr/share/sigrok-firmware-fx2lafw/60-libsigrok.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules
sudo udevadm trigger
```

## Verifying Detection

Check that your analyzer is detected:

```bash
sigrok-cli --scan
```

You should see output like:
```
The following devices were found:
fx2lafw:conn=1.43 - fx2lafw - fx2lafw
```

If no devices are found:
1. Check USB cable connection
2. Try a different USB port
3. Check USB permissions (see above)
4. Verify device LED is on (if equipped)

## Multiple Analyzers

LazySig automatically detects all connected analyzers. If you have multiple devices:

1. Connect all analyzers
2. Launch LazySig
3. Select your device from the **Devices** panel

The device ID (e.g., `USB 1.43`) corresponds to the USB bus and port.

## Troubleshooting

### Analyzer Not Detected
- Check USB cable (try a different cable)
- Check USB port (try a different port)
- Check permissions (see USB Permissions above)
- Check dmesg: `dmesg | tail` after plugging in

### No Data Captured
- Verify physical connections
- Check ground connection
- Verify target device is powered and active
- Increase capture duration
- Try higher sample rate

### Corrupted Data
- Sample rate too low - increase to at least 8 MHz
- Poor ground connection
- Wires too long - use shorter wires
- Electromagnetic interference - move away from noise sources

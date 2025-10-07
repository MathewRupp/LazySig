# Protocol Guide

## SPI (Serial Peripheral Interface)

### Configuration
- **CLK** - Clock signal from master
- **MOSI** - Master Out, Slave In
- **MISO** - Master In, Slave Out
- **CS** - Chip Select (active low)
- **CPOL** - Clock polarity (0 or 1)
- **CPHA** - Clock phase (0 or 1)

### Triggering
LazySig uses a CS falling edge trigger for SPI captures. This ensures capture starts when communication begins.

### Sample Rate Recommendations
- **Low-speed SPI (<1 MHz)**: 8-16 MHz
- **Medium-speed SPI (1-5 MHz)**: 24 MHz
- **High-speed SPI (>5 MHz)**: 48 MHz

### Example Output
```csv
time,mosi,miso
0.000000042,88,00
0.000000125,00,E4
```

## I2C (Inter-Integrated Circuit)

### Configuration
- **SDA** - Serial Data line (bidirectional)
- **SCL** - Serial Clock line
- **Address** - Device address (for reference)

### Triggering
I2C captures start immediately (no hardware trigger).

### Sample Rate Recommendations
- **Standard mode (100 kHz)**: 4-8 MHz
- **Fast mode (400 kHz)**: 12-24 MHz
- **Fast mode plus (1 MHz)**: 24-48 MHz

### Example Output
```csv
time,scl,sda
0.000000042,Start,
0.000000125,Address write: 50,
0.000000208,Data write: A5,
```

## UART (Universal Asynchronous Receiver/Transmitter)

### Configuration
- **TX** - Transmit data line
- **RX** - Receive data line
- **Baud** - Baud rate (e.g., 9600, 115200)

### Triggering
UART captures start immediately (no hardware trigger).

### Sample Rate Recommendations
The sample rate should be at least 8-10x the baud rate:
- **9600 baud**: 1-2 MHz minimum
- **115200 baud**: 12-24 MHz
- **921600 baud**: 24-48 MHz

### Example Output
```csv
time,tx,rx
0.000000042,48,
0.000000125,,65
```

## General Tips

### Sample Rate Selection
- Higher sample rates provide better accuracy but generate larger files
- Minimum 4-5x oversampling is required for reliable protocol decoding
- Use the highest rate your analyzer supports for best results

### Frame Filtering
Enable frame filtering (`f` key) to remove:
- Empty frames (no data on MOSI/MISO for SPI)
- Noise and incomplete transmissions
- Protocol overhead you don't need to analyze

This is particularly useful for SPI where CS may toggle without data transfer.

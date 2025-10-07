package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type LogicAnalyzer struct {
	ID          string
	DisplayName string
}

type captureCompleteMsg struct {
	err error
}

func discoverDevices() ([]LogicAnalyzer, error) {
	cmd := exec.Command("sigrok-cli", "--scan")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to scan for devices: %w", err)
	}

	var devices []LogicAnalyzer
	lines := strings.Split(string(output), "\n")

	// Parse output like: fx2lafw:conn=1.43 - fx2lafw - fx2lafw
	re := regexp.MustCompile(`fx2lafw:conn=([0-9.]+)`)

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			connID := matches[1]
			devices = append(devices, LogicAnalyzer{
				ID:          connID,
				DisplayName: fmt.Sprintf("fx2lafw - USB %s", connID),
			})
		}
	}

	return devices, nil
}

// isValidHexData checks if a string contains valid hex byte data
// Returns true if the string contains at least one complete hex byte (00-FF)
func isValidHexData(s string) bool {
	if s == "" {
		return false
	}
	// Match hex bytes like "00", "FF", "BC 00", etc.
	hexBytePattern := regexp.MustCompile(`\b[0-9A-Fa-f]{2}\b`)
	return hexBytePattern.MatchString(s)
}

func startCapture(m model) tea.Cmd {
	return func() tea.Msg {
		err := runCapture(m)
		return captureCompleteMsg{err: err}
	}
}

func runCapture(m model) error {
	tmpFile := "capture.sr"

	// Build sigrok-cli command
	var args []string

	// Use selected device
	deviceID := "fx2lafw:conn=" + m.devices[m.selectedDevice].ID
	args = append(args, "-d", deviceID)

	// Configure channels based on protocol
	if m.protocol == ProtocolSPI {
		channels := fmt.Sprintf("%s=MISO,%s=MOSI,%s=CLK,%s=CS",
			m.spiMISO, m.spiMOSI, m.spiCLK, m.spiCS)
		args = append(args, "--channels", channels)
	} else {
		channels := fmt.Sprintf("%s=SDA,%s=SCL",
			m.i2cSDA, m.i2cSCL)
		args = append(args, "--channels", channels)
	}

	args = append(args, "--config", "samplerate="+m.sampleRate)

	// Add trigger on CS falling edge for SPI
	if m.protocol == ProtocolSPI {
		args = append(args, "-t", "CS=f")
	}

	args = append(args, "--time", m.duration)
	args = append(args, "-o", tmpFile)

	// Run capture
	cmd := exec.Command("sigrok-cli", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	// Decode to CSV
	if err := decodeToCSV(tmpFile, m.outputFile, m.protocol, m); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	return nil
}

func decodeToCSV(srFile, outputFile string, protocol Protocol, m model) error {
	// Create output CSV
	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	if protocol == ProtocolSPI {
		// Use sigrok-cli to decode SPI - show all annotations
		cmd := exec.Command("sigrok-cli", "-i", srFile,
			"-P", "spi:clk=CLK:mosi=MOSI:miso=MISO:cs=CS:wordsize=8",
			"-A", "spi",
			"-l", "3")

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("SPI decode failed: %w", err)
		}

		// Write header
		writer.Write([]string{"time", "mosi", "miso"})

		// Parse SPI decoder output
		lines := strings.Split(string(output), "\n")
		sampleRate, _ := strconv.ParseFloat(m.sampleRate, 64)

		// Track bytes - group MOSI/MISO that occur at same time
		type spiData struct {
			timestamp float64
			mosi      string
			miso      string
		}
		dataMap := make(map[string]*spiData)

		for _, line := range lines {
			// Only process lines that start with sample numbers
			if !strings.Contains(line, "-") {
				continue
			}

			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}

			// Check if this is a data line (not debug output)
			if !strings.HasPrefix(parts[0], "cli:") && strings.Contains(line, "spi-1:") {
				// Extract sample range (e.g., "123-456")
				sampleRange := parts[0]
				samples := strings.Split(sampleRange, "-")
				if len(samples) < 1 {
					continue
				}

				startSample, err := strconv.ParseFloat(samples[0], 64)
				if err != nil {
					continue
				}

				timestamp := startSample / sampleRate
				key := sampleRange

				// Get or create data entry
				if dataMap[key] == nil {
					dataMap[key] = &spiData{timestamp: timestamp}
				}

				// Extract the data value (everything after "spi-1: ")
				spiIdx := strings.Index(line, "spi-1:")
				if spiIdx == -1 {
					continue
				}

				dataStr := strings.TrimSpace(line[spiIdx+6:])

				// Remove quotes if present
				dataStr = strings.Trim(dataStr, "\"")

				// If empty string, skip this line - no actual data
				if dataStr == "" {
					continue
				}

				// Determine if this is MOSI or MISO based on position in output
				// (sigrok outputs MISO first, then MOSI)
				if dataMap[key].miso == "" {
					dataMap[key].miso = dataStr
				} else if dataMap[key].mosi == "" {
					dataMap[key].mosi = dataStr
				}
			}
		}

		// Write out the data
		for _, data := range dataMap {
			if data.mosi != "" || data.miso != "" {
				// Apply filtering if enabled
				if m.filterFrames {
					// Skip frames that don't have valid hex data bytes
					// Valid data should be at least "00" or contain hex digits
					mosiValid := isValidHexData(data.mosi)
					misoValid := isValidHexData(data.miso)

					if !mosiValid && !misoValid {
						continue
					}
				}

				writer.Write([]string{
					fmt.Sprintf("%.9f", data.timestamp),
					data.mosi,
					data.miso,
				})
			}
		}
	} else {
		// I2C decoding
		var stdout strings.Builder
		cmd := exec.Command("sigrok-cli", "-i", srFile,
			"-P", "i2c:scl=SCL:sda=SDA",
			"-A", "i2c",
			"-l", "3")
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("I2C decode failed: %w", err)
		}

		// Write header
		writer.Write([]string{"time", "scl", "sda"})

		// Parse I2C decoder output
		lines := strings.Split(stdout.String(), "\n")
		sampleRate, _ := strconv.ParseFloat(m.sampleRate, 64)

		for _, line := range lines {
			if strings.Contains(line, "i2c-1:") {
				parts := strings.Fields(line)
				if len(parts) < 3 {
					continue
				}

				sampleRange := parts[0]
				samples := strings.Split(sampleRange, "-")
				if len(samples) < 1 {
					continue
				}

				startSample, err := strconv.ParseFloat(samples[0], 64)
				if err != nil {
					continue
				}

				timestamp := startSample / sampleRate
				data := strings.Join(parts[2:], " ")

				writer.Write([]string{
					fmt.Sprintf("%.9f", timestamp),
					data,
					"",
				})
			}
		}
	}

	return nil
}

func generateASCIITrace(srFile string) []string {
	cmd := exec.Command("sigrok-cli", "-i", srFile, "-O", "ascii")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []string{"Error generating ASCII trace: " + err.Error()}
	}

	lines := strings.Split(string(output), "\n")

	// Concatenate channel data - sigrok outputs blocks of 74 chars repeatedly
	channelData := make(map[string]string)

	for _, line := range lines {
		if strings.HasPrefix(line, "MISO:") {
			channelData["MISO"] += strings.TrimPrefix(line, "MISO:")
		} else if strings.HasPrefix(line, "MOSI:") {
			channelData["MOSI"] += strings.TrimPrefix(line, "MOSI:")
		} else if strings.HasPrefix(line, "CLK:") {
			channelData["CLK"] += strings.TrimPrefix(line, "CLK:")
		} else if strings.HasPrefix(line, "CS:") {
			channelData["CS"] += strings.TrimPrefix(line, "CS:")
		} else if strings.HasPrefix(line, "SDA:") {
			channelData["SDA"] += strings.TrimPrefix(line, "SDA:")
		} else if strings.HasPrefix(line, "SCL:") {
			channelData["SCL"] += strings.TrimPrefix(line, "SCL:")
		}
	}

	// Build result with channel labels
	result := []string{}
	if data, ok := channelData["MISO"]; ok && data != "" {
		result = append(result, "MISO:"+data)
	}
	if data, ok := channelData["MOSI"]; ok && data != "" {
		result = append(result, "MOSI:"+data)
	}
	if data, ok := channelData["CLK"]; ok && data != "" {
		result = append(result, "CLK:"+data)
	}
	if data, ok := channelData["CS"]; ok && data != "" {
		result = append(result, "CS:"+data)
	}
	if data, ok := channelData["SDA"]; ok && data != "" {
		result = append(result, "SDA:"+data)
	}
	if data, ok := channelData["SCL"]; ok && data != "" {
		result = append(result, "SCL:"+data)
	}

	return result
}

package main

import (
	"fmt"
	"strconv"
	"strings"
)

func formatSampleRate(rate string) string {
	// Handle special case
	if rate == "Custom..." {
		return rate
	}

	// Convert sample rate to human readable format
	rateInt, err := strconv.Atoi(rate)
	if err != nil {
		return rate
	}

	if rateInt >= 1000000 {
		return fmt.Sprintf("%d MHz", rateInt/1000000)
	} else if rateInt >= 1000 {
		return fmt.Sprintf("%d kHz", rateInt/1000)
	}
	return fmt.Sprintf("%d Hz", rateInt)
}

func (m model) renderDevicesPanel(width, height int) string {
	isActive := m.activePanel == panelDevices
	style := inactivePanelStyle
	if isActive {
		style = activePanelStyle
	}

	var content strings.Builder
	content.WriteString(panelTitleStyle.Render("Devices") + "\n\n")

	if len(m.devices) == 0 {
		content.WriteString(dimTextStyle.Render("No devices found"))
	} else {
		for i, device := range m.devices {
			cursor := " "
			text := device.DisplayName
			if i == m.selectedDevice {
				text = "• " + text
			}
			if isActive && m.cursor == i {
				cursor = ">"
				text = selectedStyle.Render(text)
			}
			content.WriteString(fmt.Sprintf("%s %s\n", cursor, text))
		}
	}

	return style.Width(width).Height(height).Render(content.String())
}

func (m model) renderConfigPanel(width, height int) string {
	isActive := m.activePanel == panelConfiguration
	style := inactivePanelStyle
	if isActive {
		style = activePanelStyle
	}

	var content strings.Builder
	content.WriteString(panelTitleStyle.Render("Configuration") + "\n\n")

	// Protocol selection
	protocolText := "Protocol: "
	switch m.protocol {
	case ProtocolSPI:
		protocolText += "SPI"
	case ProtocolI2C:
		protocolText += "I2C"
	case ProtocolUART:
		protocolText += "UART"
	}
	if isActive && m.cursor == 0 {
		content.WriteString("> " + selectedStyle.Render(protocolText) + "\n\n")
	} else {
		content.WriteString("  " + protocolText + "\n\n")
	}

	// Pin configuration
	if m.protocol == ProtocolSPI {
		fields := []struct{ label, value string }{
			{"CLK", m.spiCLK},
			{"MOSI", m.spiMOSI},
			{"MISO", m.spiMISO},
			{"CS", m.spiCS},
			{"CPOL", m.spiCPOL},
			{"CPHA", m.spiCPHA},
		}

		for i, field := range fields {
			cursor := " "
			value := field.value
			if isActive && m.cursor == i+1 {
				cursor = ">"
				if m.editing {
					value = m.editBuffer + "█"
				}
				value = selectedStyle.Render(value)
			}
			content.WriteString(fmt.Sprintf("%s %-4s: %s\n", cursor, field.label, value))
		}
	} else if m.protocol == ProtocolI2C {
		fields := []struct{ label, value string }{
			{"SDA", m.i2cSDA},
			{"SCL", m.i2cSCL},
			{"Addr", m.i2cAddress},
		}

		for i, field := range fields {
			cursor := " "
			value := field.value
			if isActive && m.cursor == i+1 {
				cursor = ">"
				if m.editing {
					value = m.editBuffer + "█"
				}
				value = selectedStyle.Render(value)
			}
			content.WriteString(fmt.Sprintf("%s %-4s: %s\n", cursor, field.label, value))
		}
	} else if m.protocol == ProtocolUART {
		fields := []struct{ label, value string }{
			{"TX", m.uartTX},
			{"RX", m.uartRX},
			{"Baud", m.uartBaud},
		}

		for i, field := range fields {
			cursor := " "
			value := field.value
			if isActive && m.cursor == i+1 {
				cursor = ">"
				if m.editing {
					value = m.editBuffer + "█"
				}
				value = selectedStyle.Render(value)
			}
			content.WriteString(fmt.Sprintf("%s %-4s: %s\n", cursor, field.label, value))
		}
	}

	return style.Width(width).Height(height).Render(content.String())
}

func (m model) renderCapturePanel(width, height int) string {
	isActive := m.activePanel == panelCaptureSettings
	style := inactivePanelStyle
	if isActive {
		style = activePanelStyle
	}

	var content strings.Builder
	content.WriteString(panelTitleStyle.Render("Capture") + "\n\n")

	// Format sample rate nicely
	sampleRateDisplay := formatSampleRate(m.sampleRate)

	fields := []struct{ label, value string }{
		{"Rate", sampleRateDisplay},
		{"Duration", m.duration},
		{"Output", m.outputFile},
	}

	for i, field := range fields {
		cursor := " "
		value := field.value
		if isActive && m.cursor == i {
			cursor = ">"
			if i == 0 && m.selectingSampleRate {
				value = selectedStyle.Render(value + " ▼")
			} else if i == 1 && m.selectingDuration {
				value = selectedStyle.Render(value + " ▼")
			} else if m.editing {
				value = m.editBuffer + "█"
			} else {
				value = selectedStyle.Render(value)
			}
		}
		content.WriteString(fmt.Sprintf("%s %-8s: %s\n", cursor, field.label, value))

		// Show sample rate dropdown
		if i == 0 && isActive && m.cursor == 0 && m.selectingSampleRate {
			for j, opt := range m.sampleRateOptions {
				dropdownCursor := "  "
				optText := formatSampleRate(opt)
				if j == m.sampleRateCursor {
					dropdownCursor = "  ▸"
					optText = selectedStyle.Render(optText)
				}
				content.WriteString(fmt.Sprintf("%s %s\n", dropdownCursor, optText))
			}
		}

		// Show duration dropdown
		if i == 1 && isActive && m.cursor == 1 && m.selectingDuration {
			for j, opt := range m.durationOptions {
				dropdownCursor := "  "
				optText := opt
				if j == m.durationCursor {
					dropdownCursor = "  ▸"
					optText = selectedStyle.Render(opt)
				}
				content.WriteString(fmt.Sprintf("%s %s\n", dropdownCursor, optText))
			}
		}
	}

	// Filter toggle
	cursor := " "
	filterText := fmt.Sprintf("Filter: %s", map[bool]string{true: "ON", false: "OFF"}[m.filterFrames])
	if isActive && m.cursor == 3 {
		cursor = ">"
		filterText = selectedStyle.Render(filterText)
	}
	content.WriteString(fmt.Sprintf("\n%s %s\n", cursor, filterText))

	// Start button
	cursor = " "
	startText := "[Start Capture]"
	if isActive && m.cursor == 4 {
		cursor = ">"
		startText = selectedStyle.Render(startText)
	}
	content.WriteString(fmt.Sprintf("%s %s\n", cursor, startText))

	return style.Width(width).Height(height).Render(content.String())
}

func (m model) renderOutputPanel(width, height int) string {
	isActive := m.activePanel == panelOutput
	style := inactivePanelStyle
	if isActive {
		style = activePanelStyle
	}

	var content strings.Builder
	content.WriteString(panelTitleStyle.Render("Output") + "\n\n")

	if m.capturing {
		// Show spinner
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner := frames[m.captureSpinner%len(frames)]
		content.WriteString(fmt.Sprintf("%s Capturing data...\n\n", spinner))

		// Progress bar
		barWidth := width - 8
		if barWidth > 60 {
			barWidth = 60
		}
		pos := m.captureSpinner % (barWidth * 2)
		if pos >= barWidth {
			pos = barWidth*2 - pos - 1
		}

		bar := ""
		for i := 0; i < barWidth; i++ {
			if i == pos {
				bar += "█"
			} else if i > pos-3 && i < pos {
				bar += "▓"
			} else if i > pos && i < pos+3 {
				bar += "▓"
			} else {
				bar += "░"
			}
		}
		content.WriteString("[" + bar + "]")
	} else if len(m.outputData) > 0 {
		// Show captured data
		maxLines := height - 4
		for i := 0; i < len(m.outputData) && i < maxLines; i++ {
			line := m.outputData[i]
			if len(line) > width-6 {
				line = line[:width-9] + "..."
			}
			content.WriteString(line + "\n")
		}
		if len(m.outputData) > maxLines {
			content.WriteString(dimTextStyle.Render(fmt.Sprintf("\n... %d more lines", len(m.outputData)-maxLines)))
		}
	} else {
		content.WriteString(dimTextStyle.Render("No data captured yet"))
	}

	return style.Width(width).Height(height).Render(content.String())
}

func (m model) renderStatusPanel(width, height int) string {
	isActive := m.activePanel == panelStatus
	style := inactivePanelStyle
	if isActive {
		style = activePanelStyle
	}

	// Status message with color (compact, no title)
	statusStyle := normalTextStyle
	if strings.Contains(m.statusMsg, "Error") || strings.Contains(m.statusMsg, "failed") {
		statusStyle = errorStyle
	} else if strings.Contains(m.statusMsg, "complete") || strings.Contains(m.statusMsg, "Ready") {
		statusStyle = successStyle
	}

	content := statusStyle.Render(m.statusMsg)

	return style.Width(width).Height(height).Render(content)
}

func (m model) renderStatusBar() string {
	helpText := "s: start • f: filter • d: duration • tab: next panel • 1-5: jump • ↑↓/jk: navigate • q: quit"
	if m.editing {
		helpText = "enter: save • esc: cancel"
	} else if m.selectingDuration || m.selectingSampleRate {
		helpText = "↑↓/jk: select • enter: confirm • esc: cancel"
	} else if m.capturing {
		helpText = "Capturing... please wait"
	}

	return statusBarStyle.Width(m.width).Render(helpText)
}

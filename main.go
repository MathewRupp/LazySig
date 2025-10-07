package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Protocol int

const (
	ProtocolSPI Protocol = iota
	ProtocolI2C
)

type tickMsg time.Time

type screen int

const (
	screenDeviceSelection screen = iota
	screenProtocol
	screenPinConfig
	screenCaptureSettings
	screenCapturing
	screenResults
)

type model struct {
	currentScreen screen
	cursor        int
	protocol      Protocol

	// Device selection
	devices        []LogicAnalyzer
	selectedDevice int

	// SPI config
	spiCLK      string
	spiMOSI     string
	spiMISO     string
	spiCS       string
	spiCPOL     string // Clock polarity (0 or 1)
	spiCPHA     string // Clock phase (0 or 1)

	// I2C config
	i2cSDA     string
	i2cSCL     string
	i2cAddress string

	// Capture settings
	duration     string
	outputFile   string
	sampleRate   string
	filterFrames bool // Filter out frames without valid data bytes

	// State
	err            error
	captureFile    string
	editing        bool
	editBuffer     string
	captureSpinner int  // For animated progress during capture
	selectingDuration bool // True when selecting from duration dropdown
	durationOptions []string
	durationCursor  int
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

func initialModel() model {
	devices, err := discoverDevices()
	if err != nil {
		devices = []LogicAnalyzer{}
	}

	return model{
		currentScreen: screenDeviceSelection,
		cursor:        0,
		devices:       devices,
		selectedDevice: 0,
		spiCLK:        "D2",
		spiMOSI:       "D1",
		spiMISO:       "D0",
		spiCS:         "D3",
		spiCPOL:       "0",
		spiCPHA:       "0",
		i2cSDA:        "D0",
		i2cSCL:        "D1",
		i2cAddress:    "0x50",
		duration:      "500ms",
		outputFile:    "output.csv",
		sampleRate:    "24000000",
		filterFrames:  false,
		durationOptions: []string{"2000ms", "1000ms", "500ms", "250ms", "Custom..."},
		durationCursor:  2, // Default to 500ms
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle duration dropdown selection
		if m.selectingDuration {
			switch msg.String() {
			case "up", "k":
				if m.durationCursor > 0 {
					m.durationCursor--
				}
			case "down", "j":
				if m.durationCursor < len(m.durationOptions)-1 {
					m.durationCursor++
				}
			case "enter":
				selected := m.durationOptions[m.durationCursor]
				if selected == "Custom..." {
					// Switch to editing mode for custom duration
					m.selectingDuration = false
					m.editing = true
					m.editBuffer = m.duration
				} else {
					m.duration = selected
					m.selectingDuration = false
				}
			case "esc":
				m.selectingDuration = false
			}
			return m, nil
		}

		// Handle editing mode
		if m.editing {
			switch msg.String() {
			case "enter":
				m.saveEdit()
				m.editing = false
				m.editBuffer = ""
			case "esc":
				m.editing = false
				m.editBuffer = ""
			case "backspace":
				if len(m.editBuffer) > 0 {
					m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
				}
			default:
				// Add character to buffer
				if len(msg.String()) == 1 {
					m.editBuffer += msg.String()
				}
			}
			return m, nil
		}

		// Normal navigation mode
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			// Allow escape to quit from results screen
			if m.currentScreen == screenResults {
				return m, tea.Quit
			}
		case "up", "k":
			if m.currentScreen != screenResults && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.currentScreen != screenResults {
				m.cursor++
			}
		case "tab":
			// Skip to next screen
			if m.currentScreen == screenPinConfig {
				m.currentScreen = screenCaptureSettings
				m.cursor = 0
			}
		case "enter":
			return m.handleEnter()
		}
	case tickMsg:
		// Update spinner animation during capture
		if m.currentScreen == screenCapturing {
			m.captureSpinner++
			return m, tick()
		}
	case captureCompleteMsg:
		m.err = msg.err
		m.currentScreen = screenResults
		return m, nil
	}
	return m, nil
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case screenDeviceSelection:
		if len(m.devices) > 0 && m.cursor < len(m.devices) {
			m.selectedDevice = m.cursor
			m.currentScreen = screenProtocol
			m.cursor = 0
		}
	case screenProtocol:
		if m.cursor == 0 {
			m.protocol = ProtocolSPI
		} else {
			m.protocol = ProtocolI2C
		}
		m.currentScreen = screenPinConfig
		m.cursor = 0
	case screenPinConfig:
		// Start editing the selected field
		m.editing = true
		m.editBuffer = m.getCurrentPinValue()
	case screenCaptureSettings:
		// Check if on "Start Capture" option
		maxCursor := 3
		if m.cursor == 0 {
			// Cursor 0: Sample Rate - text edit
			m.editing = true
			m.editBuffer = m.getCurrentSettingValue()
		} else if m.cursor == 1 {
			// Cursor 1: Duration - dropdown
			m.selectingDuration = true
			// Set cursor to current duration option
			for i, opt := range m.durationOptions {
				if opt == m.duration {
					m.durationCursor = i
					break
				}
			}
		} else if m.cursor == 2 {
			// Cursor 2: Output File - text edit
			m.editing = true
			m.editBuffer = m.getCurrentSettingValue()
		} else if m.cursor == maxCursor {
			// Cursor 3: toggle filter frames
			m.filterFrames = !m.filterFrames
		} else {
			// Cursor 4: Start capture
			m.currentScreen = screenCapturing
			m.captureSpinner = 0
			return m, tea.Batch(startCapture(m), tick())
		}
	}
	return m, nil
}

func (m *model) saveEdit() {
	switch m.currentScreen {
	case screenPinConfig:
		if m.protocol == ProtocolSPI {
			switch m.cursor {
			case 0:
				m.spiCLK = m.editBuffer
			case 1:
				m.spiMOSI = m.editBuffer
			case 2:
				m.spiMISO = m.editBuffer
			case 3:
				m.spiCS = m.editBuffer
			case 4:
				m.spiCPOL = m.editBuffer
			case 5:
				m.spiCPHA = m.editBuffer
			}
		} else {
			switch m.cursor {
			case 0:
				m.i2cSDA = m.editBuffer
			case 1:
				m.i2cSCL = m.editBuffer
			case 2:
				m.i2cAddress = m.editBuffer
			}
		}
	case screenCaptureSettings:
		switch m.cursor {
		case 0:
			m.sampleRate = m.editBuffer
		case 1:
			m.duration = m.editBuffer
		case 2:
			m.outputFile = m.editBuffer
		}
	}
}

func (m model) getCurrentPinValue() string {
	if m.protocol == ProtocolSPI {
		switch m.cursor {
		case 0:
			return m.spiCLK
		case 1:
			return m.spiMOSI
		case 2:
			return m.spiMISO
		case 3:
			return m.spiCS
		case 4:
			return m.spiCPOL
		case 5:
			return m.spiCPHA
		}
	} else {
		switch m.cursor {
		case 0:
			return m.i2cSDA
		case 1:
			return m.i2cSCL
		case 2:
			return m.i2cAddress
		}
	}
	return ""
}

func (m model) getCurrentSettingValue() string {
	switch m.cursor {
	case 0:
		return m.sampleRate
	case 1:
		return m.duration
	case 2:
		return m.outputFile
	}
	return ""
}

func (m model) View() string {
	switch m.currentScreen {
	case screenDeviceSelection:
		return m.viewDeviceSelection()
	case screenProtocol:
		return m.viewProtocolSelection()
	case screenPinConfig:
		return m.viewPinConfig()
	case screenCaptureSettings:
		return m.viewCaptureSettings()
	case screenCapturing:
		return m.viewCapturing()
	case screenResults:
		return m.viewResults()
	}
	return ""
}

func (m model) viewDeviceSelection() string {
	s := titleStyle.Render("Select Logic Analyzer") + "\n\n"

	if len(m.devices) == 0 {
		s += "No devices found. Please connect a logic analyzer.\n\n"
		s += helpStyle.Render("q: quit")
		return s
	}

	for i, device := range m.devices {
		cursor := " "
		displayName := device.DisplayName
		if m.cursor == i {
			cursor = ">"
			displayName = selectedStyle.Render(displayName)
		}
		s += fmt.Sprintf("%s %s\n", cursor, displayName)
	}

	s += "\n" + helpStyle.Render("↑/↓: navigate • enter: select • q: quit")
	return s
}

func (m model) viewProtocolSelection() string {
	s := titleStyle.Render("Select Protocol") + "\n\n"

	choices := []string{"SPI", "I2C"}
	for i, choice := range choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			choice = selectedStyle.Render(choice)
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	s += "\n" + helpStyle.Render("↑/↓: navigate • enter: select • q: quit")
	return s
}

func (m model) viewPinConfig() string {
	var s string

	if m.protocol == ProtocolSPI {
		s = titleStyle.Render("SPI Pin Configuration") + "\n\n"

		fields := []struct{label, value string}{
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
			if m.cursor == i {
				cursor = ">"
				if m.editing {
					value = m.editBuffer + "█"
				} else {
					value = selectedStyle.Render(value)
				}
			}
			s += fmt.Sprintf("%s %-4s: %s\n", cursor, field.label, value)
		}
	} else {
		s = titleStyle.Render("I2C Pin Configuration") + "\n\n"

		fields := []struct{label, value string}{
			{"SDA", m.i2cSDA},
			{"SCL", m.i2cSCL},
			{"Address", m.i2cAddress},
		}

		for i, field := range fields {
			cursor := " "
			value := field.value
			if m.cursor == i {
				cursor = ">"
				if m.editing {
					value = m.editBuffer + "█"
				} else {
					value = selectedStyle.Render(value)
				}
			}
			s += fmt.Sprintf("%s %-7s: %s\n", cursor, field.label, value)
		}
	}

	if m.editing {
		s += "\n" + helpStyle.Render("enter: save • esc: cancel")
	} else {
		s += "\n" + helpStyle.Render("enter: edit • tab: next screen • q: quit")
	}
	return s
}

func (m model) viewCaptureSettings() string {
	s := titleStyle.Render("Capture Settings") + "\n\n"

	fields := []struct{label, value string}{
		{"Sample Rate", m.sampleRate + " Hz"},
		{"Duration", m.duration},
		{"Output File", m.outputFile},
	}

	for i, field := range fields {
		cursor := " "
		value := field.value
		if m.cursor == i {
			cursor = ">"
			if i == 1 && m.selectingDuration {
				// Show dropdown for duration
				value = selectedStyle.Render(value + " ▼")
			} else if m.editing {
				// Show edit buffer for any field being edited
				if i == 0 {
					// Remove " Hz" suffix for editing sample rate
					value = m.editBuffer + "█"
				} else {
					value = m.editBuffer + "█"
				}
			} else {
				value = selectedStyle.Render(value)
			}
		}
		s += fmt.Sprintf("%s %-11s: %s\n", cursor, field.label, value)

		// Show dropdown menu under Duration field
		if i == 1 && m.cursor == 1 && m.selectingDuration {
			for j, opt := range m.durationOptions {
				dropdownCursor := "  "
				optText := opt
				if j == m.durationCursor {
					dropdownCursor = "  ▸"
					optText = selectedStyle.Render(opt)
				}
				s += fmt.Sprintf("%s %s\n", dropdownCursor, optText)
			}
		}
	}

	// Add filter frames toggle
	cursor := " "
	filterText := "Filter Empty: "
	if m.filterFrames {
		filterText += "ON"
	} else {
		filterText += "OFF"
	}
	if m.cursor == 3 {
		cursor = ">"
		filterText = selectedStyle.Render(filterText)
	}
	s += fmt.Sprintf("\n%s %s\n", cursor, filterText)

	// Add "Start Capture" option
	cursor = " "
	startText := "Start Capture"
	if m.cursor == 4 {
		cursor = ">"
		startText = selectedStyle.Render(startText)
	}
	s += fmt.Sprintf("%s %s\n", cursor, startText)

	if m.editing {
		s += "\n" + helpStyle.Render("enter: save • esc: cancel")
	} else {
		s += "\n" + helpStyle.Render("enter: edit/start • q: quit")
	}
	return s
}

func (m model) viewCapturing() string {
	// Spinner frames
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinner := frames[m.captureSpinner%len(frames)]

	// Progress bar animation
	barWidth := 40
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

	return titleStyle.Render("Capturing Data") + "\n\n" +
		fmt.Sprintf("%s Waiting for trigger or capturing data...\n\n", spinner) +
		"[" + bar + "]"
}

func (m model) viewResults() string {
	if m.err != nil {
		return titleStyle.Render("Error") + "\n\n" + m.err.Error()
	}
	return titleStyle.Render("Capture Complete") + "\n\n" +
		"Output saved to: " + m.outputFile + "\n\n" +
		helpStyle.Render("q/esc: quit")
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

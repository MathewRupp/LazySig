package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Protocol int

const (
	ProtocolSPI Protocol = iota
	ProtocolI2C
	ProtocolUART
)

type panel int

const (
	panelDevices panel = iota
	panelConfiguration
	panelCaptureSettings
	panelOutput
	panelStatus
)

type tickMsg time.Time

type model struct {
	activePanel panel
	cursor      int
	protocol    Protocol

	// Device selection
	devices        []LogicAnalyzer
	selectedDevice int

	// SPI config
	spiCLK  string
	spiMOSI string
	spiMISO string
	spiCS   string
	spiCPOL string // Clock polarity (0 or 1)
	spiCPHA string // Clock phase (0 or 1)

	// I2C config
	i2cSDA     string
	i2cSCL     string
	i2cAddress string

	// UART config
	uartTX   string
	uartRX   string
	uartBaud string

	// Capture settings
	duration     string
	outputFile   string
	sampleRate   string
	filterFrames bool // Filter out frames without valid data bytes

	// State
	capturing      bool
	captureErr     error
	outputData     []string // Captured output lines
	statusMsg      string
	editing        bool
	editBuffer     string
	captureSpinner int // For animated progress during capture
	selectingDuration bool // True when selecting from duration dropdown
	durationOptions   []string
	durationCursor    int
	selectingSampleRate bool // True when selecting from sample rate dropdown
	sampleRateOptions   []string
	sampleRateCursor    int

	// UI dimensions
	width  int
	height int
}

var (
	// Panel styles
	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("170")).
				Padding(1, 2)

	inactivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	normalTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)
)

func initialModel() model {
	devices, err := discoverDevices()
	if err != nil {
		devices = []LogicAnalyzer{}
	}

	statusMsg := "Ready"
	if len(devices) == 0 {
		statusMsg = "No devices found"
	}

	return model{
		activePanel:    panelDevices,
		cursor:         0,
		devices:        devices,
		selectedDevice: 0,
		protocol:       ProtocolSPI,
		spiCLK:         "D2",
		spiMOSI:        "D1",
		spiMISO:        "D0",
		spiCS:          "D3",
		spiCPOL:        "0",
		spiCPHA:        "0",
		i2cSDA:         "D0",
		i2cSCL:         "D1",
		i2cAddress:     "0x50",
		uartTX:         "D0",
		uartRX:         "D1",
		uartBaud:       "115200",
		duration:       "500ms",
		outputFile:     "output.csv",
		sampleRate:     "24000000",
		filterFrames:   false,
		durationOptions:     []string{"2000ms", "1000ms", "500ms", "250ms", "Custom..."},
		durationCursor:      2, // Default to 500ms
		sampleRateOptions:   []string{"48000000", "24000000", "16000000", "12000000", "8000000", "6000000", "4000000", "2000000", "1000000", "Custom..."},
		sampleRateCursor:    1, // Default to 24MHz
		statusMsg:           statusMsg,
		outputData:          []string{},
		capturing:           false,
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		// Handle sample rate dropdown selection
		if m.selectingSampleRate {
			switch msg.String() {
			case "up", "k":
				if m.sampleRateCursor > 0 {
					m.sampleRateCursor--
				}
			case "down", "j":
				if m.sampleRateCursor < len(m.sampleRateOptions)-1 {
					m.sampleRateCursor++
				}
			case "enter":
				selected := m.sampleRateOptions[m.sampleRateCursor]
				if selected == "Custom..." {
					// Switch to editing mode for custom sample rate
					m.selectingSampleRate = false
					m.editing = true
					m.editBuffer = m.sampleRate
				} else {
					m.sampleRate = selected
					m.selectingSampleRate = false
				}
			case "esc":
				m.selectingSampleRate = false
			}
			return m, nil
		}

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
		case "s":
			// Quick start capture with current settings
			if len(m.devices) == 0 {
				m.statusMsg = "Error: No device selected"
			} else if !m.capturing {
				m.capturing = true
				m.captureSpinner = 0
				m.statusMsg = "Capturing..."
				return m, tea.Batch(startCapture(m), tick())
			}
		case "f":
			// Toggle filter
			m.filterFrames = !m.filterFrames
			if m.filterFrames {
				m.statusMsg = "Filter: ON"
			} else {
				m.statusMsg = "Filter: OFF"
			}
		case "d":
			// Jump to duration and open dropdown
			m.activePanel = panelCaptureSettings
			m.cursor = 1 // Duration is cursor 1
			m.selectingDuration = true
			// Set cursor to current duration option
			for i, opt := range m.durationOptions {
				if opt == m.duration {
					m.durationCursor = i
					break
				}
			}
		case "tab":
			// Cycle through panels
			m.activePanel = (m.activePanel + 1) % 5
			m.cursor = 0
		case "shift+tab":
			// Cycle backwards through panels
			m.activePanel = (m.activePanel - 1 + 5) % 5
			m.cursor = 0
		case "1":
			m.activePanel = panelDevices
			m.cursor = 0
		case "2":
			m.activePanel = panelConfiguration
			m.cursor = 0
		case "3":
			m.activePanel = panelCaptureSettings
			m.cursor = 0
		case "4":
			m.activePanel = panelOutput
			m.cursor = 0
		case "5":
			m.activePanel = panelStatus
			m.cursor = 0
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.cursor++
		case "enter":
			return m.handleEnter()
		}
	case tickMsg:
		// Update spinner animation during capture
		if m.capturing {
			m.captureSpinner++
			return m, tick()
		}
	case captureCompleteMsg:
		m.capturing = false
		m.captureErr = msg.err
		if msg.err != nil {
			m.statusMsg = "Capture failed: " + msg.err.Error()
		} else {
			m.statusMsg = "Capture complete: " + m.outputFile
			// Load output data
			m.loadOutputData()
		}
		return m, nil
	}
	return m, nil
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case panelDevices:
		if len(m.devices) > 0 && m.cursor < len(m.devices) {
			m.selectedDevice = m.cursor
			m.statusMsg = "Device selected: " + m.devices[m.selectedDevice].DisplayName
		}
	case panelConfiguration:
		// Toggle protocol or edit pin values
		if m.cursor == 0 {
			// Cycle through protocols
			m.protocol = (m.protocol + 1) % 3
			switch m.protocol {
			case ProtocolSPI:
				m.statusMsg = "Protocol: SPI"
			case ProtocolI2C:
				m.statusMsg = "Protocol: I2C"
			case ProtocolUART:
				m.statusMsg = "Protocol: UART"
			}
		} else {
			// Edit pin configuration
			m.editing = true
			m.editBuffer = m.getCurrentConfigValue()
		}
	case panelCaptureSettings:
		if m.cursor == 0 {
			// Sample Rate dropdown
			m.selectingSampleRate = true
			for i, opt := range m.sampleRateOptions {
				if opt == m.sampleRate {
					m.sampleRateCursor = i
					break
				}
			}
		} else if m.cursor == 1 {
			// Duration dropdown
			m.selectingDuration = true
			for i, opt := range m.durationOptions {
				if opt == m.duration {
					m.durationCursor = i
					break
				}
			}
		} else if m.cursor == 2 {
			// Output file
			m.editing = true
			m.editBuffer = m.outputFile
		} else if m.cursor == 3 {
			// Toggle filter
			m.filterFrames = !m.filterFrames
		} else if m.cursor == 4 {
			// Start capture
			if len(m.devices) == 0 {
				m.statusMsg = "Error: No device selected"
			} else {
				m.capturing = true
				m.captureSpinner = 0
				m.statusMsg = "Capturing..."
				return m, tea.Batch(startCapture(m), tick())
			}
		}
	}
	return m, nil
}

func (m *model) saveEdit() {
	switch m.activePanel {
	case panelConfiguration:
		if m.protocol == ProtocolSPI {
			switch m.cursor - 1 { // cursor 0 is protocol toggle
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
		} else if m.protocol == ProtocolI2C {
			switch m.cursor - 1 {
			case 0:
				m.i2cSDA = m.editBuffer
			case 1:
				m.i2cSCL = m.editBuffer
			case 2:
				m.i2cAddress = m.editBuffer
			}
		} else if m.protocol == ProtocolUART {
			switch m.cursor - 1 {
			case 0:
				m.uartTX = m.editBuffer
			case 1:
				m.uartRX = m.editBuffer
			case 2:
				m.uartBaud = m.editBuffer
			}
		}
	case panelCaptureSettings:
		switch m.cursor {
		case 0:
			// Sample rate - validate it's a number
			if m.editBuffer != "" {
				m.sampleRate = m.editBuffer
			}
		case 1:
			m.duration = m.editBuffer
		case 2:
			m.outputFile = m.editBuffer
		}
	}
}

func (m model) getCurrentConfigValue() string {
	if m.protocol == ProtocolSPI {
		switch m.cursor - 1 {
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
	} else if m.protocol == ProtocolI2C {
		switch m.cursor - 1 {
		case 0:
			return m.i2cSDA
		case 1:
			return m.i2cSCL
		case 2:
			return m.i2cAddress
		}
	} else if m.protocol == ProtocolUART {
		switch m.cursor - 1 {
		case 0:
			return m.uartTX
		case 1:
			return m.uartRX
		case 2:
			return m.uartBaud
		}
	}
	return ""
}

func (m *model) loadOutputData() {
	// Read the output CSV file
	data, err := os.ReadFile(m.outputFile)
	if err != nil {
		m.outputData = []string{"Error reading file: " + err.Error()}
		return
	}
	m.outputData = strings.Split(string(data), "\n")
	if len(m.outputData) > 100 {
		m.outputData = m.outputData[:100] // Limit to first 100 lines
	}
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate panel dimensions (account for outer border)
	leftWidth := 35
	rightWidth := m.width - leftWidth - 6

	// Left panels heights
	devicesHeight := 8
	configHeight := 12
	captureHeight := 10
	leftTotalHeight := devicesHeight + configHeight + captureHeight + 6 // +6 for borders/padding

	// Right panels heights - match left total
	statusHeight := 3 // Single line of content + padding
	outputHeight := leftTotalHeight - statusHeight

	// Render panels
	devicesPanel := m.renderDevicesPanel(leftWidth, devicesHeight)
	configPanel := m.renderConfigPanel(leftWidth, configHeight)
	capturePanel := m.renderCapturePanel(leftWidth, captureHeight)
	outputPanel := m.renderOutputPanel(rightWidth, outputHeight)
	statusPanel := m.renderStatusPanel(rightWidth, statusHeight)

	// Stack left panels vertically
	leftColumn := lipgloss.JoinVertical(lipgloss.Left,
		devicesPanel,
		configPanel,
		capturePanel,
	)

	// Stack right panels vertically
	rightColumn := lipgloss.JoinVertical(lipgloss.Left,
		outputPanel,
		statusPanel,
	)

	// Join columns horizontally
	mainView := lipgloss.JoinHorizontal(lipgloss.Top,
		leftColumn,
		rightColumn,
	)

	// Add border around entire view
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	borderedView := borderStyle.Render(mainView)

	// Add status bar at bottom
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, borderedView, statusBar)
}


func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

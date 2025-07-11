package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type pingMsg time.Duration

type downloadMsg struct {
	bytes int64
	dur   time.Duration
}

type uploadMsg struct {
	bytes int64
	dur   time.Duration
}

type errorMsg struct {
	err error
}

type model struct {
	phase         string
	latency       time.Duration
	downloadSpeed float64
	uploadSpeed   float64
	progress      progress.Model
}

func initialModel() model {
	prog := progress.New(progress.WithDefaultGradient())
	return model{
		phase:    "ping",
		progress: prog,
	}
}

func (m model) Init() tea.Cmd {
	return pingTestCmd()
}

// Update handles incoming messages
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pingMsg:
		m.latency = time.Duration(msg)
		m.phase = "download"
		return m, downloadTestCmd()

	case downloadMsg:
		m.downloadSpeed = float64(msg.bytes) / msg.dur.Seconds() / (1024 * 1024)
		m.phase = "upload"
		return m, uploadTestCmd()

	case uploadMsg:
		m.uploadSpeed = float64(msg.bytes) / msg.dur.Seconds() / (1024 * 1024)
		m.phase = "done"
		return m, nil

	case errorMsg:
		// On error, show and exit
		m.phase = "error"
		m.progress.Width = 0
		return m, nil

	default:
		return m, nil
	}
}

// View renders the UI
func (m model) View() string {
	switch m.phase {
	case "ping":
		return fmt.Sprintf("Ping: measuring... %s\n", m.phase)
	case "download":
		return fmt.Sprintf("Download: measuring... %s\n%s", m.phase, m.progress.View())
	case "upload":
		return fmt.Sprintf("Upload: measuring... %s\n%s", m.phase, m.progress.View())
	case "done":
		return fmt.Sprintf("Results:\n  Ping: %v\n  Download: %.2f MB/s\n  Upload:   %.2f MB/s\n", m.latency, m.downloadSpeed, m.uploadSpeed)
	case "error":
		return "An error occurred during the test."
	default:
		return ""
	}
}

// pingTestCmd runs the ping test
func pingTestCmd() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		// TODO: perform HTTP GET /ping
		// dummy sleep for placeholder
		time.Sleep(50 * time.Millisecond)
		return pingMsg(time.Since(start))
	}
}

// downloadTestCmd runs the download test
func downloadTestCmd() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		// TODO: perform HTTP GET /download?size=... and track bytes
		time.Sleep(200 * time.Millisecond)
		bytes := int64(10 * 1024 * 1024) // placeholder 10MiB
		dur := time.Since(start)
		return downloadMsg{bytes: bytes, dur: dur}
	}
}

// uploadTestCmd runs the upload test
func uploadTestCmd() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		// TODO: perform HTTP POST /upload
		time.Sleep(150 * time.Millisecond)
		bytes := int64(10 * 1024 * 1024) // placeholder
		dur := time.Since(start)
		return uploadMsg{bytes: bytes, dur: dur}
	}
}

func main() {
	if len(os.Args) > 2 {
		command := os.Args[1]
		var m model
		switch command {
		case "help", "--help", "-h":
			m = initialModel()
			m.phase = "help"
		case "version", "--version", "-v":
			m = initialModel()
			m.phase = "version"
		default:
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Use 'speedtest help' for usage information")
			os.Exit(1)
		}

		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
	}

	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	serverBase = "http://localhost:8080" // speedtest server base URL
	testSize   = 100 * 1024 * 1024       // 100 MB payload
	streams    = 4                       // concurrent streams for down/up
)

var httpClient = &http.Client{Transport: &http.Transport{MaxIdleConns: streams, DisableCompression: true}}

type pingMsg time.Duration

type downloadMsg struct {
	bytes int64
	dur   time.Duration
}

type uploadMsg struct {
	bytes int64
	dur   time.Duration
}

type progressMsg struct {
	phase string
	done  int64
	total int64
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
		return m, tea.Quit

	case progressMsg:
		if msg.total > 0 {
			percent := float64(msg.done) / float64(msg.total)
			m.progress.SetPercent(percent)
		}
		return m, nil

	case errorMsg:
		m.phase = "error"
		m.progress.Width = 0
		return m, tea.Quit

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

// helper to periodically send progress updates
func makeProgressCmd(counter *int64, total int64, phase string) tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		done := atomic.LoadInt64(counter)
		if done >= total {
			return nil
		}
		return progressMsg{phase: phase, done: done, total: total}
	})
}

// pingTestCmd runs the ping test by performing an actual GET /ping request
func pingTestCmd() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		resp, err := httpClient.Get(serverBase + "/ping")
		if err != nil {
			return errorMsg{err}
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return pingMsg(time.Since(start))
	}
}

// downloadTestCmd runs the download test by downloading a payload of size `testSize`
func downloadTestCmd() tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("%s/download?size=%d", serverBase, testSize)
		start := time.Now()
		resp, err := httpClient.Get(url)
		if err != nil {
			return errorMsg{err}
		}
		n, err := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if err != nil {
			return errorMsg{err}
		}
		dur := time.Since(start)
		return downloadMsg{bytes: n, dur: dur}
	}
}

// uploadTestCmd runs the upload test by POST-ing a random payload of size `testSize`
func uploadTestCmd() tea.Cmd {
	return func() tea.Msg {
		var totalBytes int64
		var wg sync.WaitGroup
		wg.Add(streams)
		segSize := testSize / streams
		counter := int64(0)

		start := time.Now()

		for i := 0; i < streams; i++ {
			go func() {
				defer wg.Done()
				payload := make([]byte, segSize) // zeros, no randomisation
				req, err := http.NewRequest("POST", serverBase+"/upload", bytes.NewReader(payload))
				if err != nil {
					return
				}
				req.ContentLength = int64(segSize)
				resp, err := httpClient.Do(req)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					atomic.AddInt64(&totalBytes, int64(segSize))
					atomic.AddInt64(&counter, int64(segSize))
				}
			}()
		}

		progressCmd := makeProgressCmd(&counter, int64(testSize), "upload")

		wg.Wait()
		dur := time.Since(start)

		return tea.Batch(
			func() tea.Msg { return uploadMsg{bytes: totalBytes, dur: dur} },
			progressCmd,
		)()
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

package systray

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/getlantern/systray"
)

//go:embed icon.png
var embeddedIcon []byte

// MenuBar represents the system tray menu bar
type MenuBar struct {
	port       int
	onQuit     func()
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewMenuBar creates a new menu bar instance
func NewMenuBar(port int, onQuit func()) *MenuBar {
	ctx, cancel := context.WithCancel(context.Background())
	return &MenuBar{
		port:       port,
		onQuit:     onQuit,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// Run starts the system tray menu bar
func (m *MenuBar) Run() {
	systray.Run(m.onReady, m.onExit)
}

// Stop stops the system tray
func (m *MenuBar) Stop() {
	m.cancelFunc()
	systray.Quit()
}

func (m *MenuBar) onReady() {
	// Set icon only (no title text)
	systray.SetTitle("")
	systray.SetTooltip("Health Checker - Monitoring Services")

	// Use a simple icon
	iconData := getIcon()
	if iconData != nil {
		systray.SetIcon(iconData)
	}

	// Create menu items
	mTitle := systray.AddMenuItem("Simple Healthchecker", "")
	mTitle.Disable()
	systray.AddSeparator()
	mOpen := systray.AddMenuItem("Open Web UI", "Open the web interface in browser")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-mOpen.ClickedCh:
				m.openWebUI()
			case <-mQuit.ClickedCh:
				log.Println("Quit requested from menu bar")
				if m.onQuit != nil {
					m.onQuit()
				}
				systray.Quit()
				return
			}
		}
	}()
}

func (m *MenuBar) onExit() {
	// Cleanup when systray exits
}

func (m *MenuBar) openWebUI() {
	url := fmt.Sprintf("http://localhost:%d", m.port)
	log.Printf("Opening web UI: %s", url)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		log.Printf("Unsupported platform for opening browser: %s", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open web UI: %v", err)
	}
}

// getIcon returns a health status icon for the menu bar
// Uses the embedded icon.png file from the systray package directory
func getIcon() []byte {
	// Try to load icon from runtime file first (for easy customization)
	iconPaths := []string{
		"icon.png",
		"menubar-icon.png",
		filepath.Join("assets", "icon.png"),
	}

	for _, path := range iconPaths {
		if iconData, err := os.ReadFile(path); err == nil {
			log.Printf("Loaded menu bar icon from runtime file: %s", path)
			return iconData
		}
	}

	// Use embedded icon
	if len(embeddedIcon) > 0 {
		log.Println("Using embedded menu bar icon")
		return embeddedIcon
	}

	log.Println("Warning: No icon available")
	return nil
}

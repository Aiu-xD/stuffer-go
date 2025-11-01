package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"universal-checker/internal/checker"
	"universal-checker/internal/config"
	"universal-checker/pkg/types"
)

type GUI struct {
	app    fyne.App
	window fyne.Window

	// File paths
	configPaths []string
	comboPath   string
	proxyPath   string

	// GUI components
	configList      *widget.List
	comboEntry      *widget.Entry
	proxyEntry      *widget.Entry
	selectAllCheck  *widget.Check
	autoScrapeCheck *widget.Check
	workersEntry    *widget.Entry
	timeoutEntry    *widget.Entry

	// Status components
	statusLabel *widget.Label
	progressBar *widget.ProgressBar
	statsLabel  *widget.RichText
	logArea     *widget.RichText

	// Control buttons
	startBtn *widget.Button
	stopBtn  *widget.Button
	clearBtn *widget.Button

	// Checker instance and state
	checker   *checker.Checker
	isRunning bool
	mutex     sync.RWMutex

	// Configuration data
	configs         []types.Config
	selectedConfigs map[int]bool

	// Resource management
	statsUpdateTicker *time.Ticker
	statsUpdateDone   chan bool
	logBuffer         []string
	maxLogLines       int

	// UI update channel for thread safety
	uiUpdateChan chan func()
}

func main() {
	gui := NewGUI()
	gui.Run()
}

func NewGUI() *GUI {
	myApp := app.New()
	myApp.SetIcon(nil) // You can set a custom icon here

	window := myApp.NewWindow("Universal Checker - GUI")
	window.Resize(fyne.NewSize(800, 600))

	gui := &GUI{
		app:             myApp,
		window:          window,
		configPaths:     make([]string, 0),
		selectedConfigs: make(map[int]bool),
		isRunning:       false,
		statsUpdateDone: make(chan bool, 1),
		logBuffer:       make([]string, 0),
		maxLogLines:     1000,
		uiUpdateChan:    make(chan func(), 100),
	}

	gui.setupUI()
	gui.setupResourceManagement()
	gui.startUIUpdateHandler()
	return gui
}

func (g *GUI) setupUI() {
	// Main container
	content := container.NewVBox()

	// Header
	title := widget.NewLabelWithStyle("Universal Checker", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// File selection section
	fileSection := g.createFileSection()

	// Configuration selection section
	configSection := g.createConfigSection()

	// Settings section
	settingsSection := g.createSettingsSection()

	// Control buttons
	controlSection := g.createControlSection()

	// Status and progress section
	statusSection := g.createStatusSection()

	// Add all sections to main container
	content.Add(title)
	content.Add(widget.NewSeparator())
	content.Add(fileSection)
	content.Add(widget.NewSeparator())
	content.Add(configSection)
	content.Add(widget.NewSeparator())
	content.Add(settingsSection)
	content.Add(widget.NewSeparator())
	content.Add(controlSection)
	content.Add(widget.NewSeparator())
	content.Add(statusSection)

	// Set up drag and drop
	g.setupDragAndDrop()

	g.window.SetContent(container.NewScroll(content))
}

// setupResourceManagement configures cleanup handlers and resource management
func (g *GUI) setupResourceManagement() {
	g.window.SetCloseIntercept(func() {
		g.cleanup()
		g.window.Close()
	})
}

// startUIUpdateHandler starts the goroutine that handles thread-safe UI updates
func (g *GUI) startUIUpdateHandler() {
	go func() {
		for updateFunc := range g.uiUpdateChan {
			updateFunc()
		}
	}()
}

// scheduleUIUpdate safely schedules a UI update on the main thread
func (g *GUI) scheduleUIUpdate(updateFunc func()) {
	select {
	case g.uiUpdateChan <- updateFunc:
		// Update scheduled successfully
	default:
		// Channel full, skip this update to prevent blocking
	}
}

// cleanup properly releases all resources
func (g *GUI) cleanup() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.isRunning {
		g.stopCheckerInternal()
	}

	if g.statsUpdateTicker != nil {
		g.statsUpdateTicker.Stop()
	}

	select {
	case g.statsUpdateDone <- true:
	default:
	}

	close(g.uiUpdateChan)
}

func (g *GUI) createFileSection() *fyne.Container {
	section := container.NewVBox()

	// Combo file selection
	comboLabel := widget.NewLabel("Combo File:")
	g.comboEntry = widget.NewEntry()
	g.comboEntry.SetPlaceHolder("Select or drag combo file (.txt)")
	comboBtn := widget.NewButton("Browse", func() {
		g.selectComboFile()
	})
	comboRow := container.NewBorder(nil, nil, comboLabel, comboBtn, g.comboEntry)

	// Proxy file selection (optional)
	proxyLabel := widget.NewLabel("Proxy File:")
	g.proxyEntry = widget.NewEntry()
	g.proxyEntry.SetPlaceHolder("Optional: Select or drag proxy file (.txt)")
	proxyBtn := widget.NewButton("Browse", func() {
		g.selectProxyFile()
	})
	proxyRow := container.NewBorder(nil, nil, proxyLabel, proxyBtn, g.proxyEntry)

	// Auto-scrape option
	g.autoScrapeCheck = widget.NewCheck("Auto-scrape proxies", nil)
	g.autoScrapeCheck.SetChecked(true)

	section.Add(comboRow)
	section.Add(proxyRow)
	section.Add(g.autoScrapeCheck)

	return section
}

func (g *GUI) createConfigSection() *fyne.Container {
	section := container.NewVBox()

	// Config files header
	configHeader := container.NewHBox(
		widget.NewLabelWithStyle("Configuration Files", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewButton("Add Config", func() {
			g.selectConfigFiles()
		}),
		widget.NewButton("Clear All", func() {
			g.clearConfigs()
		}),
	)

	// Select all checkbox
	g.selectAllCheck = widget.NewCheck("Select All Configs", func(checked bool) {
		g.toggleAllConfigs(checked)
	})

	// Config list
	g.configList = widget.NewList(
		func() int {
			return len(g.configs)
		},
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			label := widget.NewLabel("Config Name")
			return container.NewHBox(check, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			container := obj.(*fyne.Container)
			check := container.Objects[0].(*widget.Check)
			label := container.Objects[1].(*widget.Label)

			if id < len(g.configs) {
				config := g.configs[id]
				label.SetText(fmt.Sprintf("%s (%s)", config.Name, strings.ToUpper(string(config.Type))))
				check.SetChecked(g.selectedConfigs[id])
				check.OnChanged = func(checked bool) {
					g.selectedConfigs[id] = checked
					g.updateSelectAllCheck()
				}
			}
		},
	)
	g.configList.Resize(fyne.NewSize(400, 150))

	section.Add(configHeader)
	section.Add(g.selectAllCheck)
	section.Add(g.configList)

	return section
}

func (g *GUI) createSettingsSection() *fyne.Container {
	section := container.NewVBox()

	settingsLabel := widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Workers setting
	workersLabel := widget.NewLabel("Workers:")
	g.workersEntry = widget.NewEntry()
	g.workersEntry.SetText("100")
	workersRow := container.NewBorder(nil, nil, workersLabel, nil, g.workersEntry)

	// Timeout setting
	timeoutLabel := widget.NewLabel("Timeout (ms):")
	g.timeoutEntry = widget.NewEntry()
	g.timeoutEntry.SetText("30000")
	timeoutRow := container.NewBorder(nil, nil, timeoutLabel, nil, g.timeoutEntry)

	settingsGrid := container.NewGridWithColumns(2, workersRow, timeoutRow)

	section.Add(settingsLabel)
	section.Add(settingsGrid)

	return section
}

func (g *GUI) createControlSection() *fyne.Container {
	g.startBtn = widget.NewButton("Start Checking", func() {
		g.startChecking()
	})
	g.startBtn.Importance = widget.HighImportance

	g.stopBtn = widget.NewButton("Stop", func() {
		g.stopChecking()
	})
	g.stopBtn.Disable()

	g.clearBtn = widget.NewButton("Clear Results", func() {
		g.clearResults()
	})

	return container.NewHBox(g.startBtn, g.stopBtn, g.clearBtn)
}

func (g *GUI) createStatusSection() *fyne.Container {
	section := container.NewVBox()

	// Status label
	g.statusLabel = widget.NewLabelWithStyle("Ready", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Progress bar
	g.progressBar = widget.NewProgressBar()
	g.progressBar.Hide()

	// Statistics
	g.statsLabel = widget.NewRichTextFromMarkdown("")
	g.statsLabel.Resize(fyne.NewSize(400, 100))

	// Log area
	logLabel := widget.NewLabelWithStyle("Log Output", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	g.logArea = widget.NewRichText()
	g.logArea.Resize(fyne.NewSize(400, 150))
	logScroll := container.NewScroll(g.logArea)
	logScroll.SetMinSize(fyne.NewSize(400, 150))

	section.Add(g.statusLabel)
	section.Add(g.progressBar)
	section.Add(g.statsLabel)
	section.Add(logLabel)
	section.Add(logScroll)

	return section
}

func (g *GUI) setupDragAndDrop() {
	// Note: Fyne doesn't have built-in drag and drop for files yet
	// This is a placeholder for when that feature is available
	// For now, users will use the browse buttons
}

// validateNumericInput validates and sanitizes numeric input with bounds checking
func (g *GUI) validateNumericInput(input string, defaultValue, min, max int) int {
	if strings.TrimSpace(input) == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return defaultValue
	}

	if value < min {
		return min
	}
	if value > max {
		return max
	}

	return value
}

// validateInputs validates all user inputs before starting the checker
func (g *GUI) validateInputs() error {
	if g.comboPath == "" {
		return fmt.Errorf("please select a combo file")
	}

	if !g.fileExists(g.comboPath) {
		return fmt.Errorf("combo file does not exist: %s", g.comboPath)
	}

	if g.proxyPath != "" && !g.fileExists(g.proxyPath) {
		return fmt.Errorf("proxy file does not exist: %s", g.proxyPath)
	}

	selectedConfigs := g.getSelectedConfigs()
	if len(selectedConfigs) == 0 {
		return fmt.Errorf("please select at least one configuration")
	}

	return nil
}

// fileExists checks if a file exists and is readable
func (g *GUI) fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func (g *GUI) selectComboFile() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		g.comboPath = reader.URI().Path()
		g.comboEntry.SetText(filepath.Base(g.comboPath))
		g.logMessage(fmt.Sprintf("Loaded combo file: %s", filepath.Base(g.comboPath)))
	}, g.window)
}

func (g *GUI) selectProxyFile() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		g.proxyPath = reader.URI().Path()
		g.proxyEntry.SetText(filepath.Base(g.proxyPath))
		g.logMessage(fmt.Sprintf("Loaded proxy file: %s", filepath.Base(g.proxyPath)))
	}, g.window)
}

func (g *GUI) selectConfigFiles() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		configPath := reader.URI().Path()
		ext := strings.ToLower(filepath.Ext(configPath))

		if ext != ".opk" && ext != ".svb" && ext != ".loli" {
			dialog.ShowError(fmt.Errorf("unsupported config format: %s", ext), g.window)
			return
		}

		// Parse the config
		parser := config.NewParser()
		cfg, err := parser.ParseConfig(configPath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to parse config: %v", err), g.window)
			return
		}

		g.configs = append(g.configs, *cfg)
		g.configPaths = append(g.configPaths, configPath)
		g.selectedConfigs[len(g.configs)-1] = true

		g.configList.Refresh()
		g.updateSelectAllCheck()
		g.logMessage(fmt.Sprintf("Loaded config: %s (%s)", cfg.Name, strings.ToUpper(string(cfg.Type))))
	}, g.window)
}

func (g *GUI) clearConfigs() {
	g.configs = make([]types.Config, 0)
	g.configPaths = make([]string, 0)
	g.selectedConfigs = make(map[int]bool)
	g.configList.Refresh()
	g.selectAllCheck.SetChecked(false)
	g.logMessage("Cleared all configurations")
}

func (g *GUI) toggleAllConfigs(checked bool) {
	for i := range g.configs {
		g.selectedConfigs[i] = checked
	}
	g.configList.Refresh()
}

func (g *GUI) updateSelectAllCheck() {
	allSelected := true
	for i := range g.configs {
		if !g.selectedConfigs[i] {
			allSelected = false
			break
		}
	}
	g.selectAllCheck.SetChecked(allSelected && len(g.configs) > 0)
}

// startChecking validates inputs and starts the checking process
func (g *GUI) startChecking() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.isRunning {
		return
	}

	// Validate all inputs
	if err := g.validateInputs(); err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	// Parse and validate settings with proper bounds
	workers := g.validateNumericInput(g.workersEntry.Text, 100, 1, 1000)
	timeout := g.validateNumericInput(g.timeoutEntry.Text, 30000, 1000, 300000)

	// Update UI to show validated values
	g.workersEntry.SetText(fmt.Sprintf("%d", workers))
	g.timeoutEntry.SetText(fmt.Sprintf("%d", timeout))

	// Create checker configuration
	checkerConfig := &types.CheckerConfig{
		MaxWorkers:        workers,
		ProxyTimeout:      5000,
		RequestTimeout:    timeout,
		RetryCount:        3,
		ProxyRotation:     true,
		AutoScrapeProxies: g.autoScrapeCheck.Checked,
		SaveValidOnly:     true,
		OutputFormat:      "txt",
		OutputDirectory:   "results",
	}

	// Create checker instance
	g.checker = checker.NewChecker(checkerConfig)

	// Set only selected configs
	g.checker.Configs = g.getSelectedConfigs()

	// Update UI state
	g.setRunningState(true)

	// Start checking in goroutine
	go g.runChecker()
}

// stopChecking stops the checking process
func (g *GUI) stopChecking() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.isRunning {
		return
	}

	g.stopCheckerInternal()
	g.setRunningState(false)
	g.logMessage("Checking stopped by user")
}

// stopCheckerInternal stops the checker without UI updates (internal use)
func (g *GUI) stopCheckerInternal() {
	if g.checker != nil {
		g.checker.Stop()
	}

	if g.statsUpdateTicker != nil {
		g.statsUpdateTicker.Stop()
		g.statsUpdateTicker = nil
	}

	select {
	case g.statsUpdateDone <- true:
	default:
	}
}

// setRunningState centralizes UI state management for running/stopped states
func (g *GUI) setRunningState(running bool) {
	g.isRunning = running

	g.scheduleUIUpdate(func() {
		if running {
			g.startBtn.Disable()
			g.stopBtn.Enable()
			g.statusLabel.SetText("Starting checker...")
			g.progressBar.Show()
		} else {
			g.startBtn.Enable()
			g.stopBtn.Disable()
			g.statusLabel.SetText("Stopped")
			g.progressBar.Hide()
		}
	})
}

func (g *GUI) runChecker() {
	// Load combos
	g.logMessage("Loading combos...")
	if err := g.checker.LoadCombos(g.comboPath); err != nil {
		g.logMessage(fmt.Sprintf("Error loading combos: %v", err))
		g.stopChecking()
		return
	}
	g.logMessage(fmt.Sprintf("Loaded %d combos", len(g.checker.Combos)))

	// Load proxies
	if g.proxyPath != "" {
		g.logMessage("Loading proxies...")
		if err := g.checker.LoadProxies(g.proxyPath); err != nil {
			g.logMessage(fmt.Sprintf("Warning: Failed to load proxies: %v", err))
		} else {
			g.logMessage(fmt.Sprintf("Loaded %d proxies", len(g.checker.Proxies)))
		}
	} else {
		g.logMessage("Auto-scraping proxies...")
		if err := g.checker.LoadProxies(""); err != nil {
			g.logMessage(fmt.Sprintf("Warning: Failed to scrape proxies: %v", err))
		} else {
			g.logMessage(fmt.Sprintf("Scraped %d working proxies", len(g.checker.Proxies)))
		}
	}

	// Start checking
	g.logMessage("Starting checker...")
	if err := g.checker.Start(); err != nil {
		g.logMessage(fmt.Sprintf("Error starting checker: %v", err))
		g.stopChecking()
		return
	}

	// Update status periodically
	go g.updateStats()
}

// updateStats safely updates statistics with thread safety and division by zero protection
func (g *GUI) updateStats() {
	g.statsUpdateTicker = time.NewTicker(2 * time.Second)
	defer func() {
		if g.statsUpdateTicker != nil {
			g.statsUpdateTicker.Stop()
		}
	}()

	for {
		select {
		case <-g.statsUpdateTicker.C:
			g.mutex.RLock()
			isRunning := g.isRunning
			checker := g.checker
			g.mutex.RUnlock()

			if !isRunning || checker == nil {
				return
			}

			stats := checker.GetStats()
			g.updateUIWithStats(stats)

		case <-g.statsUpdateDone:
			return
		}
	}
}

// updateUIWithStats safely updates UI elements with statistics
func (g *GUI) updateUIWithStats(stats types.CheckerStats) {
	g.scheduleUIUpdate(func() {
		// Update status
		g.statusLabel.SetText(fmt.Sprintf("Running - CPM: %.1f", stats.CurrentCPM))

		// Calculate progress with division by zero protection
		totalTasks := g.calculateTotalTasks(stats)
		processed := stats.ValidCombos + stats.InvalidCombos + stats.ErrorCombos
		progressPercent := g.calculateProgressPercent(processed, totalTasks)

		if totalTasks > 0 {
			g.progressBar.SetValue(float64(processed) / float64(totalTasks))
		}

		// Update stats display
		statsText := g.formatStatsText(stats, progressPercent)
		g.statsLabel.ParseMarkdown(statsText)
	})
}

// calculateTotalTasks safely calculates total tasks with zero protection
func (g *GUI) calculateTotalTasks(stats types.CheckerStats) int {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if g.checker == nil || len(g.checker.Configs) == 0 {
		return 1 // Prevent division by zero
	}

	totalTasks := stats.TotalCombos * len(g.checker.Configs)
	if totalTasks <= 0 {
		return 1 // Prevent division by zero
	}

	return totalTasks
}

// calculateProgressPercent safely calculates progress percentage
func (g *GUI) calculateProgressPercent(processed, totalTasks int) float64 {
	if totalTasks <= 0 {
		return 0.0
	}
	return float64(processed) / float64(totalTasks) * 100
}

// formatStatsText creates formatted statistics text
func (g *GUI) formatStatsText(stats types.CheckerStats, progressPercent float64) string {
	return fmt.Sprintf(`**Statistics**

⏱️ **Elapsed Time:** %s
📊 **Total Combos:** %d
✅ **Valid:** %d
❌ **Invalid:** %d
⚠️ **Errors:** %d
🚀 **Current CPM:** %.1f
👥 **Active Workers:** %d
🌐 **Working Proxies:** %d/%d
📈 **Progress:** %.1f%%`,
		g.formatDuration(stats.ElapsedTime),
		stats.TotalCombos,
		stats.ValidCombos,
		stats.InvalidCombos,
		stats.ErrorCombos,
		stats.CurrentCPM,
		stats.ActiveWorkers,
		stats.WorkingProxies,
		stats.TotalProxies,
		progressPercent)
}

func (g *GUI) getSelectedConfigs() []types.Config {
	var selected []types.Config
	for i, config := range g.configs {
		if g.selectedConfigs[i] {
			selected = append(selected, config)
		}
	}
	return selected
}

// clearResults clears all results and resets UI state
func (g *GUI) clearResults() {
	g.mutex.Lock()
	g.logBuffer = make([]string, 0)
	g.mutex.Unlock()

	g.scheduleUIUpdate(func() {
		g.logArea.ParseMarkdown("")
		g.statsLabel.ParseMarkdown("")
		g.progressBar.SetValue(0)
		g.statusLabel.SetText("Ready")
	})
}

// logMessage safely adds a message to the log with size management
func (g *GUI) logMessage(message string) {
	timestamp := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, message)

	// Add to buffer with size management
	g.addToLogBuffer(logEntry)

	// Update UI safely
	g.scheduleUIUpdate(func() {
		g.updateLogDisplay()
	})
}

// addToLogBuffer manages log buffer size and adds new entries
func (g *GUI) addToLogBuffer(entry string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.logBuffer = append(g.logBuffer, entry)

	// Trim buffer if it exceeds max size
	if len(g.logBuffer) > g.maxLogLines {
		// Keep only the last maxLogLines entries
		g.logBuffer = g.logBuffer[len(g.logBuffer)-g.maxLogLines:]
	}
}

// updateLogDisplay updates the log area display
func (g *GUI) updateLogDisplay() {
	g.mutex.RLock()
	logBuffer := make([]string, len(g.logBuffer))
	copy(logBuffer, g.logBuffer)
	g.mutex.RUnlock()

	// Build log text efficiently
	var builder strings.Builder
	for _, entry := range logBuffer {
		builder.WriteString(entry)
		builder.WriteString("\n")
	}

	g.logArea.ParseMarkdown(builder.String())
}

func (g *GUI) formatDuration(seconds int) string {
	duration := time.Duration(seconds) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	} else {
		return fmt.Sprintf("%ds", secs)
	}
}

func (g *GUI) Run() {
	g.window.ShowAndRun()
}

package checker

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"universal-checker/internal/config"
	"universal-checker/internal/logger"
	"universal-checker/internal/proxy"
	"universal-checker/pkg/httpclient"
	"universal-checker/pkg/types"
	"universal-checker/pkg/utils"
)

// ============================================================================
// TYPES AND STRUCTURES
// ============================================================================

// Checker represents the main checker engine with thread-safe concurrent operations
//
// CONCURRENCY CONTRACT:
// - Goroutine Coordination: All goroutines (workers, result processor, task generator)
//   are tracked via sync.WaitGroup ensuring graceful shutdown
// - Context Cancellation: All goroutines respect ctx.Done() for immediate cancellation
// - Statistics: Protected by statsMutex (RWMutex) - concurrent reads, exclusive writes
// - Proxy Rotation: Protected by proxyMutex (Mutex) - exclusive access to proxyIndex
// - Channel Safety: taskChan and resultChan use buffered channels with context-aware operations
//
// THREAD-SAFETY GUARANTEES:
// - Stats.* fields: MUST acquire statsMutex before read/write
// - proxyIndex: MUST acquire proxyMutex before read/write
// - Channels: Thread-safe by Go runtime, context-aware sends prevent deadlocks
// - All other fields: Read-only after initialization (safe for concurrent access)
type Checker struct {
	Config      *types.CheckerConfig
	Stats       *types.CheckerStats
	Proxies     []types.Proxy
	Configs     []types.Config
	Combos      []types.Combo
	
	// Channels for communication (buffered, thread-safe)
	taskChan   chan types.WorkerTask
	resultChan chan types.WorkerResult
	
	// Worker management and coordination
	ctx        context.Context     // Cancellation signal for all goroutines
	cancel     context.CancelFunc  // Trigger for graceful shutdown
	wg         sync.WaitGroup      // Tracks all spawned goroutines (workers + auxiliaries)
	
	// Statistics tracking (protected by statsMutex)
	statsMutex sync.RWMutex        // RWMutex: concurrent reads, exclusive writes
	
	// Proxy rotation (protected by proxyMutex)
	proxyIndex int                 // Current proxy index (protected by proxyMutex)
	proxyMutex sync.Mutex          // Mutex: exclusive access to proxyIndex
	
	// Result exporter
	exporter   *ResultExporter
	
// Enhanced parsing and variable systems
	workflowEngine *WorkflowEngine
	varManipulator *VariableManipulator

	// Advanced proxy management systems
	proxyManager    *AdvancedProxyManager
	healthMonitor   *ProxyHealthMonitor
	
	// Logging and reporting
	logger          *logger.StructuredLogger
}

// ============================================================================
// CONSTRUCTOR AND INITIALIZATION
// ============================================================================

// initializeContext creates a cancellable context for checker lifecycle management
func initializeContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// initializeWorkflowSystem creates and configures the workflow engine and variable manipulator
func initializeWorkflowSystem() (*WorkflowEngine, *VariableManipulator) {
	workflowEngine := NewWorkflowEngine()
	varManipulator := NewVariableManipulator(workflowEngine.variables)
	return workflowEngine, varManipulator
}

// initializeProxySystem creates and configures the proxy management and health monitoring systems
func initializeProxySystem() (*AdvancedProxyManager, *ProxyHealthMonitor) {
	proxyManager := NewAdvancedProxyManager(StrategyBestScore)
	healthMonitor := NewProxyHealthMonitor(proxyManager)
	return proxyManager, healthMonitor
}

// initializeLogger creates a structured logger with fallback to stdout on file error
func initializeLogger() (*logger.StructuredLogger, error) {
	loggerConfig := logger.LoggerConfig{
		Level:      logger.INFO,
		JSONFormat: true,
		OutputFile: "logs/checker.log",
		BufferSize: 1000,
		Component:  "checker",
	}
	
	structuredLogger, err := logger.NewStructuredLogger(loggerConfig)
	if err != nil {
		// Fall back to stdout if file logging fails
		loggerConfig.OutputFile = ""
		structuredLogger, fallbackErr := logger.NewStructuredLogger(loggerConfig)
		if fallbackErr != nil {
			return nil, fmt.Errorf("failed to initialize logger (file: %v, stdout: %v)", err, fallbackErr)
		}
		return structuredLogger, nil
	}
	
	return structuredLogger, nil
}

// initializeChannels creates task and result channels based on worker configuration
func initializeChannels(config *types.CheckerConfig) (chan types.WorkerTask, chan types.WorkerResult) {
	channelSize := config.MaxWorkers * 2
	taskChan := make(chan types.WorkerTask, channelSize)
	resultChan := make(chan types.WorkerResult, channelSize)
	return taskChan, resultChan
}

// NewChecker creates a new checker instance by orchestrating subsystem initializers
func NewChecker(config *types.CheckerConfig) *Checker {
	// Initialize context for lifecycle management
	ctx, cancel := initializeContext()
	
	// Initialize workflow processing subsystem
	workflowEngine, varManipulator := initializeWorkflowSystem()
	
	// Initialize proxy management subsystem
	proxyManager, healthMonitor := initializeProxySystem()
	
	// Initialize logging subsystem with error handling
	structuredLogger, err := initializeLogger()
	if err != nil {
		// Logger initialization failed completely (both file and stdout)
		// This is extremely rare but we handle it gracefully
		log.Printf("[ERROR] Failed to initialize structured logger: %v - checker will have limited logging", err)
		// structuredLogger will be nil, causing panic on first use
		// In practice, stdout logging should never fail, so this is a critical system error
		panic(fmt.Sprintf("critical: unable to initialize any logging mechanism: %v", err))
	}
	
	// Initialize communication channels
	taskChan, resultChan := initializeChannels(config)
	
	// Assemble the checker with all initialized subsystems
	return &Checker{
		Config:         config,
		Stats:          &types.CheckerStats{},
		Proxies:        make([]types.Proxy, 0),
		Configs:        make([]types.Config, 0),
		Combos:         make([]types.Combo, 0),
		taskChan:       taskChan,
		resultChan:     resultChan,
		ctx:            ctx,
		cancel:         cancel,
		exporter:       NewResultExporter(config.OutputDirectory, config.OutputFormat),
		workflowEngine: workflowEngine,
		varManipulator: varManipulator,
		proxyManager:   proxyManager,
		healthMonitor:  healthMonitor,
		logger:         structuredLogger,
	}
}

// LoadConfigs loads configuration files
func (c *Checker) LoadConfigs(configPaths []string) error {
	parser := config.NewParser()
	
	for _, configPath := range configPaths {
		cfg, err := parser.ParseConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to parse config %s: %v", configPath, err)
		}
		c.Configs = append(c.Configs, *cfg)
	}
	
	return nil
}

// LoadCombos loads combos from a file
func (c *Checker) LoadCombos(comboPath string) error {
	file, err := os.Open(comboPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		combo := c.parseCombo(line)
		if combo != nil {
			c.Combos = append(c.Combos, *combo)
		}
	}

	c.Stats.TotalCombos = len(c.Combos)
	return scanner.Err()
}

// LoadProxies loads proxies from file or auto-scrapes them
func (c *Checker) LoadProxies(proxyPath string) error {
	if c.Config.AutoScrapeProxies {
		scraper := proxy.NewScraper(c.Config, c.logger)
		proxies, err := scraper.ScrapeAndValidate()
		if err != nil {
			return err
		}
		// Add scraped proxies to the advanced proxy manager
		for _, proxy := range proxies {
			if err := c.proxyManager.AddProxy(proxy); err != nil {
				log.Printf("[WARN] Failed to add scraped proxy %s:%d: %v", proxy.Host, proxy.Port, err)
			}
		}
		c.Proxies = proxies
	} else if proxyPath != "" {
		file, err := os.Open(proxyPath)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			proxy := c.parseProxy(line)
			if proxy != nil {
				// Add to advanced proxy manager
				if err := c.proxyManager.AddProxy(*proxy); err != nil {
					log.Printf("[WARN] Failed to add proxy %s:%d: %v", proxy.Host, proxy.Port, err)
				} else {
					c.Proxies = append(c.Proxies, *proxy)
				}
			}
		}
	}

	c.Stats.TotalProxies = len(c.Proxies)
	return nil
}

// ============================================================================
// CORE CHECKER OPERATIONS
// ============================================================================

// Start begins the checking process with coordinated subsystem initialization
func (c *Checker) Start() error {
	c.Stats.StartTime = time.Now()
	
	c.logger.Info("Starting checker", map[string]interface{}{
		"max_workers": c.Config.MaxWorkers,
		"total_combos": len(c.Combos),
		"total_configs": len(c.Configs),
		"total_proxies": len(c.Proxies),
	})
	
	// Start health monitor for proxy management
	c.healthMonitor.Start()
	
	// Start worker subsystems with lifecycle tracking
	c.startWorkerPool()
	c.startResultProcessor()
	c.startTaskGenerator()

	c.logger.Info("Checker started successfully")
	return nil
}

// Stop stops the checking process with coordinated shutdown sequence
func (c *Checker) Stop() {
	c.logger.Info("Stopping checker")
	
	// Stop external subsystems first
	c.healthMonitor.Stop()
	
	// Execute worker pool shutdown sequence
	c.stopWorkerPool()
	
	// Log final statistics
	stats := c.GetStats()
	c.logger.Info("Checker stopped", map[string]interface{}{
		"total_processed": stats.ValidCombos + stats.InvalidCombos + stats.ErrorCombos,
		"valid_combos": stats.ValidCombos,
		"invalid_combos": stats.InvalidCombos,
		"error_combos": stats.ErrorCombos,
		"current_cpm": stats.CurrentCPM,
		"elapsed_time": stats.ElapsedTime,
	})
	
	// Close logger
	c.logger.Close()
}

// ============================================================================
// WORKER MANAGEMENT
// ============================================================================

// startWorkerPool spawns N worker goroutines with proper lifecycle tracking
func (c *Checker) startWorkerPool() {
	for i := 0; i < c.Config.MaxWorkers; i++ {
		c.wg.Add(1)
		go c.worker()
	}
}

// startResultProcessor spawns the result processing goroutine with WaitGroup tracking
func (c *Checker) startResultProcessor() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.processResults()
	}()
}

// startTaskGenerator spawns the task generation goroutine with WaitGroup tracking
func (c *Checker) startTaskGenerator() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.generateTasks()
	}()
}

// stopWorkerPool initiates worker pool shutdown sequence with proper ordering
func (c *Checker) stopWorkerPool() {
	// Signal cancellation to all goroutines
	c.cancel()
	
	// Close task channel to signal workers to exit
	close(c.taskChan)
	
	// Wait for all workers and auxiliary goroutines to complete
	c.wg.Wait()
	
	// Close result channel after all workers finished
	close(c.resultChan)
}

// receiveTask receives a task from the task channel with context and close handling
func (c *Checker) receiveTask() (types.WorkerTask, bool) {
	select {
	case <-c.ctx.Done():
		return types.WorkerTask{}, false
	case task, ok := <-c.taskChan:
		return task, ok
	}
}

// sendResult sends a result to the result channel (blocking operation)
func (c *Checker) sendResult(result types.WorkerResult) {
	c.resultChan <- result
}

// worker is the main worker function that processes tasks
func (c *Checker) worker() {
	defer c.wg.Done()

	for {
		task, ok := c.receiveTask()
		if !ok {
			return // Channel closed or context cancelled
		}

		result := c.checkCombo(task)
		c.sendResult(result)
	}
}

// selectProxyForConfig selects appropriate proxy based on config requirements
func (c *Checker) selectProxyForConfig(config types.Config) *types.Proxy {
	if config.RequiresProxy {
		return c.getNextHealthyProxy()
	} else if config.UseProxy {
		return c.getNextProxy()
	}
	return nil
}

// createTask creates a worker task with appropriate proxy selection
func (c *Checker) createTask(combo types.Combo, config types.Config) (types.WorkerTask, bool) {
	// Skip if config requires proxy but none available
	if c.shouldSkipTaskDueToProxy(config) {
		return types.WorkerTask{}, false
	}
	
	proxy := c.selectProxyForConfig(config)
	if config.RequiresProxy && proxy == nil {
		c.logger.Warn(fmt.Sprintf("No proxy available for required proxy config %s", config.Name), nil)
		return types.WorkerTask{}, false
	}
	
	task := types.WorkerTask{
		Combo:  combo,
		Config: config,
		Proxy:  proxy,
	}
	return task, true
}

// sendTaskWithContext sends a task through the channel with context cancellation support
func (c *Checker) sendTaskWithContext(task types.WorkerTask) bool {
	select {
	case <-c.ctx.Done():
		return false
	case c.taskChan <- task:
		return true
	}
}

// generateTasks generates tasks for all combo/config combinations
func (c *Checker) generateTasks() {
	for _, combo := range c.Combos {
		for _, config := range c.Configs {
			task, ok := c.createTask(combo, config)
			if !ok {
				continue
			}
			
			if !c.sendTaskWithContext(task) {
				return // Context cancelled
			}
		}
	}
}

// ============================================================================
// COMBO CHECKING
// ============================================================================

// checkCombo checks a single combo against a config with comprehensive logging
func (c *Checker) checkCombo(task types.WorkerTask) types.WorkerResult {
	start := time.Now()
	correlationID := utils.GenerateCorrelationID()
	taskID := utils.GenerateTaskID("check")
	
	// Log task start
	c.logger.LogTaskStart(taskID, "combo_check", correlationID)
	
	// Create HTTP client with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
retryCount := 0
	var resp *http.Response
	var req *http.Request
	var err error
	
	// Set default retry count if not configured
	retryLimit := c.Config.RetryCount
	if retryLimit == 0 {
		retryLimit = 3 // Default to 3 retries
	}
	
	for retryCount < retryLimit {
		client := c.createHTTPClient(task.Proxy)
		
		// Build request
		req, err = c.buildRequest(task.Combo, task.Config)
		if err != nil {
			// If we can't build the request, don't retry
			c.logger.Error(fmt.Sprintf("Failed to build request for task %s", taskID), err, nil)
			break
		}
		
		// Set request context
		req = req.WithContext(ctx)
		
		// Log detailed request information
		c.logDetailedRequest(req, retryCount+1, correlationID, task.Proxy)
		
		// Execute request
		resp, err = client.Do(req)
		
		if err == nil {
			// Log detailed response information
			c.logDetailedResponse(resp, retryCount+1, correlationID, time.Since(start))
			break // Exit retry loop if request is successful
		}
		
		// Log failed request
		c.logger.LogNetworkRequest(req.Method, req.URL.String(), 0, time.Since(start), task.Proxy, correlationID, err)
		retryCount++
		
		// Only retry if we have more attempts left
		if retryCount < retryLimit {
			c.logger.Warn(fmt.Sprintf("Retrying combo check for task %s (retry %d/%d) - %s", taskID, retryCount, retryLimit, err.Error()), nil)
			
			// For proxy-required configs, try to get a different proxy
			if task.Config.RequiresProxy {
				newProxy := c.getNextHealthyProxy()
				if newProxy != nil {
					task.Proxy = newProxy
				} else {
					c.logger.Warn(fmt.Sprintf("No healthy proxy available for retry %d", retryCount), nil)
					// Continue with current proxy as last resort
				}
			} else if task.Config.UseProxy {
				// Optional proxy usage - try another proxy or go without
				task.Proxy = c.getNextProxy()
			}
			
			// Add a small delay between retries to avoid overwhelming the server
			time.Sleep(time.Duration(500*retryCount) * time.Millisecond)
		}
	}
	
	if err != nil {
		c.logger.LogTaskComplete(taskID, "combo_check", correlationID, time.Since(start), false, err)
		return types.WorkerResult{
			Result: types.CheckResult{
				Combo:     task.Combo,
				Config:    task.Config.Name,
				Status:    "error",
				Error:     err.Error(),
				Timestamp: time.Now(),
			},
			Error: err,
		}
	}
	defer resp.Body.Close()
	
	// Log successful request
	c.logger.LogNetworkRequest(req.Method, req.URL.String(), resp.StatusCode, time.Since(start), task.Proxy, correlationID, nil)

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.LogTaskComplete(taskID, "combo_check", correlationID, time.Since(start), false, err)
		return types.WorkerResult{
			Result: types.CheckResult{
				Combo:     task.Combo,
				Config:    task.Config.Name,
				Status:    "error",
				Error:     err.Error(),
				Timestamp: time.Now(),
				Latency:   int(time.Since(start).Milliseconds()),
			},
			Error: err,
		}
	}

	// Analyze response
	status := c.analyzeResponse(string(body), resp.StatusCode, task.Config)
	duration := time.Since(start)
	
	// Log task completion
	c.logger.LogTaskComplete(taskID, "combo_check", correlationID, duration, status == types.BotStatusSuccess, nil)
	
	return types.WorkerResult{
		Result: types.CheckResult{
			Combo:     task.Combo,
			Config:    task.Config.Name,
			Status:    status,
			Response:  string(body),
			Proxy:     task.Proxy,
			Timestamp: time.Now(),
			Latency:   int(duration.Milliseconds()),
		},
		Error: nil,
	}
}

// ============================================================================
// HTTP CLIENT MANAGEMENT
// ============================================================================

// createHTTPClient creates an azuretls HTTP client with optional proxy and timeout
func (c *Checker) createHTTPClient(proxy *types.Proxy) httpclient.HTTPClientInterface {
	// Enforce maximum 30s timeout
	timeout := time.Duration(c.Config.RequestTimeout) * time.Millisecond
	if timeout > 30*time.Second {
		timeout = 30 * time.Second
	}

	client, err := httpclient.NewAzureTLSClient(proxy, timeout)
	if err != nil {
		// Fallback to standard HTTP client if azuretls fails
		c.logger.Warn("Failed to create azuretls client, falling back to standard HTTP client", map[string]interface{}{
			"error": err.Error(),
		})
		return c.createFallbackHTTPClient(proxy, timeout)
	}

	return client
}

// createFallbackHTTPClient creates a standard HTTP client as fallback
func (c *Checker) createFallbackHTTPClient(proxy *types.Proxy, timeout time.Duration) *http.Client {
	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       100,
	}

	if proxy != nil {
		proxyURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", string(proxy.Type), proxy.Host, proxy.Port))
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

// ============================================================================
// REQUEST BUILDING AND PROCESSING
// ============================================================================

// buildRequest builds an HTTP request from combo and config
func (c *Checker) buildRequest(combo types.Combo, config types.Config) (*http.Request, error) {
	// Replace variables in URL
	url := c.replaceVariables(config.URL, combo)
	
	// Create request
	var req *http.Request
	var err error

	if config.Method == "GET" {
		req, err = http.NewRequest("GET", url, nil)
	} else {
		// Build form data
		formData := c.buildFormData(config.Data, combo)
		req, err = http.NewRequest(config.Method, url, strings.NewReader(formData))
	}

	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range config.Headers {
		req.Header.Set(key, c.replaceVariables(value, combo))
	}

	// Set content type for POST requests
	if config.Method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return req, nil
}

// buildFormData builds form data from config data and combo
func (c *Checker) buildFormData(data map[string]interface{}, combo types.Combo) string {
	var formData []string
	
	for key, value := range data {
		valueStr := fmt.Sprintf("%v", value)
		valueStr = c.replaceVariables(valueStr, combo)
		formData = append(formData, fmt.Sprintf("%s=%s", key, url.QueryEscape(valueStr)))
	}
	
	return strings.Join(formData, "&")
}

// replaceVariables replaces variables in strings with combo values and dynamic variables
func (c *Checker) replaceVariables(text string, combo types.Combo) string {
	// Set combo variables in the variable manipulator
	c.varManipulator.SetVariable("USER", combo.Username, false)
	c.varManipulator.SetVariable("PASS", combo.Password, false)
	c.varManipulator.SetVariable("EMAIL", combo.Email, false)
	c.varManipulator.SetVariable("username", combo.Username, false)
	c.varManipulator.SetVariable("password", combo.Password, false)
	c.varManipulator.SetVariable("email", combo.Email, false)
	
	// Use the variable manipulator for enhanced variable replacement
	return c.varManipulator.ReplaceVariables(text)
}

// analyzeResponse analyzes the response to determine success/failure
func (c *Checker) analyzeResponse(body string, statusCode int, config types.Config) types.BotStatus {
	// Check status codes first
	for _, successCode := range config.SuccessStatus {
		if statusCode == successCode {
			return types.BotStatusSuccess
		}
	}
	
	for _, failureCode := range config.FailureStatus {
		if statusCode == failureCode {
			return types.BotStatusFail
		}
	}

	// Check success strings
	for _, successStr := range config.SuccessStrings {
		if strings.Contains(body, successStr) {
			return types.BotStatusSuccess
		}
	}

	// Check failure strings
	for _, failureStr := range config.FailureStrings {
		if strings.Contains(body, failureStr) {
			return types.BotStatusFail
		}
	}

	// Default to invalid if no specific conditions match
	return types.BotStatusFail
}

// ============================================================================
// RESULT PROCESSING AND STATISTICS
// ============================================================================

// handleResult processes a single worker result with logging and persistence
func (c *Checker) handleResult(result types.WorkerResult) {
	c.updateStats(result.Result)
	
	// Log successful results
	if result.Result.Status == types.BotStatusSuccess {
		c.logger.LogCheckerEvent("valid_combo_found", result.Result, nil)
	}
	
	// Log errors
	if result.Error != nil {
		c.logger.Error("Worker error", result.Error, map[string]interface{}{
			"combo": result.Result.Combo.Username,
			"config": result.Result.Config,
		})
	}
	
	// Save result if needed
	if !c.Config.SaveValidOnly || result.Result.Status == types.BotStatusSuccess {
		c.saveResult(result.Result)
	}
}

// processResults handles the result processing goroutine
func (c *Checker) processResults() {
	for result := range c.resultChan {
		c.handleResult(result)
	}
}

// updateStats updates checker statistics with exclusive write lock
// THREAD-SAFETY: Acquires statsMutex (write lock) to ensure atomic stat updates
// Called concurrently by handleResult() from result processor goroutine
func (c *Checker) updateStats(result types.CheckResult) {
	c.statsMutex.Lock()   // Exclusive write access to Stats
	defer c.statsMutex.Unlock()

	switch result.Status {
	case types.BotStatusSuccess:
		c.Stats.ValidCombos++
	case types.BotStatusFail:
		c.Stats.InvalidCombos++
	case types.BotStatusError:
		c.Stats.ErrorCombos++
	}

	// Update CPM
	elapsed := time.Since(c.Stats.StartTime).Minutes()
	if elapsed > 0 {
		totalChecks := c.Stats.ValidCombos + c.Stats.InvalidCombos + c.Stats.ErrorCombos
		c.Stats.CurrentCPM = float64(totalChecks) / elapsed
	}
}

// saveResult saves a result to file
func (c *Checker) saveResult(result types.CheckResult) {
	if err := c.exporter.ExportResult(result); err != nil {
	log.Printf("[ERROR] Failed to export result: %v", err)
	}
}

// ============================================================================
// PROXY MANAGEMENT
// ============================================================================

// getNextProxy returns the next proxy using the advanced proxy manager
// THREAD-SAFETY: Uses proxyMutex for fallback rotation to ensure atomic index increment
// Called concurrently by createTask() during task generation
func (c *Checker) getNextProxy() *types.Proxy {
	// Use the advanced proxy manager to get the best proxy
	proxy, err := c.proxyManager.GetBestProxy()
	if err != nil {
		// Fallback to simple rotation if advanced manager fails
		c.proxyMutex.Lock()   // Exclusive access to proxyIndex
		defer c.proxyMutex.Unlock()
		
		if len(c.Proxies) == 0 {
			return nil
		}
		
		if c.Config.ProxyRotation {
			proxy := &c.Proxies[c.proxyIndex]
			c.proxyIndex = (c.proxyIndex + 1) % len(c.Proxies)
			return proxy
		}
		
		// Random proxy selection
		return &c.Proxies[rand.Intn(len(c.Proxies))]
	}
	
	return proxy
}

// getNextHealthyProxy returns the next healthy proxy with fallback logic
func (c *Checker) getNextHealthyProxy() *types.Proxy {
	// Try to get a healthy proxy multiple times
	for attempts := 0; attempts < 5; attempts++ {
		proxy := c.getNextProxy()
		if proxy != nil && proxy.Working {
			return proxy
		}
	}
	
	// If no healthy proxy found, return any proxy (might be marked as unhealthy but could still work)
	return c.getNextProxy()
}

// shouldSkipTaskDueToProxy determines if a task should be skipped due to proxy requirements
func (c *Checker) shouldSkipTaskDueToProxy(config types.Config) bool {
	if config.RequiresProxy {
		// Config absolutely requires a proxy
		if len(c.Proxies) == 0 {
			// No proxies available at all
			c.logger.Warn(fmt.Sprintf("Skipping config %s - requires proxy but none available", config.Name), nil)
			return true
		}
		
		// Check if we have any working proxies
		workingProxies := c.getWorkingProxies()
		if len(workingProxies) == 0 {
			c.logger.Warn(fmt.Sprintf("Skipping config %s - requires proxy but all proxies are dead", config.Name), nil)
			return true
		}
	}
	
	return false
}

// parseCombo parses a combo line into a Combo struct
func (c *Checker) parseCombo(line string) *types.Combo {
	// Support different formats: username:password, email:password
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return nil
	}

	combo := &types.Combo{
		Line:     line,
		Username: parts[0],
		Password: parts[1],
	}

	// Check if username looks like an email
	if strings.Contains(combo.Username, "@") {
		combo.Email = combo.Username
	}

	return combo
}

// parseProxy parses a proxy line into a Proxy struct
func (c *Checker) parseProxy(line string) *types.Proxy {
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return nil
	}

	proxy := &types.Proxy{
		Host: parts[0],
		Port: c.parseInt(parts[1]),
		Type: types.ProxyTypeHTTP, // Default to HTTP
	}

	// Try to detect proxy type from line
	if len(parts) > 2 {
		switch strings.ToLower(parts[2]) {
		case "socks4":
			proxy.Type = types.ProxyTypeSOCKS4
		case "socks5":
			proxy.Type = types.ProxyTypeSOCKS5
		case "https":
			proxy.Type = types.ProxyTypeHTTPS
		}
	}

	return proxy
}

// parseInt parses a string to integer
func (c *Checker) parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// GetStats returns current statistics with concurrent read lock
// THREAD-SAFETY: Acquires statsMutex (read lock) allowing concurrent reads
// Safe to call from multiple goroutines simultaneously
func (c *Checker) GetStats() types.CheckerStats {
	c.statsMutex.RLock()  // Shared read access to Stats
	defer c.statsMutex.RUnlock()
	
	stats := *c.Stats
	stats.ElapsedTime = int(time.Since(c.Stats.StartTime).Seconds())
	stats.ActiveWorkers = c.Config.MaxWorkers
	stats.WorkingProxies = len(c.getWorkingProxies())
	
	return stats
}

// getWorkingProxies returns only working proxies
func (c *Checker) getWorkingProxies() []types.Proxy {
	var working []types.Proxy
	for _, proxy := range c.Proxies {
		if proxy.Working {
			working = append(working, proxy)
		}
	}
	return working
}

// ============================================================================
// LOGGING METHODS
// ============================================================================

// logDetailedRequest logs comprehensive request information
func (c *Checker) logDetailedRequest(req *http.Request, reqNumber int, correlationID string, proxy *types.Proxy) {
	// Read request body for logging (need to restore it after)
	var bodyContent string
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			bodyContent = string(bodyBytes)
			// Restore the body for actual request
			req.Body = io.NopCloser(strings.NewReader(bodyContent))
		}
	}
	
	// Format proxy info
	proxyInfo := "direct"
	if proxy != nil {
		proxyInfo = fmt.Sprintf("%s://%s:%d", proxy.Type, proxy.Host, proxy.Port)
	}
	
	// Log detailed request
	c.logger.Info("=== DETAILED REQUEST ===", map[string]interface{}{
		"correlation_id": correlationID,
		"request_number": reqNumber,
		"proxy": proxyInfo,
		"request_details": map[string]interface{}{
			"method": req.Method,
			"url": req.URL.String(),
			"headers": c.formatHeaders(req.Header),
			"cookies": c.formatCookies(req.Cookies()),
			"body": bodyContent,
		},
	})
}

// logDetailedResponse logs comprehensive response information
func (c *Checker) logDetailedResponse(resp *http.Response, reqNumber int, correlationID string, duration time.Duration) {
	// Read response body for logging (need to restore it after)
	var bodyContent string
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			bodyContent = string(bodyBytes)
			// Restore the body for further processing
			resp.Body = io.NopCloser(strings.NewReader(bodyContent))
		}
	}
	
	// Log detailed response
	c.logger.Info("=== DETAILED RESPONSE ===", map[string]interface{}{
		"correlation_id": correlationID,
		"request_number": reqNumber,
		"duration_ms": duration.Milliseconds(),
		"response_details": map[string]interface{}{
			"status_code": resp.StatusCode,
			"status": resp.Status,
			"url": resp.Request.URL.String(),
			"headers": c.formatHeaders(resp.Header),
			"cookies": c.formatResponseCookies(resp.Cookies()),
			"body": c.truncateBody(bodyContent, 2000), // Limit body size in logs
			"body_length": len(bodyContent),
		},
	})
}

// formatHeaders formats HTTP headers for logging
func (c *Checker) formatHeaders(headers http.Header) map[string]string {
	formatted := make(map[string]string)
	for name, values := range headers {
		if len(values) > 0 {
			formatted[name] = strings.Join(values, "; ")
		}
	}
	return formatted
}

// formatCookies formats request cookies for logging
func (c *Checker) formatCookies(cookies []*http.Cookie) map[string]string {
	formatted := make(map[string]string)
	for _, cookie := range cookies {
		formatted[cookie.Name] = cookie.Value
	}
	return formatted
}

// formatResponseCookies formats response cookies for logging
func (c *Checker) formatResponseCookies(cookies []*http.Cookie) []map[string]interface{} {
	var formatted []map[string]interface{}
	for _, cookie := range cookies {
		cookieInfo := map[string]interface{}{
			"name": cookie.Name,
			"value": cookie.Value,
			"domain": cookie.Domain,
			"path": cookie.Path,
			"expires": cookie.Expires.Format(time.RFC3339),
			"secure": cookie.Secure,
			"httponly": cookie.HttpOnly,
		}
		formatted = append(formatted, cookieInfo)
	}
	return formatted
}

// truncateBody truncates response body for logging if it's too long
func (c *Checker) truncateBody(body string, maxLength int) string {
	if len(body) <= maxLength {
		return body
	}
	return body[:maxLength] + fmt.Sprintf("... [truncated, total length: %d]", len(body))
}

// ============================================================================
// PUBLIC TEST METHODS
// ============================================================================

// ShouldSkipTaskDueToProxy exposes the private method for testing
func (c *Checker) ShouldSkipTaskDueToProxy(config types.Config) bool {
	return c.shouldSkipTaskDueToProxy(config)
}

// GetNextProxy exposes the private method for testing
func (c *Checker) GetNextProxy() *types.Proxy {
	return c.getNextProxy()
}

// GetNextHealthyProxy exposes the private method for testing
func (c *Checker) GetNextHealthyProxy() *types.Proxy {
	return c.getNextHealthyProxy()
}

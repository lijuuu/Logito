package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/fatih/color"
)

// Color instances for consistent styling
var (
	headerColor    = color.New(color.FgCyan, color.Bold)
	successColor   = color.New(color.FgGreen, color.Bold)
	warningColor   = color.New(color.FgYellow, color.Bold)
	errorColor     = color.New(color.FgRed, color.Bold)
	infoColor      = color.New(color.FgBlue, color.Bold)
	metricColor    = color.New(color.FgMagenta, color.Bold)
	tableColor     = color.New(color.FgWhite, color.Bold)
	progressColor  = color.New(color.FgCyan)
	separatorColor = color.New(color.FgBlue)
)

// Table formatting functions
func printTableHeader(title string) {
	headerColor.Printf("\n%s\n", title)
	separatorColor.Println(strings.Repeat("═", len(title)))
}

func printTableRow(label string, value interface{}, colorFunc *color.Color) {
	colorFunc.Printf("   %-20s %v\n", label+":", value)
}

func printSeparator() {
	separatorColor.Println(strings.Repeat("─", 80))
}

func printProgressBar(current, total int, width int) string {
	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s] %.1f%%", bar, percentage*100)
}

// LoadTestConfig represents configuration for a load test scenario
type LoadTestConfig struct {
	Name           string        `json:"name"`
	BaseURL        string        `json:"base_url"`
	TotalRequests  int           `json:"total_requests"`
	Concurrency    int           `json:"concurrency"`
	BatchSize      int           `json:"batch_size"`
	RequestTimeout time.Duration `json:"request_timeout"`
}

// LoadTestMetrics holds performance metrics for a load test
type LoadTestMetrics struct {
	TotalRequests   int64           `json:"total_requests"`
	TotalLogs       int64           `json:"total_logs"`
	SuccessfulReqs  int64           `json:"successful_requests"`
	FailedReqs      int64           `json:"failed_requests"`
	TotalDuration   time.Duration   `json:"total_duration"`
	AvgResponseTime time.Duration   `json:"avg_response_time"`
	MinResponseTime time.Duration   `json:"min_response_time"`
	MaxResponseTime time.Duration   `json:"max_response_time"`
	P50ResponseTime time.Duration   `json:"p50_response_time"`
	P90ResponseTime time.Duration   `json:"p90_response_time"`
	P95ResponseTime time.Duration   `json:"p95_response_time"`
	P99ResponseTime time.Duration   `json:"p99_response_time"`
	RequestsPerSec  float64         `json:"requests_per_second"`
	LogsPerSec      float64         `json:"logs_per_second"`
	ErrorRate       float64         `json:"error_rate"`
	ResponseTimes   []time.Duration `json:"-"` // Internal tracking for percentiles
}

// MockLogEntry represents a mock log entry for testing
type MockLogEntry struct {
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	ResourceID string                 `json:"resourceId"`
	Timestamp  time.Time              `json:"timestamp"`
	TraceID    string                 `json:"traceId"`
	SpanID     string                 `json:"spanId"`
	Commit     string                 `json:"commit"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// LoadTester handles load testing operations
type LoadTester struct {
	config        LoadTestConfig
	metrics       LoadTestMetrics
	client        *http.Client
	mu            sync.Mutex
	responseTimes []time.Duration
	startTime     time.Time
	progressChan  chan int
}

// NewLoadTester creates a new load tester instance
func NewLoadTester(config LoadTestConfig) *LoadTester {
	return &LoadTester{
		config:        config,
		client:        &http.Client{Timeout: config.RequestTimeout},
		responseTimes: make([]time.Duration, 0, config.TotalRequests),
		progressChan:  make(chan int, config.TotalRequests),
	}
}

// generateMockLogEntry creates a realistic mock log entry
func (lt *LoadTester) generateMockLogEntry() MockLogEntry {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG", "FATAL"}
	services := []string{"auth-service", "user-service", "payment-service", "notification-service", "api-gateway"}

	return MockLogEntry{
		Level:      levels[rand.Intn(len(levels))],
		Message:    gofakeit.Sentence(rand.Intn(10) + 5), //generate 5-15 word sentence
		ResourceID: fmt.Sprintf("%s-%d", services[rand.Intn(len(services))], rand.Intn(1000)),
		Timestamp:  time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second), //random timestamp within last hour
		TraceID:    gofakeit.UUID(),
		SpanID:     gofakeit.UUID()[:16],
		Commit:     gofakeit.UUID(),
		Metadata: map[string]interface{}{
			"service":          services[rand.Intn(len(services))],
			"version":          fmt.Sprintf("v%d.%d.%d", rand.Intn(5), rand.Intn(10), rand.Intn(100)),
			"environment":      []string{"dev", "staging", "prod"}[rand.Intn(3)],
			"userId":           gofakeit.Number(1, 100000),
			"parentResourceId": gofakeit.UUID(),
			"durationMs":       rand.Intn(5000),
		},
	}
}

// generateBatchLogs creates a batch of mock log entries
func (lt *LoadTester) generateBatchLogs(batchSize int) []MockLogEntry {
	logs := make([]MockLogEntry, batchSize)
	for i := 0; i < batchSize; i++ {
		logs[i] = lt.generateMockLogEntry()
	}
	return logs
}

// calculatePercentiles calculates response time percentiles
func (lt *LoadTester) calculatePercentiles() {
	if len(lt.responseTimes) == 0 {
		return
	}

	// Sort response times for percentile calculation
	sortedTimes := make([]time.Duration, len(lt.responseTimes))
	copy(sortedTimes, lt.responseTimes)
	sort.Slice(sortedTimes, func(i, j int) bool {
		return sortedTimes[i] < sortedTimes[j]
	})

	// Calculate percentiles
	lt.metrics.P50ResponseTime = sortedTimes[int(float64(len(sortedTimes))*0.50)]
	lt.metrics.P90ResponseTime = sortedTimes[int(float64(len(sortedTimes))*0.90)]
	lt.metrics.P95ResponseTime = sortedTimes[int(float64(len(sortedTimes))*0.95)]
	lt.metrics.P99ResponseTime = sortedTimes[int(float64(len(sortedTimes))*0.99)]
}

// calculateAverageResponseTime calculates the proper average response time
func (lt *LoadTester) calculateAverageResponseTime() {
	if len(lt.responseTimes) == 0 {
		return
	}

	var total time.Duration
	for _, rt := range lt.responseTimes {
		total += rt
	}
	lt.metrics.AvgResponseTime = total / time.Duration(len(lt.responseTimes))
}

// sendRequest sends a single request to the log ingestor
func (lt *LoadTester) sendRequest(logs []MockLogEntry) (time.Duration, error) {
	start := time.Now()

	var body interface{}
	if len(logs) == 1 {
		body = logs[0] //single log entry
	} else {
		body = logs //batch of log entries
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal json: %w", err)
	}

	req, err := http.NewRequest("POST", lt.config.BaseURL+"/logs", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := lt.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		return duration, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return duration, nil
}

// worker runs load testing in a worker goroutine
func (lt *LoadTester) worker(requestsPerWorker int, wg *sync.WaitGroup) {
	defer wg.Done()

	for i := 0; i < requestsPerWorker; i++ {
		batchSize := 1
		if lt.config.BatchSize > 1 {
			batchSize = rand.Intn(lt.config.BatchSize) + 1 //random batch size 1 to max
		}

		logs := lt.generateBatchLogs(batchSize)

		duration, err := lt.sendRequest(logs)
		atomic.AddInt64(&lt.metrics.TotalRequests, 1)
		atomic.AddInt64(&lt.metrics.TotalLogs, int64(len(logs)))

		lt.mu.Lock()
		if err != nil {
			atomic.AddInt64(&lt.metrics.FailedReqs, 1)
		} else {
			atomic.AddInt64(&lt.metrics.SuccessfulReqs, 1)
			// Only track successful response times for percentiles
			lt.responseTimes = append(lt.responseTimes, duration)
		}

		//track min/max response times
		if lt.metrics.MinResponseTime == 0 || duration < lt.metrics.MinResponseTime {
			lt.metrics.MinResponseTime = duration
		}
		if duration > lt.metrics.MaxResponseTime {
			lt.metrics.MaxResponseTime = duration
		}
		lt.mu.Unlock()

		// Send progress update
		select {
		case lt.progressChan <- 1:
		default:
		}

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10))) //small random delay
	}
}

// RunLoadTest executes the load test
func (lt *LoadTester) RunLoadTest() LoadTestMetrics {
	headerColor.Println("🚀 Starting Load Test...")
	printTableHeader("📋 Configuration")
	printTableRow("Base URL", lt.config.BaseURL, infoColor)
	printTableRow("Total Requests", lt.config.TotalRequests, infoColor)
	printTableRow("Concurrency", lt.config.Concurrency, infoColor)
	printTableRow("Batch Size", lt.config.BatchSize, infoColor)
	printTableRow("Request Timeout", lt.config.RequestTimeout, infoColor)
	printSeparator()

	lt.startTime = time.Now()

	// Start progress monitor
	go lt.monitorProgress()

	requestsPerWorker := lt.config.TotalRequests / lt.config.Concurrency
	var wg sync.WaitGroup
	for i := 0; i < lt.config.Concurrency; i++ {
		wg.Add(1)
		go lt.worker(requestsPerWorker, &wg)
	}

	wg.Wait()
	close(lt.progressChan)

	lt.metrics.TotalDuration = time.Since(lt.startTime)

	// Calculate all metrics
	lt.calculateAverageResponseTime()
	lt.calculatePercentiles()
	lt.metrics.RequestsPerSec = float64(lt.metrics.TotalRequests) / lt.metrics.TotalDuration.Seconds()
	lt.metrics.LogsPerSec = float64(lt.metrics.TotalLogs) / lt.metrics.TotalDuration.Seconds()
	lt.metrics.ErrorRate = float64(lt.metrics.FailedReqs) / float64(lt.metrics.TotalRequests) * 100

	return lt.metrics
}

// monitorProgress provides real-time progress updates
func (lt *LoadTester) monitorProgress() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentCount := atomic.LoadInt64(&lt.metrics.TotalRequests)
			currentLogs := atomic.LoadInt64(&lt.metrics.TotalLogs)
			elapsed := time.Since(lt.startTime)

			if currentCount > 0 {
				currentRPS := float64(currentCount) / elapsed.Seconds()
				currentLPS := float64(currentLogs) / elapsed.Seconds()
				progressBar := printProgressBar(int(currentCount), lt.config.TotalRequests, 20)

				progressColor.Printf("\r%s | Requests: %d/%d | RPS: %.1f | Logs/sec: %.1f | Elapsed: %v",
					progressBar, currentCount, lt.config.TotalRequests, currentRPS, currentLPS, elapsed.Round(time.Second))
			}

		case <-lt.progressChan:
			// Just consume progress updates
			continue
		}
	}
}

// PrintMetrics prints the load test results
func (lt *LoadTester) PrintMetrics() {
	fmt.Println() // Clear progress line

	printTableHeader("📊 THROUGHPUT METRICS")
	printTableRow("Total Requests", lt.metrics.TotalRequests, metricColor)
	printTableRow("Total Logs", lt.metrics.TotalLogs, metricColor)
	printTableRow("Requests/Second", fmt.Sprintf("%.2f", lt.metrics.RequestsPerSec), metricColor)
	printTableRow("Logs/Second", fmt.Sprintf("%.2f", lt.metrics.LogsPerSec), metricColor)
	printTableRow("Total Duration", lt.metrics.TotalDuration, metricColor)

	printTableHeader("📈 SUCCESS/ERROR METRICS")
	successRate := float64(lt.metrics.SuccessfulReqs) / float64(lt.metrics.TotalRequests) * 100
	printTableRow("Successful", fmt.Sprintf("%d (%.1f%%)", lt.metrics.SuccessfulReqs, successRate), successColor)
	printTableRow("Failed", fmt.Sprintf("%d (%.1f%%)", lt.metrics.FailedReqs, lt.metrics.ErrorRate), errorColor)

	printTableHeader("⏱️  RESPONSE TIME METRICS")
	printTableRow("Average", lt.metrics.AvgResponseTime, infoColor)
	printTableRow("Minimum", lt.metrics.MinResponseTime, infoColor)
	printTableRow("Maximum", lt.metrics.MaxResponseTime, infoColor)
	printTableRow("P50 (Median)", lt.metrics.P50ResponseTime, infoColor)
	printTableRow("P90", lt.metrics.P90ResponseTime, infoColor)
	printTableRow("P95", lt.metrics.P95ResponseTime, infoColor)
	printTableRow("P99", lt.metrics.P99ResponseTime, infoColor)

	printTableHeader("🎯 PERFORMANCE ANALYSIS")
	if lt.metrics.ErrorRate > 5.0 {
		warningColor.Printf("   ⚠️  High error rate detected (%.1f%%) - system may be overwhelmed\n", lt.metrics.ErrorRate)
	} else {
		successColor.Printf("   ✅ Error rate is acceptable (%.1f%%)\n", lt.metrics.ErrorRate)
	}

	if lt.metrics.LogsPerSec < 100 {
		warningColor.Printf("   ⚠️  Low log throughput (%.1f logs/sec) - system may be bottlenecked\n", lt.metrics.LogsPerSec)
	} else {
		successColor.Printf("   ✅ Log throughput is good (%.1f logs/sec)\n", lt.metrics.LogsPerSec)
	}

	if lt.metrics.P95ResponseTime > 1*time.Second {
		warningColor.Printf("   ⚠️  High P95 response time (%v) - consider optimization\n", lt.metrics.P95ResponseTime)
	} else {
		successColor.Printf("   ✅ P95 response time is acceptable (%v)\n", lt.metrics.P95ResponseTime)
	}

	printSeparator()
}

// RunLoadTestSuite runs multiple load tests with increasing intensity
func RunLoadTestSuite() {
	scenarios := []LoadTestConfig{
		{
			Name:           "light_traffic",
			BaseURL:        "http://localhost:3000",
			TotalRequests:  500,
			Concurrency:    5,
			BatchSize:      1,
			RequestTimeout: 5 * time.Second,
		},
		{
			Name:           "moderate_traffic",
			BaseURL:        "http://localhost:3000",
			TotalRequests:  2000,
			Concurrency:    10,
			BatchSize:      5,
			RequestTimeout: 10 * time.Second,
		},
		{
			Name:           "high_traffic",
			BaseURL:        "http://localhost:3000",
			TotalRequests:  4000,
			Concurrency:    20,
			BatchSize:      10,
			RequestTimeout: 10 * time.Second,
		},
		{
			Name:           "peak_traffic",
			BaseURL:        "http://localhost:3000",
			TotalRequests:  7000,
			Concurrency:    50,
			BatchSize:      15,
			RequestTimeout: 15 * time.Second,
		},
		{
			Name:           "stress_traffic",
			BaseURL:        "http://localhost:3000",
			TotalRequests:  10000,
			Concurrency:    100,
			BatchSize:      20,
			RequestTimeout: 20 * time.Second,
		},
		{
			Name:           "burst_traffic",
			BaseURL:        "http://localhost:3000",
			TotalRequests:  3000,
			Concurrency:    300,
			BatchSize:      5,
			RequestTimeout: 5 * time.Second,
		},
	}

	scenarioNames := []string{
		"LIGHT TRAFFIC (5 users)",
		"MODERATE TRAFFIC (10 users)",
		"HIGH TRAFFIC (20 users)",
		"PEAK TRAFFIC (50 users)",
		"STRESS TRAFFIC (100 users)",
		"BURST TRAFFIC (300 users)",
	}

	headerColor.Println("🚀 Logito Load Testing Suite")
	infoColor.Println("Testing log ingestion performance with realistic data")
	separatorColor.Println(strings.Repeat("═", 80))
	fmt.Println()

	var allResults []LoadTestMetrics

	for i, config := range scenarios {
		headerColor.Printf("📋 Scenario %d/%d: %s\n", i+1, len(scenarios), scenarioNames[i])
		separatorColor.Println(strings.Repeat("═", 80))

		loadTester := NewLoadTester(config)
		metrics := loadTester.RunLoadTest()
		loadTester.PrintMetrics()
		allResults = append(allResults, metrics)

		if metrics.ErrorRate > 10.0 {
			warningColor.Printf("⚠️  Warning: High error rate detected (%.2f%%) - system may be overwhelmed\n", metrics.ErrorRate)
		}
		if metrics.LogsPerSec < 50 {
			warningColor.Printf("⚠️  Warning: Low log throughput (%.2f logs/sec) - system may be bottlenecked\n", metrics.LogsPerSec)
		}
		if metrics.P95ResponseTime > 2*time.Second {
			warningColor.Printf("⚠️  Warning: High P95 response time (%v) - consider optimization\n", metrics.P95ResponseTime)
		}

	}

	printTableHeader("📊 Load Test Summary")

	// Table header
	tableColor.Printf("┌%-30s┬%-15s┬%-15s┬%-12s┬%-15s┬%-12s┐\n",
		strings.Repeat("─", 30), strings.Repeat("─", 15), strings.Repeat("─", 15),
		strings.Repeat("─", 12), strings.Repeat("─", 15), strings.Repeat("─", 12))
	tableColor.Printf("│%-30s│%-15s│%-15s│%-12s│%-15s│%-12s│\n",
		"Scenario", "RPS", "Logs/sec", "Error%", "P95", "Duration")
	tableColor.Printf("├%-30s┼%-15s┼%-15s┼%-12s┼%-15s┼%-12s┤\n",
		strings.Repeat("─", 30), strings.Repeat("─", 15), strings.Repeat("─", 15),
		strings.Repeat("─", 12), strings.Repeat("─", 15), strings.Repeat("─", 12))

	// Table rows
	for i, result := range allResults {
		// Color code based on performance
		var rpsColor, logsColor, errorRateColor, p95Color *color.Color

		// RPS color coding
		if result.RequestsPerSec > 1000 {
			rpsColor = successColor
		} else if result.RequestsPerSec > 500 {
			rpsColor = infoColor
		} else {
			rpsColor = warningColor
		}

		// Logs/sec color coding
		if result.LogsPerSec > 5000 {
			logsColor = successColor
		} else if result.LogsPerSec > 1000 {
			logsColor = infoColor
		} else {
			logsColor = warningColor
		}

		// Error rate color coding
		if result.ErrorRate < 1.0 {
			errorRateColor = successColor
		} else if result.ErrorRate < 5.0 {
			errorRateColor = warningColor
		} else {
			errorRateColor = errorColor
		}

		// P95 color coding
		if result.P95ResponseTime < 100*time.Millisecond {
			p95Color = successColor
		} else if result.P95ResponseTime < 1*time.Second {
			p95Color = infoColor
		} else {
			p95Color = warningColor
		}

		// Print row with colors
		fmt.Printf("│%-30s│", scenarioNames[i])
		rpsColor.Printf("%-15.1f", result.RequestsPerSec)
		fmt.Printf("│")
		logsColor.Printf("%-15.1f", result.LogsPerSec)
		fmt.Printf("│")
		errorRateColor.Printf("%-12.1f", result.ErrorRate)
		fmt.Printf("│")
		p95Color.Printf("%-15v", result.P95ResponseTime)
		fmt.Printf("│")
		infoColor.Printf("%-12v", result.TotalDuration.Round(time.Second))
		fmt.Printf("│\n")
	}

	// Table footer
	tableColor.Printf("└%-30s┴%-15s┴%-15s┴%-12s┴%-15s┴%-12s┘\n",
		strings.Repeat("─", 30), strings.Repeat("─", 15), strings.Repeat("─", 15),
		strings.Repeat("─", 12), strings.Repeat("─", 15), strings.Repeat("─", 12))

	summary := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"scenarios": allResults,
	}

	resultsJSON, err := json.MarshalIndent(summary, "", "  ") //format json output
	if err != nil {
		fmt.Printf("error marshaling results to json: %v\n", err)
		return
	}

	// Create benchmark directory if it doesn't exist
	benchmarkDir := "benchmark"
	if err := os.MkdirAll(benchmarkDir, 0755); err != nil {
		fmt.Printf("error creating benchmark directory: %v\n", err)
		return
	}

	// Save results to JSON file
	filename := filepath.Join(benchmarkDir, "progressive_results.json")
	if err := os.WriteFile(filename, resultsJSON, 0644); err != nil {
		fmt.Printf("error writing results to file: %v\n", err)
		return
	}

	successColor.Printf("\n💾 Saving comprehensive results to %s\n", filename)
	successColor.Println("✅ Results saved successfully!")
	headerColor.Println("\n🎉 Load testing completed successfully!")
}

func main() {
	RunLoadTestSuite()
}

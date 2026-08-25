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

type LoadTestConfig struct {
	Name           string        `json:"name"`
	BaseURL        string        `json:"base_url"`
	TotalRequests  int           `json:"total_requests"`
	Concurrency    int           `json:"concurrency"`
	BatchSize      int           `json:"batch_size"`
	RequestTimeout time.Duration `json:"request_timeout"`
}

type LoadTestMetrics struct {
	TotalRequests   int64           `json:"total_requests"`
	TotalLogs       int64           `json:"total_logs"`
	SuccessfulReqs  int64           `json:"successful_requests"`
	TimeoutReqs     int64           `json:"timeout_requests"`
	ActualMissReqs  int64           `json:"actual_miss_requests"`
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
	TimeoutRate     float64         `json:"timeout_rate"`
	ActualMissRate  float64         `json:"actual_miss_rate"`
	SLAMissRate     float64         `json:"sla_miss_rate"`
	ResponseTimes   []time.Duration `json:"-"`
}

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

type LoadTester struct {
	config        LoadTestConfig
	metrics       LoadTestMetrics
	client        *http.Client
	mu            sync.Mutex
	responseTimes []time.Duration
	startTime     time.Time
	progressChan  chan int
}

func NewLoadTester(config LoadTestConfig) *LoadTester {
	// stdlib's default transport caps idle connections at 2 per host, which
	// throttles the client itself once concurrency goes past that - not a
	// server-side limit, but it was skewing these results.
	transport := &http.Transport{
		MaxIdleConns:        config.Concurrency * 2,
		MaxIdleConnsPerHost: config.Concurrency * 2,
		IdleConnTimeout:     90 * time.Second,
	}
	return &LoadTester{
		config:        config,
		client:        &http.Client{Timeout: config.RequestTimeout, Transport: transport},
		responseTimes: make([]time.Duration, 0, config.TotalRequests),
		progressChan:  make(chan int, config.TotalRequests),
	}
}

func (lt *LoadTester) generateMockLogEntry() MockLogEntry {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG", "FATAL"}
	services := []string{"auth-service", "user-service", "payment-service", "notification-service", "api-gateway"}

	return MockLogEntry{
		Level:      levels[rand.Intn(len(levels))],
		Message:    gofakeit.Sentence(rand.Intn(10) + 5),
		ResourceID: fmt.Sprintf("%s-%d", services[rand.Intn(len(services))], rand.Intn(1000)),
		Timestamp:  time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second),
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

func (lt *LoadTester) generateBatchLogs(batchSize int) []MockLogEntry {
	logs := make([]MockLogEntry, batchSize)
	for i := 0; i < batchSize; i++ {
		logs[i] = lt.generateMockLogEntry()
	}
	return logs
}

func (lt *LoadTester) calculatePercentiles() {
	if len(lt.responseTimes) == 0 {
		return
	}

	sortedTimes := make([]time.Duration, len(lt.responseTimes))
	copy(sortedTimes, lt.responseTimes)
	sort.Slice(sortedTimes, func(i, j int) bool {
		return sortedTimes[i] < sortedTimes[j]
	})
	lt.metrics.P50ResponseTime = sortedTimes[int(float64(len(sortedTimes))*0.50)]
	lt.metrics.P90ResponseTime = sortedTimes[int(float64(len(sortedTimes))*0.90)]
	lt.metrics.P95ResponseTime = sortedTimes[int(float64(len(sortedTimes))*0.95)]
	lt.metrics.P99ResponseTime = sortedTimes[int(float64(len(sortedTimes))*0.99)]
}

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

type RequestResult struct {
	Duration     time.Duration
	Error        error
	IsTimeout    bool
	IsActualMiss bool
}

func (lt *LoadTester) sendRequest(logs []MockLogEntry) RequestResult {
	start := time.Now()

	var body interface{}
	if len(logs) == 1 {
		body = logs[0]
	} else {
		body = logs
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return RequestResult{
			Duration:     0,
			Error:        fmt.Errorf("failed to marshal json: %w", err),
			IsTimeout:    false,
			IsActualMiss: true,
		}
	}

	req, err := http.NewRequest("POST", lt.config.BaseURL+"/logs", bytes.NewBuffer(jsonData))
	if err != nil {
		return RequestResult{
			Duration:     0,
			Error:        fmt.Errorf("failed to create request: %w", err),
			IsTimeout:    false,
			IsActualMiss: true,
		}
	}

	req.Header.Set("Content-Type", "application/json")
	if tok := os.Getenv("LOAD_TEST_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := lt.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		isTimeout := strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "deadline exceeded") ||
			strings.Contains(err.Error(), "context deadline exceeded")

		return RequestResult{
			Duration:     duration,
			Error:        fmt.Errorf("failed to send request: %w", err),
			IsTimeout:    isTimeout,
			IsActualMiss: !isTimeout,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RequestResult{
			Duration:     duration,
			Error:        fmt.Errorf("unexpected status code: %d", resp.StatusCode),
			IsTimeout:    false,
			IsActualMiss: true,
		}
	}

	return RequestResult{
		Duration:     duration,
		Error:        nil,
		IsTimeout:    false,
		IsActualMiss: false,
	}
}

func (lt *LoadTester) worker(requestsPerWorker int, wg *sync.WaitGroup) {
	defer wg.Done()

	for i := 0; i < requestsPerWorker; i++ {
		batchSize := 1
		if lt.config.BatchSize > 1 {
			batchSize = rand.Intn(lt.config.BatchSize) + 1
		}

		logs := lt.generateBatchLogs(batchSize)

		result := lt.sendRequest(logs)
		atomic.AddInt64(&lt.metrics.TotalRequests, 1)
		atomic.AddInt64(&lt.metrics.TotalLogs, int64(len(logs)))

		lt.mu.Lock()
		if result.Error != nil {
			if result.IsTimeout {
				atomic.AddInt64(&lt.metrics.TimeoutReqs, 1)
			} else if result.IsActualMiss {
				atomic.AddInt64(&lt.metrics.ActualMissReqs, 1)
			}
		} else {
			atomic.AddInt64(&lt.metrics.SuccessfulReqs, 1)
			lt.responseTimes = append(lt.responseTimes, result.Duration)
		}
		if lt.metrics.MinResponseTime == 0 || result.Duration < lt.metrics.MinResponseTime {
			lt.metrics.MinResponseTime = result.Duration
		}
		if result.Duration > lt.metrics.MaxResponseTime {
			lt.metrics.MaxResponseTime = result.Duration
		}
		lt.mu.Unlock()

		select {
		case lt.progressChan <- 1:
		default:
		}

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)))
	}
}

func (lt *LoadTester) RunLoadTest() LoadTestMetrics {
	headerColor.Println("Starting Load Test...")
	printTableHeader("Configuration")
	printTableRow("Base URL", lt.config.BaseURL, infoColor)
	printTableRow("Total Requests", lt.config.TotalRequests, infoColor)
	printTableRow("Concurrency", lt.config.Concurrency, infoColor)
	printTableRow("Batch Size", lt.config.BatchSize, infoColor)
	printTableRow("Request Timeout", lt.config.RequestTimeout, infoColor)
	printSeparator()

	lt.startTime = time.Now()

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

	lt.calculateAverageResponseTime()
	lt.calculatePercentiles()
	lt.metrics.RequestsPerSec = float64(lt.metrics.TotalRequests) / lt.metrics.TotalDuration.Seconds()
	lt.metrics.LogsPerSec = float64(lt.metrics.TotalLogs) / lt.metrics.TotalDuration.Seconds()
	lt.metrics.TimeoutRate = float64(lt.metrics.TimeoutReqs) / float64(lt.metrics.TotalRequests) * 100
	lt.metrics.ActualMissRate = float64(lt.metrics.ActualMissReqs) / float64(lt.metrics.TotalRequests) * 100
	lt.metrics.SLAMissRate = float64(lt.metrics.TimeoutReqs+lt.metrics.ActualMissReqs) / float64(lt.metrics.TotalRequests) * 100

	return lt.metrics
}

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
			continue
		}
	}
}

func (lt *LoadTester) PrintMetrics() {
	fmt.Println()

	printTableHeader("THROUGHPUT METRICS")
	printTableRow("Total Requests", lt.metrics.TotalRequests, metricColor)
	printTableRow("Total Logs", lt.metrics.TotalLogs, metricColor)
	printTableRow("Requests/Second", fmt.Sprintf("%.2f", lt.metrics.RequestsPerSec), metricColor)
	printTableRow("Logs/Second", fmt.Sprintf("%.2f", lt.metrics.LogsPerSec), metricColor)
	printTableRow("Total Duration", lt.metrics.TotalDuration, metricColor)

	printTableHeader("SUCCESS/MISS METRICS")
	successRate := float64(lt.metrics.SuccessfulReqs) / float64(lt.metrics.TotalRequests) * 100
	printTableRow("Successful", fmt.Sprintf("%d (%.1f%%)", lt.metrics.SuccessfulReqs, successRate), successColor)
	printTableRow("Timeout (SLA Miss)", fmt.Sprintf("%d (%.1f%%)", lt.metrics.TimeoutReqs, lt.metrics.TimeoutRate), warningColor)
	printTableRow("Actual Misses", fmt.Sprintf("%d (%.1f%%)", lt.metrics.ActualMissReqs, lt.metrics.ActualMissRate), errorColor)
	printTableRow("Total SLA Misses", fmt.Sprintf("%d (%.1f%%)", lt.metrics.TimeoutReqs+lt.metrics.ActualMissReqs, lt.metrics.SLAMissRate), errorColor)

	printTableHeader("RESPONSE TIME METRICS")
	printTableRow("Average", lt.metrics.AvgResponseTime, infoColor)
	printTableRow("Minimum", lt.metrics.MinResponseTime, infoColor)
	printTableRow("Maximum", lt.metrics.MaxResponseTime, infoColor)
	printTableRow("P50 (Median)", lt.metrics.P50ResponseTime, infoColor)
	printTableRow("P90", lt.metrics.P90ResponseTime, infoColor)
	printTableRow("P95", lt.metrics.P95ResponseTime, infoColor)
	printTableRow("P99", lt.metrics.P99ResponseTime, infoColor)

	printTableHeader("PERFORMANCE ANALYSIS")
	if lt.metrics.SLAMissRate > 5.0 {
		warningColor.Printf("   WARNING: High SLA miss rate detected (%.1f%%) - system may be overwhelmed\n", lt.metrics.SLAMissRate)
	} else {
		successColor.Printf("   OK: SLA miss rate is acceptable (%.1f%%)\n", lt.metrics.SLAMissRate)
	}

	if lt.metrics.TimeoutRate > 2.0 {
		warningColor.Printf("   WARNING: High timeout rate (%.1f%%) - consider increasing timeout or optimizing performance\n", lt.metrics.TimeoutRate)
	} else {
		successColor.Printf("   OK: Timeout rate is acceptable (%.1f%%)\n", lt.metrics.TimeoutRate)
	}

	if lt.metrics.ActualMissRate > 1.0 {
		errorColor.Printf("   ERROR: High actual miss rate (%.1f%%) - system has real issues\n", lt.metrics.ActualMissRate)
	} else {
		successColor.Printf("   OK: Actual miss rate is low (%.1f%%)\n", lt.metrics.ActualMissRate)
	}

	if lt.metrics.LogsPerSec < 100 {
		warningColor.Printf("   WARNING: Low log throughput (%.1f logs/sec) - system may be bottlenecked\n", lt.metrics.LogsPerSec)
	} else {
		successColor.Printf("   OK: Log throughput is good (%.1f logs/sec)\n", lt.metrics.LogsPerSec)
	}

	if lt.metrics.P95ResponseTime > 1*time.Second {
		warningColor.Printf("   WARNING: High P95 response time (%v) - consider optimization\n", lt.metrics.P95ResponseTime)
	} else {
		successColor.Printf("   OK: P95 response time is acceptable (%v)\n", lt.metrics.P95ResponseTime)
	}

	printSeparator()
}

func RunBatchSizeTestSuite() {
	scenarios := []LoadTestConfig{
		{
			Name:           "batch_1",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  500,
			Concurrency:    5,
			BatchSize:      1,
			RequestTimeout: 10 * time.Second,
		},
		{
			Name:           "batch_5",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  500,
			Concurrency:    5,
			BatchSize:      5,
			RequestTimeout: 10 * time.Second,
		},
		{
			Name:           "batch_10",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  500,
			Concurrency:    5,
			BatchSize:      10,
			RequestTimeout: 10 * time.Second,
		},
		{
			Name:           "batch_25",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  500,
			Concurrency:    5,
			BatchSize:      25,
			RequestTimeout: 10 * time.Second,
		},
		{
			Name:           "batch_50",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  500,
			Concurrency:    5,
			BatchSize:      50,
			RequestTimeout: 10 * time.Second,
		},
		{
			Name:           "batch_100",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  500,
			Concurrency:    5,
			BatchSize:      100,
			RequestTimeout: 10 * time.Second,
		},
		{
			Name:           "batch_200",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  500,
			Concurrency:    5,
			BatchSize:      200,
			RequestTimeout: 15 * time.Second,
		},
	}

	scenarioNames := []string{
		"BATCH SIZE 1",
		"BATCH SIZE 5",
		"BATCH SIZE 10",
		"BATCH SIZE 25",
		"BATCH SIZE 50",
		"BATCH SIZE 100",
		"BATCH SIZE 200",
	}

	headerColor.Println("Logito Batch Size Test Suite")
	infoColor.Println("Testing optimal batch size for maximum throughput")
	separatorColor.Println(strings.Repeat("═", 80))
	fmt.Println()

	var allResults []LoadTestMetrics

	for i, config := range scenarios {
		headerColor.Printf("Scenario %d/%d: %s\n", i+1, len(scenarios), scenarioNames[i])
		separatorColor.Println(strings.Repeat("═", 80))

		loadTester := NewLoadTester(config)
		metrics := loadTester.RunLoadTest()
		loadTester.PrintMetrics()
		allResults = append(allResults, metrics)

		if metrics.SLAMissRate > 10.0 {
			warningColor.Printf("WARNING: High SLA miss rate detected (%.2f%%) - system may be overwhelmed\n", metrics.SLAMissRate)
		}
		if metrics.TimeoutRate > 5.0 {
			warningColor.Printf("WARNING: High timeout rate (%.2f%%) - consider increasing timeout or optimizing performance\n", metrics.TimeoutRate)
		}
		if metrics.ActualMissRate > 2.0 {
			errorColor.Printf("CRITICAL: High actual miss rate (%.2f%%) - system has real issues\n", metrics.ActualMissRate)
		}
		if metrics.LogsPerSec < 50 {
			warningColor.Printf("WARNING: Low log throughput (%.2f logs/sec) - system may be bottlenecked\n", metrics.LogsPerSec)
		}
		if metrics.P95ResponseTime > 2*time.Second {
			warningColor.Printf("WARNING: High P95 response time (%v) - consider optimization\n", metrics.P95ResponseTime)
		}

		// Batch size analysis
		if i > 0 {
			prevResult := allResults[i-1]
			throughputChange := (metrics.LogsPerSec - prevResult.LogsPerSec) / prevResult.LogsPerSec * 100

			if throughputChange > 20 {
				successColor.Printf("📈 BATCH SIZE WIN: Throughput increased %.1f%% with larger batches\n", throughputChange)
			} else if throughputChange < -20 {
				warningColor.Printf("📉 BATCH SIZE LOSS: Throughput decreased %.1f%% with larger batches\n", -throughputChange)
			}
		}

	}

	printTableHeader("Batch Size Test Summary")

	tableColor.Printf("┌%-30s┬%-15s┬%-15s┬%-12s┬%-12s┬%-15s┬%-12s┐\n",
		strings.Repeat("─", 30), strings.Repeat("─", 15), strings.Repeat("─", 15),
		strings.Repeat("─", 12), strings.Repeat("─", 12), strings.Repeat("─", 15), strings.Repeat("─", 12))
	tableColor.Printf("│%-30s│%-15s│%-15s│%-12s│%-12s│%-15s│%-12s│\n",
		"Scenario", "RPS", "Logs/sec", "SLA Miss%", "Timeout%", "P95", "Duration")
	tableColor.Printf("├%-30s┼%-15s┼%-15s┼%-12s┼%-12s┼%-15s┼%-12s┤\n",
		strings.Repeat("─", 30), strings.Repeat("─", 15), strings.Repeat("─", 15),
		strings.Repeat("─", 12), strings.Repeat("─", 12), strings.Repeat("─", 15), strings.Repeat("─", 12))

	for i, result := range allResults {
		var rpsColor, logsColor, slaMissColor, timeoutColor, p95Color *color.Color
		if result.RequestsPerSec > 1000 {
			rpsColor = successColor
		} else if result.RequestsPerSec > 500 {
			rpsColor = infoColor
		} else {
			rpsColor = warningColor
		}

		if result.LogsPerSec > 5000 {
			logsColor = successColor
		} else if result.LogsPerSec > 1000 {
			logsColor = infoColor
		} else {
			logsColor = warningColor
		}

		if result.SLAMissRate < 1.0 {
			slaMissColor = successColor
		} else if result.SLAMissRate < 5.0 {
			slaMissColor = warningColor
		} else {
			slaMissColor = errorColor
		}

		if result.TimeoutRate < 1.0 {
			timeoutColor = successColor
		} else if result.TimeoutRate < 3.0 {
			timeoutColor = warningColor
		} else {
			timeoutColor = errorColor
		}

		if result.P95ResponseTime < 100*time.Millisecond {
			p95Color = successColor
		} else if result.P95ResponseTime < 1*time.Second {
			p95Color = infoColor
		} else {
			p95Color = warningColor
		}
		fmt.Printf("│%-30s│", scenarioNames[i])
		rpsColor.Printf("%-15.1f", result.RequestsPerSec)
		fmt.Printf("│")
		logsColor.Printf("%-15.1f", result.LogsPerSec)
		fmt.Printf("│")
		slaMissColor.Printf("%-12.1f", result.SLAMissRate)
		fmt.Printf("│")
		timeoutColor.Printf("%-12.1f", result.TimeoutRate)
		fmt.Printf("│")
		p95Color.Printf("%-15v", result.P95ResponseTime)
		fmt.Printf("│")
		infoColor.Printf("%-12v", result.TotalDuration.Round(time.Second))
		fmt.Printf("│\n")
	}

	tableColor.Printf("└%-30s┴%-15s┴%-15s┴%-12s┴%-12s┴%-15s┴%-12s┘\n",
		strings.Repeat("─", 30), strings.Repeat("─", 15), strings.Repeat("─", 15),
		strings.Repeat("─", 12), strings.Repeat("─", 12), strings.Repeat("─", 15), strings.Repeat("─", 12))

	summary := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"test_type": "batch_size",
		"scenarios": allResults,
	}

	resultsJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Printf("error marshaling results to json: %v\n", err)
		return
	}

	benchmarkDir := "benchmark"
	if err := os.MkdirAll(benchmarkDir, 0755); err != nil {
		fmt.Printf("error creating benchmark directory: %v\n", err)
		return
	}

	filename := filepath.Join(benchmarkDir, "batch_size_results.json")
	if err := os.WriteFile(filename, resultsJSON, 0644); err != nil {
		fmt.Printf("error writing results to file: %v\n", err)
		return
	}

	successColor.Printf("\nSaving batch size results to %s\n", filename)
	successColor.Println("Batch size test completed successfully!")

	// Batch size analysis
	printSeparator()
	headerColor.Println("BATCH SIZE ANALYSIS")
	printSeparator()

	bestThroughput := 0.0
	bestBatchSize := 0
	optimalBatchSize := 0

	for i, result := range allResults {
		batchSize := scenarios[i].BatchSize
		if result.LogsPerSec > bestThroughput {
			bestThroughput = result.LogsPerSec
			bestBatchSize = batchSize
		}

		// Find where performance starts degrading significantly
		if i > 0 {
			prevResult := allResults[i-1]
			throughputDrop := (prevResult.LogsPerSec - result.LogsPerSec) / prevResult.LogsPerSec * 100
			if throughputDrop > 15 && optimalBatchSize == 0 {
				optimalBatchSize = scenarios[i-1].BatchSize
			}
		}
	}

	infoColor.Printf("📊 Best throughput: %.1f logs/sec with batch size %d\n", bestThroughput, bestBatchSize)
	if optimalBatchSize > 0 {
		warningColor.Printf("⚠️  Performance degradation starts at batch size %d\n", optimalBatchSize)
		infoColor.Printf("💡 Recommended optimal batch size: %d\n", optimalBatchSize)
	} else {
		infoColor.Printf("💡 Larger batches continue to improve performance\n")
	}
}

func RunConcurrencyTestSuite() {
	scenarios := []LoadTestConfig{
		{
			Name:           "concurrency_2",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  200,
			Concurrency:    2,
			BatchSize:      10,
			RequestTimeout: 5 * time.Second,
		},
		{
			Name:           "concurrency_5",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  300,
			Concurrency:    5,
			BatchSize:      10,
			RequestTimeout: 8 * time.Second,
		},
		{
			Name:           "concurrency_10",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  400,
			Concurrency:    10,
			BatchSize:      10,
			RequestTimeout: 10 * time.Second,
		},
		{
			Name:           "concurrency_15",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  500,
			Concurrency:    15,
			BatchSize:      10,
			RequestTimeout: 12 * time.Second,
		},
		{
			Name:           "concurrency_20",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  600,
			Concurrency:    20,
			BatchSize:      10,
			RequestTimeout: 15 * time.Second,
		},
		{
			Name:           "concurrency_30",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  700,
			Concurrency:    30,
			BatchSize:      10,
			RequestTimeout: 20 * time.Second,
		},
		{
			Name:           "concurrency_50",
			BaseURL:        "http://localhost:3001",
			TotalRequests:  800,
			Concurrency:    50,
			BatchSize:      10,
			RequestTimeout: 25 * time.Second,
		},
	}

	scenarioNames := []string{
		"CONCURRENCY 2",
		"CONCURRENCY 5",
		"CONCURRENCY 10",
		"CONCURRENCY 15",
		"CONCURRENCY 20",
		"CONCURRENCY 30",
		"CONCURRENCY 50",
	}

	headerColor.Println("Logito Concurrency Test Suite")
	infoColor.Println("Testing system limits under increasing concurrent load")
	separatorColor.Println(strings.Repeat("═", 80))
	fmt.Println()

	var allResults []LoadTestMetrics

	for i, config := range scenarios {
		headerColor.Printf("Scenario %d/%d: %s\n", i+1, len(scenarios), scenarioNames[i])
		separatorColor.Println(strings.Repeat("═", 80))

		loadTester := NewLoadTester(config)
		metrics := loadTester.RunLoadTest()
		loadTester.PrintMetrics()
		allResults = append(allResults, metrics)

		if metrics.SLAMissRate > 10.0 {
			warningColor.Printf("WARNING: High SLA miss rate detected (%.2f%%) - system may be overwhelmed\n", metrics.SLAMissRate)
		}
		if metrics.TimeoutRate > 5.0 {
			warningColor.Printf("WARNING: High timeout rate (%.2f%%) - consider increasing timeout or optimizing performance\n", metrics.TimeoutRate)
		}
		if metrics.ActualMissRate > 2.0 {
			errorColor.Printf("CRITICAL: High actual miss rate (%.2f%%) - system has real issues\n", metrics.ActualMissRate)
		}
		if metrics.LogsPerSec < 50 {
			warningColor.Printf("WARNING: Low log throughput (%.2f logs/sec) - system may be bottlenecked\n", metrics.LogsPerSec)
		}
		if metrics.P95ResponseTime > 2*time.Second {
			warningColor.Printf("WARNING: High P95 response time (%v) - consider optimization\n", metrics.P95ResponseTime)
		}

		// Concurrency bottleneck analysis
		if i > 0 {
			prevResult := allResults[i-1]
			throughputDrop := (prevResult.LogsPerSec - metrics.LogsPerSec) / prevResult.LogsPerSec * 100
			responseTimeIncrease := float64(metrics.P95ResponseTime-prevResult.P95ResponseTime) / float64(prevResult.P95ResponseTime) * 100

			if throughputDrop > 20 {
				errorColor.Printf("🚨 CONCURRENCY BOTTLENECK: Throughput dropped %.1f%% from previous test\n", throughputDrop)
			}
			if responseTimeIncrease > 100 {
				errorColor.Printf("🚨 CONCURRENCY BOTTLENECK: Response time increased %.1f%% from previous test\n", responseTimeIncrease)
			}
		}

	}

	printTableHeader("Concurrency Test Summary")

	tableColor.Printf("┌%-30s┬%-15s┬%-15s┬%-12s┬%-12s┬%-15s┬%-12s┐\n",
		strings.Repeat("─", 30), strings.Repeat("─", 15), strings.Repeat("─", 15),
		strings.Repeat("─", 12), strings.Repeat("─", 12), strings.Repeat("─", 15), strings.Repeat("─", 12))
	tableColor.Printf("│%-30s│%-15s│%-15s│%-12s│%-12s│%-15s│%-12s│\n",
		"Scenario", "RPS", "Logs/sec", "SLA Miss%", "Timeout%", "P95", "Duration")
	tableColor.Printf("├%-30s┼%-15s┼%-15s┼%-12s┼%-12s┼%-15s┼%-12s┤\n",
		strings.Repeat("─", 30), strings.Repeat("─", 15), strings.Repeat("─", 15),
		strings.Repeat("─", 12), strings.Repeat("─", 12), strings.Repeat("─", 15), strings.Repeat("─", 12))

	for i, result := range allResults {
		var rpsColor, logsColor, slaMissColor, timeoutColor, p95Color *color.Color
		if result.RequestsPerSec > 1000 {
			rpsColor = successColor
		} else if result.RequestsPerSec > 500 {
			rpsColor = infoColor
		} else {
			rpsColor = warningColor
		}

		if result.LogsPerSec > 5000 {
			logsColor = successColor
		} else if result.LogsPerSec > 1000 {
			logsColor = infoColor
		} else {
			logsColor = warningColor
		}

		if result.SLAMissRate < 1.0 {
			slaMissColor = successColor
		} else if result.SLAMissRate < 5.0 {
			slaMissColor = warningColor
		} else {
			slaMissColor = errorColor
		}

		if result.TimeoutRate < 1.0 {
			timeoutColor = successColor
		} else if result.TimeoutRate < 3.0 {
			timeoutColor = warningColor
		} else {
			timeoutColor = errorColor
		}

		if result.P95ResponseTime < 100*time.Millisecond {
			p95Color = successColor
		} else if result.P95ResponseTime < 1*time.Second {
			p95Color = infoColor
		} else {
			p95Color = warningColor
		}
		fmt.Printf("│%-30s│", scenarioNames[i])
		rpsColor.Printf("%-15.1f", result.RequestsPerSec)
		fmt.Printf("│")
		logsColor.Printf("%-15.1f", result.LogsPerSec)
		fmt.Printf("│")
		slaMissColor.Printf("%-12.1f", result.SLAMissRate)
		fmt.Printf("│")
		timeoutColor.Printf("%-12.1f", result.TimeoutRate)
		fmt.Printf("│")
		p95Color.Printf("%-15v", result.P95ResponseTime)
		fmt.Printf("│")
		infoColor.Printf("%-12v", result.TotalDuration.Round(time.Second))
		fmt.Printf("│\n")
	}

	tableColor.Printf("└%-30s┴%-15s┴%-15s┴%-12s┴%-12s┴%-15s┴%-12s┘\n",
		strings.Repeat("─", 30), strings.Repeat("─", 15), strings.Repeat("─", 15),
		strings.Repeat("─", 12), strings.Repeat("─", 12), strings.Repeat("─", 15), strings.Repeat("─", 12))

	summary := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"test_type": "concurrency",
		"scenarios": allResults,
	}

	resultsJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Printf("error marshaling results to json: %v\n", err)
		return
	}

	benchmarkDir := "benchmark"
	if err := os.MkdirAll(benchmarkDir, 0755); err != nil {
		fmt.Printf("error creating benchmark directory: %v\n", err)
		return
	}

	filename := filepath.Join(benchmarkDir, "concurrency_results.json")
	if err := os.WriteFile(filename, resultsJSON, 0644); err != nil {
		fmt.Printf("error writing results to file: %v\n", err)
		return
	}

	successColor.Printf("\nSaving concurrency results to %s\n", filename)
	successColor.Println("Concurrency test completed successfully!")

	// Concurrency bottleneck analysis
	printSeparator()
	headerColor.Println("CONCURRENCY BOTTLENECK ANALYSIS")
	printSeparator()

	bestThroughput := 0.0
	bestConcurrency := 0
	optimalConcurrency := 0

	for i, result := range allResults {
		concurrency := scenarios[i].Concurrency
		if result.LogsPerSec > bestThroughput {
			bestThroughput = result.LogsPerSec
			bestConcurrency = concurrency
		}

		// Find where performance starts degrading significantly
		if i > 0 {
			prevResult := allResults[i-1]
			throughputDrop := (prevResult.LogsPerSec - result.LogsPerSec) / prevResult.LogsPerSec * 100
			if throughputDrop > 15 && optimalConcurrency == 0 {
				optimalConcurrency = scenarios[i-1].Concurrency
			}
		}
	}

	infoColor.Printf("📊 Best throughput: %.1f logs/sec at %d concurrent users\n", bestThroughput, bestConcurrency)
	if optimalConcurrency > 0 {
		warningColor.Printf("⚠️  Performance degradation starts at %d concurrent users\n", optimalConcurrency)
		infoColor.Printf("💡 Recommended max concurrency: %d users\n", optimalConcurrency)
	} else {
		infoColor.Printf("💡 System handled all concurrency levels well\n")
	}
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "batch":
			RunBatchSizeTestSuite()
		case "concurrency":
			RunConcurrencyTestSuite()
		case "both":
			RunBatchSizeTestSuite()
			fmt.Println()
			RunConcurrencyTestSuite()
		default:
			fmt.Println("Usage: go run main.go [batch|concurrency|both]")
			fmt.Println("  batch       - Test optimal batch size")
			fmt.Println("  concurrency - Test concurrency limits")
			fmt.Println("  both        - Run both test suites")
			os.Exit(1)
		}
	} else {
		// Default: run both tests
		RunBatchSizeTestSuite()
		fmt.Println()
		RunConcurrencyTestSuite()
	}
}

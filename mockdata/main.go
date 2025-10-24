package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

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
	TotalRequests   int64         `json:"total_requests"`
	SuccessfulReqs  int64         `json:"successful_requests"`
	FailedReqs      int64         `json:"failed_requests"`
	TotalDuration   time.Duration `json:"total_duration"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	MinResponseTime time.Duration `json:"min_response_time"`
	MaxResponseTime time.Duration `json:"max_response_time"`
	RequestsPerSec  float64       `json:"requests_per_second"`
	ErrorRate       float64       `json:"error_rate"`
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
	config  LoadTestConfig
	metrics LoadTestMetrics
	client  *http.Client
	mu      sync.Mutex
}

// NewLoadTester creates a new load tester instance
func NewLoadTester(config LoadTestConfig) *LoadTester {
	return &LoadTester{
		config: config,
		client: &http.Client{
			Timeout: config.RequestTimeout,
		},
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

		lt.mu.Lock()
		if err != nil {
			atomic.AddInt64(&lt.metrics.FailedReqs, 1)
		} else {
			atomic.AddInt64(&lt.metrics.SuccessfulReqs, 1)
		}

		//track min/max response times
		if lt.metrics.MinResponseTime == 0 || duration < lt.metrics.MinResponseTime {
			lt.metrics.MinResponseTime = duration
		}
		if duration > lt.metrics.MaxResponseTime {
			lt.metrics.MaxResponseTime = duration
		}
		lt.mu.Unlock()

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10))) //small random delay
	}
}

// RunLoadTest executes the load test
func (lt *LoadTester) RunLoadTest() LoadTestMetrics {
	fmt.Printf("starting load test...\n")
	fmt.Printf("configuration:\n")
	fmt.Printf("   - Base URL: %s\n", lt.config.BaseURL)
	fmt.Printf("   - Total Requests: %d\n", lt.config.TotalRequests)
	fmt.Printf("   - Concurrency: %d\n", lt.config.Concurrency)
	fmt.Printf("   - Batch Size: %d\n", lt.config.BatchSize)
	fmt.Printf("   - Request Timeout: %v\n\n", lt.config.RequestTimeout)

	startTime := time.Now()

	requestsPerWorker := lt.config.TotalRequests / lt.config.Concurrency //distribute requests across workers
	var wg sync.WaitGroup
	for i := 0; i < lt.config.Concurrency; i++ {
		wg.Add(1)
		go lt.worker(requestsPerWorker, &wg) //spawn worker goroutines
	}

	wg.Wait() //wait for all workers to complete
	lt.metrics.TotalDuration = time.Since(startTime)
	lt.metrics.RequestsPerSec = float64(lt.metrics.TotalRequests) / lt.metrics.TotalDuration.Seconds() //calculate throughput
	lt.metrics.ErrorRate = float64(lt.metrics.FailedReqs) / float64(lt.metrics.TotalRequests) * 100    //calculate error percentage

	if lt.metrics.SuccessfulReqs > 0 {
		lt.metrics.AvgResponseTime = (lt.metrics.MinResponseTime + lt.metrics.MaxResponseTime) / 2 //simple average calculation
	}

	return lt.metrics
}

// PrintMetrics prints the load test results
func (lt *LoadTester) PrintMetrics() {
	fmt.Printf("\nload test results:\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("Total Requests:     %d\n", lt.metrics.TotalRequests)
	fmt.Printf("Successful:         %d\n", lt.metrics.SuccessfulReqs)
	fmt.Printf("Failed:             %d\n", lt.metrics.FailedReqs)
	fmt.Printf("Error Rate:         %.2f%%\n", lt.metrics.ErrorRate)
	fmt.Printf("Total Duration:     %v\n", lt.metrics.TotalDuration)
	fmt.Printf("Requests/Second:    %.2f\n", lt.metrics.RequestsPerSec)
	fmt.Printf("Avg Response Time:  %v\n", lt.metrics.AvgResponseTime)
	fmt.Printf("Min Response Time:  %v\n", lt.metrics.MinResponseTime)
	fmt.Printf("Max Response Time:  %v\n", lt.metrics.MaxResponseTime)
	fmt.Printf("═══════════════════════════════════════\n")
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

	fmt.Printf("Logito Load Testing Suite\n")
	fmt.Printf("Testing log ingestion performance with realistic data\n")
	fmt.Printf("═══════════════════════════════════════════════════════════\n\n")

	var allResults []LoadTestMetrics

	for i, config := range scenarios {
		fmt.Printf("scenario %d/%d: %s\n", i+1, len(scenarios), scenarioNames[i])
		fmt.Printf("═══════════════════════════════════════════════════════════\n")

		loadTester := NewLoadTester(config)
		metrics := loadTester.RunLoadTest()
		loadTester.PrintMetrics()
		allResults = append(allResults, metrics)

		if metrics.ErrorRate > 50.0 {
			fmt.Printf("warning: high error rate detected (%.2f%%) - system may be overwhelmed\n", metrics.ErrorRate)
		}
		if metrics.RequestsPerSec < 100 {
			fmt.Printf("warning: low throughput (%.2f req/s) - system may be bottlenecked\n", metrics.RequestsPerSec)
		}

	}

	fmt.Printf("\nLoad Test Summary\n")
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	for i, result := range allResults {
		fmt.Printf("%s: %.2f req/s, %.2f%% error rate\n",
			scenarioNames[i], result.RequestsPerSec, result.ErrorRate)
	}

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

	fmt.Printf("\nSaving comprehensive results to %s\n", filename)
	fmt.Printf("Results saved successfully!\n")
	fmt.Printf("\nLoad testing completed successfully!\n")
}

func main() {
	RunLoadTestSuite()
}

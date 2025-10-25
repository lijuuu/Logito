package main

import (
	"strings"
	"time"
)

func runDemo() {
	// Demo with a single scenario
	config := LoadTestConfig{
		Name:           "demo_traffic",
		BaseURL:        "http://localhost:3000",
		TotalRequests:  100,
		Concurrency:    5,
		BatchSize:      3,
		RequestTimeout: 5 * time.Second,
	}

	headerColor.Println("🎯 Demo Load Test")
	separatorColor.Println(strings.Repeat("═", 50))

	loadTester := NewLoadTester(config)
	loadTester.RunLoadTest()
	loadTester.PrintMetrics()

	headerColor.Println("🎉 Demo completed!")
}

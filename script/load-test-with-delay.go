package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Transaction represents the transaction payload
type Transaction struct {
	State         string `json:"state"`
	Amount        string `json:"amount"`
	TransactionID string `json:"transactionId"`
}

// Response represents the API response
type Response struct {
	TransactionID string `json:"transactionId"`
	UserID        uint64 `json:"userId"`
	Success       bool   `json:"success"`
	ResultBalance string `json:"resultBalance,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

// TestResult contains metrics for a single request
type TestResult struct {
	Success      bool
	ResponseTime time.Duration
	StatusCode   int
	Error        error
}

// TestStats contains aggregated test statistics
type TestStats struct {
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	TotalTime          time.Duration
	MinResponseTime    time.Duration
	MaxResponseTime    time.Duration
	TotalResponseTime  time.Duration
	ResponseTimes      []time.Duration
	ErrorCounts        map[string]int
	UserStats          map[int]int    // Track requests per user
	ScenarioStats      map[string]int // Track requests per scenario
	Lock               sync.Mutex
}

// TransactionScenario defines a transaction scenario
type TransactionScenario struct {
	Name   string // For stats tracking
	State  string
	Amount string
}

func main() {

	// Define command line flags
	concurrency := flag.Int("c", 5, "Number of concurrent goroutines")
	totalRequests := flag.Int("n", 100, "Total number of requests to make")
	userIDsStr := flag.String("u", "1,2,3", "Comma-separated list of user IDs to distribute load across")
	baseURL := flag.String("url", "http://localhost:8080", "Base URL for the API")
	sourceType := flag.String("source", "game", "Source-Type header (game, server, or payment)")
	delayMs := flag.Int("delay", 100, "Delay between requests in milliseconds")
	flag.Parse()

	// Parse user IDs
	var userIDs []int
	for _, idStr := range strings.Split(*userIDsStr, ",") {
		var id int
		if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil && id > 0 {
			userIDs = append(userIDs, id)
		}
	}

	// Default to user ID 1 if no valid IDs provided
	if len(userIDs) == 0 {
		userIDs = []int{1}
	}

	// Define transaction scenarios
	scenarios := []TransactionScenario{
		{"Win Small", "win", "10.00"},
		{"Win Medium", "win", "20.00"},
		{"Win Large", "win", "30.00"},
		{"Lose Small", "lose", "15.00"},
		{"Lose Medium", "lose", "40.00"},
		{"Lose Large", "lose", "60.00"},
	}

	fmt.Printf("Load testing API across %d users: %v\n", len(userIDs), userIDs)
	fmt.Printf("Transaction scenarios: %d different combinations\n", len(scenarios))
	fmt.Printf("Concurrency: %d goroutines\n", *concurrency)
	fmt.Printf("Total requests: %d\n", *totalRequests)
	fmt.Printf("Delay between requests: %d ms\n", *delayMs)

	// Initialize test statistics
	stats := &TestStats{
		TotalRequests:   *totalRequests,
		MinResponseTime: time.Hour, // Start with a high value that will be replaced
		ErrorCounts:     make(map[string]int),
		ResponseTimes:   make([]time.Duration, 0, *totalRequests),
		UserStats:       make(map[int]int),
		ScenarioStats:   make(map[string]int),
	}

	// Initialize stats for each user
	for _, id := range userIDs {
		stats.UserStats[id] = 0
	}

	// Initialize stats for each scenario
	for _, scenario := range scenarios {
		stats.ScenarioStats[scenario.Name] = 0
	}

	// Channel to collect results
	results := make(chan TestResult, *totalRequests)

	// Channel to distribute work
	jobs := make(chan int, *totalRequests)

	// Start worker goroutines
	var wg sync.WaitGroup
	fmt.Println("Starting worker goroutines...")
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker(workerID, *baseURL, *sourceType, *delayMs, userIDs, scenarios, jobs, results, stats)
		}(i)
	}

	// Fill the jobs channel
	go func() {
		for i := 0; i < *totalRequests; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	// Start a goroutine to collect results
	go func() {
		for result := range results {
			stats.Lock.Lock()
			if result.Success {
				stats.SuccessfulRequests++
			} else {
				stats.FailedRequests++
				errMsg := "unknown"
				if result.Error != nil {
					errMsg = result.Error.Error()
				}
				stats.ErrorCounts[errMsg]++
			}

			stats.ResponseTimes = append(stats.ResponseTimes, result.ResponseTime)
			stats.TotalResponseTime += result.ResponseTime

			if result.ResponseTime < stats.MinResponseTime {
				stats.MinResponseTime = result.ResponseTime
			}
			if result.ResponseTime > stats.MaxResponseTime {
				stats.MaxResponseTime = result.ResponseTime
			}
			stats.Lock.Unlock()
		}
	}()

	// Start the timer
	startTime := time.Now()
	fmt.Println("Test running...")

	// Print progress periodically
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			stats.Lock.Lock()
			completed := stats.SuccessfulRequests + stats.FailedRequests
			if completed > 0 {
				fmt.Printf("Progress: %d/%d requests completed (%.1f%%)\n",
					completed, stats.TotalRequests, float64(completed)/float64(stats.TotalRequests)*100)
			}
			stats.Lock.Unlock()
		}
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(results)
	ticker.Stop()

	// Calculate the total test time
	stats.TotalTime = time.Since(startTime)

	// Print results
	printResults(stats)
}

func worker(id int, baseURL, sourceType string, delayMs int, userIDs []int,
	scenarios []TransactionScenario, jobs <-chan int, results chan<- TestResult, stats *TestStats) {

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for jobID := range jobs {
		// Optional delay between requests to prevent rate limiting
		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}

		// Randomly select a user ID
		userID := userIDs[rand.Intn(len(userIDs))]

		// Randomly select a transaction scenario
		scenario := scenarios[rand.Intn(len(scenarios))]

		// Update stats for which user and scenario was selected
		stats.Lock.Lock()
		stats.UserStats[userID]++
		stats.ScenarioStats[scenario.Name]++
		stats.Lock.Unlock()

		// Create API URL for this user
		apiURL := fmt.Sprintf("%s/user/%d/transaction", baseURL, userID)

		// Create a unique transaction
		transaction := Transaction{
			State:         scenario.State,
			Amount:        scenario.Amount,
			TransactionID: fmt.Sprintf("test-%d-%d-%d", id, jobID, rand.Intn(1000000)),
		}

		jsonData, err := json.Marshal(transaction)
		if err != nil {
			results <- TestResult{Success: false, Error: err}
			continue
		}

		// Create request
		req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
		if err != nil {
			results <- TestResult{Success: false, Error: err}
			continue
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Source-Type", sourceType)

		// Send the request and measure response time
		startTime := time.Now()
		resp, err := client.Do(req)
		responseTime := time.Since(startTime)

		result := TestResult{
			ResponseTime: responseTime,
		}

		if err != nil {
			result.Success = false
			result.Error = err
		} else {
			statusCode := resp.StatusCode
			result.StatusCode = statusCode
			result.Success = (statusCode >= 200 && statusCode < 300)
			if !result.Success {
				result.Error = fmt.Errorf("HTTP status code %d", statusCode)
			}
			resp.Body.Close()
		}

		results <- result
	}
}

func printResults(stats *TestStats) {
	// Calculate theoretical TPS (ignores actual delays between requests)
	rawTps := float64(stats.SuccessfulRequests) / stats.TotalTime.Seconds()

	// Calculate TPS if all requests were successful
	theoreticalTps := float64(stats.TotalRequests) / stats.TotalTime.Seconds()

	// Calculate success rate adjusted TPS
	adjustedTps := theoreticalTps * (float64(stats.SuccessfulRequests) / float64(stats.TotalRequests))

	// Calculate average response time
	var avgResponseTime time.Duration
	if len(stats.ResponseTimes) > 0 {
		avgResponseTime = stats.TotalResponseTime / time.Duration(len(stats.ResponseTimes))
	}

	// Calculate percentiles
	var p50, p90, p95, p99 time.Duration
	if len(stats.ResponseTimes) > 0 {
		// Sort the response times
		sortedTimes := make([]time.Duration, len(stats.ResponseTimes))
		copy(sortedTimes, stats.ResponseTimes)

		// Simple bubble sort (OK for small datasets)
		for i := 0; i < len(sortedTimes); i++ {
			for j := i + 1; j < len(sortedTimes); j++ {
				if sortedTimes[i] > sortedTimes[j] {
					sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
				}
			}
		}

		p50 = sortedTimes[len(sortedTimes)*50/100]
		p90 = sortedTimes[len(sortedTimes)*90/100]
		p95 = sortedTimes[len(sortedTimes)*95/100]
		p99 = sortedTimes[len(sortedTimes)*99/100]
	}

	// Print results
	fmt.Println("\n================= TEST RESULTS =================")
	fmt.Printf("Total Requests:      %d\n", stats.TotalRequests)
	fmt.Printf("Successful Requests: %d (%.1f%%)\n", stats.SuccessfulRequests,
		float64(stats.SuccessfulRequests)/float64(stats.TotalRequests)*100)
	fmt.Printf("Failed Requests:     %d (%.1f%%)\n", stats.FailedRequests,
		float64(stats.FailedRequests)/float64(stats.TotalRequests)*100)
	fmt.Printf("Total Test Time:     %.2f seconds\n", stats.TotalTime.Seconds())

	fmt.Println("\n----------------- PERFORMANCE -----------------")
	fmt.Printf("Raw TPS:             %.2f (successful requests / total time)\n", rawTps)
	fmt.Printf("Theoretical TPS:     %.2f (if all requests were successful)\n", theoreticalTps)
	fmt.Printf("Success-adjusted TPS: %.2f (theoretical * success rate)\n", adjustedTps)

	fmt.Println("\n----------------- RESPONSE TIMES -----------------")
	fmt.Printf("Average Response:    %v\n", avgResponseTime)
	fmt.Printf("Minimum Response:    %v\n", stats.MinResponseTime)
	fmt.Printf("Maximum Response:    %v\n", stats.MaxResponseTime)
	fmt.Printf("P50 Response:        %v\n", p50)
	fmt.Printf("P90 Response:        %v\n", p90)
	fmt.Printf("P95 Response:        %v\n", p95)
	fmt.Printf("P99 Response:        %v\n", p99)

	// Print user distribution
	fmt.Println("\n----------------- USER DISTRIBUTION -----------------")
	totalUsers := 0
	for _, count := range stats.UserStats {
		totalUsers += count
	}
	for userID, count := range stats.UserStats {
		if count > 0 {
			fmt.Printf("User %d:    %d requests (%.1f%%)\n", userID, count,
				float64(count)/float64(totalUsers)*100)
		}
	}

	// Print scenario distribution
	fmt.Println("\n----------------- SCENARIO DISTRIBUTION -----------------")
	totalScenarios := 0
	for _, count := range stats.ScenarioStats {
		totalScenarios += count
	}
	for scenario, count := range stats.ScenarioStats {
		if count > 0 {
			fmt.Printf("%-15s: %d requests (%.1f%%)\n", scenario, count,
				float64(count)/float64(totalScenarios)*100)
		}
	}

	// Print error distribution if there were errors
	if stats.FailedRequests > 0 {
		fmt.Println("\n----------------- ERROR DISTRIBUTION -----------------")
		for errMsg, count := range stats.ErrorCounts {
			fmt.Printf("%-40s: %d (%.1f%%)\n", errMsg, count,
				float64(count)/float64(stats.TotalRequests)*100)
		}
	}

	// Final conclusion
	fmt.Println("\n================= CONCLUSION =================")
	if theoreticalTps >= 30 {
		fmt.Printf("✅ SYSTEM CAN THEORETICALLY EXCEED the required 30 TPS threshold (%.2f TPS)\n", theoreticalTps)

		if rawTps < 30 {
			fmt.Println("⚠️ But rate limiting or other issues are preventing full performance")
		}
	} else {
		fmt.Printf("❌ SYSTEM DOES NOT MEET the required 30 TPS threshold (%.2f TPS)\n", theoreticalTps)
	}
	fmt.Println("================================================")
}

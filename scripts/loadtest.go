package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type TransferRequest struct {
	MerchantID         int64  `json:"merchant_id"`
	Amount             int64  `json:"amount"`
	DestinationAccount string `json:"destination_account"`
}

func main() {
	baseURL := flag.String("url", "http://localhost:8090", "API base URL")
	merchantID := flag.Int64("merchant", 1, "Merchant ID")
	amount := flag.Int64("amount", 1000, "Transfer amount")
	destination := flag.String("destination", "1234567890", "Destination account")
	total := flag.Int("n", 100, "Total requests")
	concurrency := flag.Int("c", 10, "Concurrent workers")
	targetRPS := flag.Int("rps", 10, "Target requests per second")
	flag.Parse()

	if *total <= 0 || *concurrency <= 0 || *targetRPS <= 0 {
		panic("n, c, and rps must be greater than zero")
	}

	payload, err := json.Marshal(TransferRequest{
		MerchantID:         *merchantID,
		Amount:             *amount,
		DestinationAccount: *destination,
	})
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var (
		success     atomic.Int64
		clientError atomic.Int64
		serverError atomic.Int64
		otherError  atomic.Int64
	)

	jobs := make(chan int, *concurrency)
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range jobs {
				req, err := http.NewRequest(
					http.MethodPost,
					*baseURL+"/transfer",
					bytes.NewReader(payload),
				)
				if err != nil {
					otherError.Add(1)
					continue
				}

				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					otherError.Add(1)
					fmt.Printf("Request failed: %v\n", err)
					continue
				}

				resp.Body.Close()

				switch {
				case resp.StatusCode >= 200 && resp.StatusCode < 300:
					success.Add(1)
				case resp.StatusCode >= 400 && resp.StatusCode < 500:
					clientError.Add(1)
				case resp.StatusCode >= 500:
					serverError.Add(1)
				default:
					otherError.Add(1)
				}
			}
		}()
	}

	interval := time.Second / time.Duration(*targetRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for i := 0; i < *total; i++ {
		<-ticker.C
		jobs <- i
	}

	close(jobs)
	wg.Wait()

	duration := time.Since(start)
	completed := success.Load() + clientError.Load() + serverError.Load() + otherError.Load()

	fmt.Println("\n========== LOAD TEST RESULT ==========")
	fmt.Printf("Target RPS     : %d\n", *targetRPS)
	fmt.Printf("Total Requests : %d\n", *total)
	fmt.Printf("Completed      : %d\n", completed)
	fmt.Printf("2xx Success    : %d\n", success.Load())
	fmt.Printf("4xx Rejected   : %d\n", clientError.Load())
	fmt.Printf("5xx Server Err : %d\n", serverError.Load())
	fmt.Printf("Network Errors : %d\n", otherError.Load())
	fmt.Printf("Duration       : %s\n", duration)
	fmt.Printf("Actual RPS     : %.2f\n", float64(completed)/duration.Seconds())
	fmt.Println("======================================")
}

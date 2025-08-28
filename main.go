package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chromedp/chromedp"
)

type AxeResult struct {
	ID          string   `json:"id"`
	Impact      string   `json:"impact"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	Help        string   `json:"help"`
	HelpURL     string   `json:"helpUrl"`
	Nodes       []Node   `json:"nodes"`
}

type Node struct {
	Impact         string   `json:"impact"`
	HTML           string   `json:"html"`
	Target         []string `json:"target"`
	FailureSummary string   `json:"failureSummary"`
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set CORS headers to allow frontend access
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Get URL parameter
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		targetURL = "Type you URL"
	}

	log.Printf("Analyzing URL: %s", targetURL)

	// Create allocator options for better Chrome detection
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Create chrome context
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var violationsJSON string
	
	err := chromedp.Run(ctx,
		// Navigate to the target URL
		chromedp.Navigate(targetURL),
		
		// Wait for page to load
		chromedp.WaitVisible("body", chromedp.ByQuery),
		
		// Inject axe-core library
		chromedp.Evaluate(`
			(function() {
				const script = document.createElement('script');
				script.src = 'https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.8.2/axe.min.js';
				document.head.appendChild(script);
			})();
		`, nil),
		
		// Wait for axe to be loaded
		chromedp.ActionFunc(func(ctx context.Context) error {
			timeout := time.Now().Add(15 * time.Second)
			for time.Now().Before(timeout) {
				var axeLoaded bool
				err := chromedp.Evaluate(`typeof axe !== 'undefined' && typeof axe.run === 'function'`, &axeLoaded).Do(ctx)
				if err != nil {
					return err
				}
				if axeLoaded {
					log.Println("Axe-core successfully loaded")
					return nil
				}
				time.Sleep(200 * time.Millisecond)
			}
			return fmt.Errorf("timeout waiting for axe-core to load")
		}),
		
		// Set up a global variable to store results and start axe analysis
		chromedp.Evaluate(`
			window.axeResults = null;
			window.axeComplete = false;
			axe.run().then((results) => {
				window.axeResults = JSON.stringify(results.violations);
				window.axeComplete = true;
			}).catch((error) => {
				console.error('Axe run error:', error);
				window.axeResults = JSON.stringify([]);
				window.axeComplete = true;
			});
		`, nil),
		
		// Poll for completion and get results
		chromedp.ActionFunc(func(ctx context.Context) error {
			timeout := time.Now().Add(30 * time.Second)
			for time.Now().Before(timeout) {
				var complete bool
				err := chromedp.Evaluate(`window.axeComplete`, &complete).Do(ctx)
				if err != nil {
					return err
				}
				if complete {
					// Get the results
					err := chromedp.Evaluate(`window.axeResults`, &violationsJSON).Do(ctx)
					if err != nil {
						return err
					}
					log.Println("Axe analysis completed successfully")
					return nil
				}
				time.Sleep(500 * time.Millisecond)
			}
			return fmt.Errorf("timeout waiting for axe analysis to complete")
		}),
	)

	if err != nil {
		log.Printf("Failed to analyze: %v", err)
		errorMsg := fmt.Sprintf(`{"error": "Failed to analyze: %s"}`, err.Error())
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}

	log.Printf("Raw violations JSON length: %d", len(violationsJSON))

	// Handle empty or invalid JSON
	if violationsJSON == "" {
		violationsJSON = "[]"
	}

	// Parse the JSON string into AxeResult slice
	var violations []AxeResult
	if err := json.Unmarshal([]byte(violationsJSON), &violations); err != nil {
		log.Printf("Failed to parse results: %v", err)
		log.Printf("Raw JSON: %s", violationsJSON)
		errorMsg := fmt.Sprintf(`{"error": "Failed to parse results: %s"}`, err.Error())
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully parsed %d violations", len(violations))

	// Create response with additional metadata
	response := map[string]interface{}{
		"url":        targetURL,
		"timestamp":  time.Now().Format(time.RFC3339),
		"violations": violations,
		"count":      len(violations),
	}

	// Return the results as JSON
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/analyze", analyzeHandler)
	
	log.Println("Server starting on port 8000...")
	log.Println("Make sure Chrome/Chromium is installed on your system")
	
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

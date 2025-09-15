package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/yhonda-ohishi/etc_meisai/config"
	"github.com/yhonda-ohishi/etc_meisai/scraper"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not loaded")
	}

	// Load corporate accounts from environment
	accounts, err := config.LoadCorporateAccountsFromEnv()
	if err != nil || len(accounts) == 0 {
		log.Fatal("No accounts found in environment variables")
	}

	account := accounts[0]
	log.Printf("Using account: %s", account.UserID)

	// Calculate date range: Last month's 1st to today
	now := time.Now()
	today := now

	// Get first day of last month
	lastMonth := now.AddDate(0, -1, 0)
	fromDate := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, time.Local)

	// Use today as the end date
	toDate := time.Date(today.Year(), today.Month(), today.Day(), 23, 59, 59, 0, time.Local)

	log.Printf("📅 Date range: %s to %s (先月1日から今日まで)",
		fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))

	// Create download directory
	downloadPath := "./test_downloads"
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		log.Fatalf("Failed to create download directory: %v", err)
	}

	// Configure scraper with improved settings
	config := &scraper.ScraperConfig{
		UserID:       account.UserID,
		Password:     account.Password,
		DownloadPath: downloadPath,
		Headless:     false, // Set to false to see the browser
		Timeout:      30000,
		RetryCount:   3,
		SlowMo:       100,
		UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Viewport: &scraper.ViewportSize{
			Width:  1920,
			Height: 1080,
		},
	}

	// Create and initialize scraper
	s, err := scraper.NewActualETCScraper(config)
	if err != nil {
		log.Fatalf("Failed to create scraper: %v", err)
	}

	log.Println("Initializing browser...")
	if err := s.Initialize(); err != nil {
		log.Fatalf("Failed to initialize scraper: %v", err)
	}
	defer s.Close()

	log.Println("Attempting login...")
	if err := s.Login(); err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	log.Println("Login successful!")

	log.Println("Downloading ETC statements...")
	csvPath, err := s.SearchAndDownloadCSV(fromDate, toDate)
	if err != nil {
		log.Fatalf("Failed to download ETC meisai: %v", err)
	}

	log.Printf("✅ Download completed successfully!")
	log.Printf("CSV file saved to: %s", csvPath)

	// Check if file exists and show size
	if info, err := os.Stat(csvPath); err == nil {
		log.Printf("File size: %d bytes", info.Size())

		// Read and display sample data with date range
		displayCSVDateRange(csvPath)
	}

	log.Println("Test completed successfully!")
}

func displayCSVDateRange(csvPath string) {
	file, err := os.Open(csvPath)
	if err != nil {
		log.Printf("Failed to open CSV: %v", err)
		return
	}
	defer file.Close()

	// Create Shift-JIS decoder
	reader := transform.NewReader(file, japanese.ShiftJIS.NewDecoder())
	csvReader := csv.NewReader(reader)

	log.Println("\n=== CSV Data Summary ===")

	var firstDate, lastDate string
	rowCount := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading CSV: %v", err)
			break
		}

		rowCount++

		// Skip header
		if rowCount == 1 {
			continue
		}

		// Get date from first column (利用年月日（自）)
		if record[0] != "" {
			if firstDate == "" {
				firstDate = record[0]
			}
			lastDate = record[0]
		}

		// Show first 5 data rows
		if rowCount <= 6 {
			fmt.Printf("Row %d: %s %s | %s → %s | ¥%s\n",
				rowCount-1, record[0], record[1], record[4], record[5], record[8])
		}
	}

	log.Printf("\n📊 Total records: %d", rowCount-1)
	log.Printf("📅 Date range in CSV: %s to %s", firstDate, lastDate)
	log.Println("===================================")
}
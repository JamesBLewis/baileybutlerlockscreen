package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	// Define flags
	width := flag.Int("width", 3440, "Width of the screenshot/window")
	height := flag.Int("height", 1440, "Height of the screenshot/window")
	sleepTime := flag.Int("sleep", 10, "Sleep time in minutes between screenshots")
	flag.Parse()

	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting screenshot service...")

	// Create directory for screenshots if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	screenshotDir := filepath.Join(homeDir, "Screenshots")
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Run forever, taking screenshots every sleepTime minutes
	for {
		// Try up to 3 times if there's an error
		var lastErr error
		for attempts := 0; attempts < 3; attempts++ {
			if attempts > 0 {
				log.Printf("Retry attempt %d/3 after failure", attempts+1)
				time.Sleep(30 * time.Second) // Wait between retries
			}

			if err := takeAndSetScreenshot(screenshotDir, *width, *height); err != nil {
				lastErr = err
				log.Printf("Attempt %d failed with error: %v", attempts+1, err)
				continue
			}
			lastErr = nil
			break
		}

		if lastErr != nil {
			log.Printf("All attempts failed. Last error: %v", lastErr)
		} else {
			log.Println("Successfully updated screenshot")
		}

		log.Printf("Waiting %d minutes before next update...", *sleepTime)
		time.Sleep(time.Duration(*sleepTime) * time.Minute)
	}
}

func setLockScreen(path string) error {
	// First try setting the lock screen
	lockCmd := `osascript -e '
        try
            tell application "System Events"
                tell every desktop
                    set pictures folder to "` + filepath.Dir(path) + `"
                    set picture to "` + path + `"
                end tell
            end tell
            return "Success"
        on error errMsg
            return "Error: " & errMsg
        end try'`

	output, err := exec.Command("bash", "-c", lockCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("lock screen command failed: %v, output: %s", err, string(output))
	}

	// If that doesn't work, try setting it as the desktop background first
	if string(output) != "Success" {
		log.Println("Trying alternate method...")

		desktopCmd := `osascript -e '
            try
                tell application "Finder"
                    set desktop picture to POSIX file "` + path + `"
                end tell
                return "Success"
            on error errMsg
                return "Error: " & errMsg
            end try'`

		output, err = exec.Command("bash", "-c", desktopCmd).CombinedOutput()
		if err != nil {
			return fmt.Errorf("desktop background command failed: %v, output: %s", err, string(output))
		}
	}

	return nil
}

func takeAndSetScreenshot(dir string, width, height int) error {
	// Create Chrome instance with optimized options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-background-networking", false),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.WindowSize(width, height),
	)

	// Create allocator context with 2-minute timeout
	allocCtx, allocCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer allocCancel()

	allocatorContext, cancel := chromedp.NewExecAllocator(allocCtx, opts...)
	defer cancel()

	// Create browser context with detailed logging
	ctx, cancel := chromedp.NewContext(
		allocatorContext,
		chromedp.WithLogf(func(format string, args ...interface{}) {
			log.Printf("Browser: "+format, args...)
		}),
		chromedp.WithErrorf(func(format string, args ...interface{}) {
			log.Printf("Browser Error: "+format, args...)
		}),
	)
	defer cancel()

	// Add timeout for the entire operation
	ctx, pageCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer pageCancel()

	// Path for the screenshot
	screenshotPath := filepath.Join(dir, fmt.Sprintf("bailey_status_%s.png", time.Now().Format("20060102_150405")))

	log.Println("Starting browser and navigating to page...")

	// Take screenshot with enhanced error handling and logging
	var buf []byte
	if err := chromedp.Run(ctx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{
			"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
		}),
		chromedp.EmulateViewport(int64(width), int64(height)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Navigating to page...")
			return nil
		}),
		chromedp.Navigate("https://isbaileybutlerintheoffice.today"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Waiting for body to be visible...")
			return nil
		}),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Waiting for full page render...")
			return nil
		}),
		chromedp.Sleep(5*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Taking screenshot...")
			return nil
		}),
		chromedp.FullScreenshot(&buf, 100),
	); err != nil {
		return fmt.Errorf("failed to capture screenshot: %w", err)
	}

	log.Printf("Saving screenshot to: %s", screenshotPath)

	// Save screenshot
	if err := os.WriteFile(screenshotPath, buf, 0644); err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}

	log.Println("Setting as lock screen...")
	if err := setLockScreen(screenshotPath); err != nil {
		return fmt.Errorf("failed to set lock screen: %w", err)
	}

	return nil
}

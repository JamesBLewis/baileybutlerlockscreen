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
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	maxRetries     = 3
	retryDelay     = 30 * time.Second
	browserTimeout = 3 * time.Minute
	renderWait     = 5 * time.Second
	watermarkJS    = `
		const watermark = document.createElement('div');
		watermark.style.cssText = 'position: fixed; top: 50%; left: 50%; transform: translate(-50%, -50%) rotate(-45deg); font-size: 72px; font-family: Arial, sans-serif; color: rgba(0, 0, 0, 0.15); white-space: nowrap; pointer-events: none; user-select: none; z-index: 9999; letter-spacing: 2px;';
		watermark.innerText = 'isbaileybutlerintheoffice.today';
		document.body.appendChild(watermark);

		// Create repeating watermark pattern
		const pattern = document.createElement('div');
		pattern.style.cssText = 'position: fixed; top: 0; left: 0; width: 100%; height: 100%; display: flex; flex-wrap: wrap; justify-content: center; align-items: center; gap: 100px; transform: rotate(-45deg); pointer-events: none; z-index: 9998;';
		
		for (let i = 0; i < 9; i++) {
			const mark = document.createElement('div');
			mark.style.cssText = 'color: rgba(0, 0, 0, 0.075); font-size: 36px; font-family: Arial, sans-serif; white-space: nowrap; letter-spacing: 1px;';
			mark.innerText = 'isbaileybutlerintheoffice.today';
			pattern.appendChild(mark);
		}
		document.body.appendChild(pattern);
	`
)

func main() {
	config := parseFlags()
	setupLogging()
	screenshotDir := setupScreenshotDirectory()

	log.Println("Starting screenshot service...")
	runScreenshotService(config, screenshotDir)
}

type config struct {
	width     int
	height    int
	sleepTime time.Duration
}

func parseFlags() config {
	width := flag.Int("width", 3440, "Width of the screenshot/window")
	height := flag.Int("height", 1440, "Height of the screenshot/window")
	sleepTime := flag.Int("sleep", 10, "Sleep time in minutes between screenshots")
	flag.Parse()

	return config{
		width:     *width,
		height:    *height,
		sleepTime: time.Duration(*sleepTime) * time.Minute,
	}
}

func setupLogging() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func setupScreenshotDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get user home directory:", err)
	}

	screenshotDir := filepath.Join(homeDir, "Screenshots")
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		log.Fatal("Failed to create screenshots directory:", err)
	}
	return screenshotDir
}

func runScreenshotService(cfg config, dir string) {
	for {
		if err := takeScreenshotWithRetry(cfg, dir); err != nil {
			log.Printf("All screenshot attempts failed: %v", err)
		}

		log.Printf("Waiting %d minutes before next update...", cfg.sleepTime/time.Minute)
		time.Sleep(cfg.sleepTime)
	}
}

func takeScreenshotWithRetry(cfg config, dir string) error {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			log.Printf("Retry attempt %d/%d", attempt, maxRetries)
			time.Sleep(retryDelay)
		}

		if err := takeAndSetScreenshot(cfg, dir); err != nil {
			lastErr = fmt.Errorf("attempt %d failed: %w", attempt, err)
			continue
		}
		return nil
	}
	return lastErr
}

func takeAndSetScreenshot(cfg config, dir string) error {
	ctx, cancel := createBrowserContext(cfg)
	defer cancel()

	filename := fmt.Sprintf("bailey_status_%s.png", time.Now().Format("20060102_150405"))
	screenshotPath := filepath.Join(dir, filename)

	var buf []byte
	if err := captureScreenshot(ctx, cfg, &buf); err != nil {
		return fmt.Errorf("failed to capture screenshot: %w", err)
	}

	if err := os.WriteFile(screenshotPath, buf, 0644); err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}

	if err := setLockScreen(screenshotPath); err != nil {
		return fmt.Errorf("failed to set lock screen: %w", err)
	}

	log.Println("Successfully updated screenshot")
	return nil
}

func createBrowserContext(cfg config) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(cfg.width, cfg.height),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx,
		chromedp.WithLogf(log.Printf),
		chromedp.WithErrorf(log.Printf),
	)
	ctx, cancel = context.WithTimeout(ctx, browserTimeout)
	return ctx, cancel
}

func captureScreenshot(ctx context.Context, cfg config, buf *[]byte) error {
	return chromedp.Run(ctx,
		network.Enable(),
		network.SetExtraHTTPHeaders(map[string]interface{}{
			"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		}),
		chromedp.Navigate("https://isbaileybutlerintheoffice.today"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(renderWait),
		// Inject watermark before taking screenshot
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, exp, err := runtime.Evaluate(watermarkJS).Do(ctx)
			if err != nil {
				return err
			}
			if exp != nil {
				return exp
			}
			return nil
		}),
		// Add a small delay to ensure watermark renders
		chromedp.Sleep(500*time.Millisecond),
		chromedp.FullScreenshot(buf, 100),
	)
}

func setLockScreen(path string) error {
	script := fmt.Sprintf(`
		tell application "System Events"
			tell every desktop
				set pictures folder to "%s"
				set picture to "%s"
			end tell
		end tell`, filepath.Dir(path), path)

	cmd := exec.Command("osascript", "-e", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set lock screen: %v (output: %s)", err, output)
	}
	return nil
}

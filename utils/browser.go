package utils

import (
	"context"

	"github.com/chromedp/chromedp"

	"airbnb-scraper-w3e/config"
)

// NewAllocator creates a Chrome exec allocator context from the given Config.
func NewAllocator(parent context.Context, cfg config.Config) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", cfg.Headless),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent(cfg.UserAgent),
		chromedp.WindowSize(1440, 900),
	)
	return chromedp.NewExecAllocator(parent, opts...)
}

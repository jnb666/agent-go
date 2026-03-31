package scrape

import (
	"time"

	"github.com/playwright-community/playwright-go"
)

// Default options if not overridden in NewBrowser or Scrape calls
var DefaultOptions = Options{
	Timeout:           15 * time.Second,
	MaxAge:            8 * time.Hour,
	MaxSpeed:          time.Second,
	WaitUntil:         *playwright.WaitUntilStateNetworkidle,
	Locale:            "en-GB",
	Timezone:          "Europe/London",
	IgnoreHttpsErrors: true,
}

// Options for playwright browser.
type Options struct {
	Timeout           time.Duration // Timeout for each goto request
	MaxAge            time.Duration // Used cached response if age of request less than this
	MaxSpeed          time.Duration // Minimum delay between requests from same host
	WaitFor           time.Duration // Wait after load has completed
	WaitUntil         playwright.WaitUntilState
	Referer           string
	Locale            string
	Timezone          string
	IgnoreHttpsErrors bool
}

type Option func(*Options)

func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

func WithMaxAge(d time.Duration) Option {
	return func(o *Options) {
		o.MaxAge = d
	}
}

func WithMaxSpeed(d time.Duration) Option {
	return func(o *Options) {
		o.MaxSpeed = d
	}
}

func WithWaitFor(d time.Duration) Option {
	return func(o *Options) {
		o.WaitFor = d
	}
}

func WithWaitUntil(s *playwright.WaitUntilState) Option {
	return func(o *Options) {
		o.WaitUntil = *s
	}
}

func WithReferer(ref string) Option {
	return func(o *Options) {
		o.Referer = ref
	}
}

func WithLocale(loc string) Option {
	return func(o *Options) {
		o.Locale = loc
	}
}

func WithTimezone(zone string) Option {
	return func(o *Options) {
		o.Timezone = zone
	}
}

func WithIgnoreHttpsErrors(on bool) Option {
	return func(o *Options) {
		o.IgnoreHttpsErrors = on
	}
}

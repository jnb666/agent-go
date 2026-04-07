// Package scrape provides tools to scrape web page content using Playwright and convert the results to Markdown.
package scrape

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
	log "github.com/sirupsen/logrus"
)

// Browser instance with default options and cache.
type Browser struct {
	playwright *playwright.Playwright
	browser    playwright.Browser
	url        string
	line       int
	cache      map[string]Response
	options    Options
	mu         sync.Mutex
}

// Scrape page response data
type Response struct {
	URL        string
	Title      string
	Markdown   string
	RawHTML    string
	MainHTML   string
	Status     int
	StatusText string
	Timestamp  time.Time
}

var lockErrorRegex = regexp.MustCompile(`ENOENT: no such file or directory, stat '(.+?/firefox/lock)'`)

// Load new firefox browser. Options can be given to override those in DefaultOptions.
func New(options ...Option) (*Browser, error) {
	log.Info("scrape: new browser")
	var err error
	b := &Browser{
		cache:   map[string]Response{},
		options: DefaultOptions,
	}
	for _, opt := range options {
		opt(&b.options)
	}
	b.playwright, err = playwright.Run()
	if err != nil {
		return nil, err
	}
	b.browser, err = b.playwright.Firefox.Launch()
	if err == nil {
		return b, nil
	}
	e := new(playwright.Error)
	if errors.As(err, &e) {
		m := lockErrorRegex.FindStringSubmatch(e.Message)
		if len(m) == 2 {
			log.Warnf("clear stale lock file: %s", m[1])
			if err = os.Remove(m[1]); err == nil {
				b.browser, err = b.playwright.Firefox.Launch()
			}
		}
	}
	return b, err
}

// Close browser and stop playwright
func (b *Browser) Shutdown() {
	log.Info("scrape: shutdown browser")
	if b.browser != nil {
		b.browser.Close()
	}
	b.playwright.Stop()
}

// Most recently scraped URL
func (b *Browser) LastURL() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.url
}

// Current line document is positioned at
func (b *Browser) Line() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.line
}

// Update line from browser tool
func (b *Browser) SetLine(line int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.line = line
}

// Scrape HTML content from given URL and convert to Markdown.  Options can be given to override those from NewBrowser.
func (b *Browser) Scrape(ctx context.Context, uri string, options ...Option) (r Response, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if uri != b.url {
		b.line = 0
	}
	b.url = uri

	opts := b.options
	for _, opt := range options {
		opt(&opts)
	}
	host := getHost(uri)
	for _, domain := range waitDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			opts.WaitFor = max(waitDefault, opts.WaitFor)
		}
	}
	if r, ok := b.cache[uri]; ok && r.Status == 200 && time.Since(r.Timestamp) < opts.MaxAge {
		log.Debugf("scrape: get %s from cache", uri)
		return r, nil
	}
	log.Info("scrape: ", uri)
	if opts.MaxSpeed > 0 {
		b.delay(uri, opts.MaxSpeed)
	}
	r, err = b.scrape(ctx, uri, opts)
	if err != nil {
		return r, err
	}
	r, err = toMarkdown(r)
	if err != nil {
		return r, err
	}

	b.cache[uri] = r
	return r, nil
}

func (b *Browser) delay(uri string, maxSpeed time.Duration) {
	host := getHost(uri)
	latest := 24 * time.Hour
	for key, val := range b.cache {
		if getHost(key) == host {
			latest = min(latest, time.Since(val.Timestamp))
		}
	}
	if latest < maxSpeed {
		d := maxSpeed - latest
		log.Debugf("scrape: sleep for %s", d.Round(time.Millisecond))
		time.Sleep(d)
	}
}

func (b *Browser) scrape(ctx context.Context, uri string, opt Options) (r Response, err error) {
	viewport := playwright.Size{Width: 1280 + rand.IntN(400), Height: 720 + rand.IntN(200)}
	var page playwright.Page

	var c playwright.BrowserContext
	var aborted bool
	c, err = b.browser.NewContext(playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: &opt.IgnoreHttpsErrors,
		Locale:            &opt.Locale,
		TimezoneId:        &opt.Timezone,
		Viewport:          &viewport,
	})
	if err != nil {
		return r, fmt.Errorf("new browser context error: %w", err)
	}
	defer func() {
		if !aborted {
			c.Close()
		}
	}()
	go func() {
		<-ctx.Done()
		aborted = true
		c.Close()
	}()

	page, err = c.NewPage()
	if err != nil {
		return r, fmt.Errorf("new page error: %w", err)
	}
	page.AddInitScript(playwright.Script{Content: &stealthJS})

	page.Route("**/*", func(r playwright.Route) {
		uri := r.Request().URL()
		ext := path.Ext(uri)
		if ext != "" && slices.Contains(mediaExtensions, ext[1:]) {
			log.Trace("skip extension ", uri)
			r.Abort()
			return
		}
		u, err := url.Parse(uri)
		if err == nil && slices.Contains(addServingDomains, u.Hostname()) {
			log.Trace("skip domain ", uri)
			r.Abort()
			return
		}
		log.Trace("get ", uri)
		r.Continue()
	})

	page.OnResponse(func(r playwright.Response) {
		if r.Status() >= 300 && r.Status() < 400 {
			log.Debugf("%d : redirect %s  => %v", r.Status(), r.Request().URL(), r.Headers()["location"])
		}
	})

	var timeout *float64
	if opt.Timeout > 0 {
		timeout = new(float64)
		*timeout = float64(opt.Timeout.Milliseconds())
	}

	gotoOpts := playwright.PageGotoOptions{
		WaitUntil: &opt.WaitUntil,
		Timeout:   timeout,
	}
	if opt.Referer != "" {
		gotoOpts.Referer = &opt.Referer
	}
	resp, err := page.Goto(uri, gotoOpts)
	if err != nil {
		return r, fmt.Errorf("page goto error: %w", err)
	}
	r.Timestamp = time.Now()
	r.URL = uri
	r.Status = resp.Status()
	if text, ok := statusCodes[r.Status]; ok {
		r.StatusText = text
	} else if resp.Ok() {
		r.StatusText = "OK"
	}
	for range 2 {
		r.RawHTML, r.Title, err = getContent(page, opt)
		if err != nil {
			return r, fmt.Errorf("get page content error: %w", err)
		}
		if page.URL() != uri {
			log.Infof("redirected to %s", page.URL())
			r.URL = page.URL()
			if strings.Contains(page.URL(), "consent") {
				err = clickCoookieConsent(page, timeout)
				if err != nil {
					return r, fmt.Errorf("click cookie consent error: %w", err)
				}
				continue
			}
		}
		break
	}
	return r, nil
}

// annoying interstitial cookie pages (e.g. yahoo, google) - try and click through
func clickCoookieConsent(page playwright.Page, timeout *float64) error {
	button, err := getAcceptButton(page)
	if err != nil {
		return err
	}
	log.Info("clicking cookie consent button")
	err = button.Click()
	if err != nil {
		return err
	}
	return page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateLoad, Timeout: timeout})
}

func getAcceptButton(page playwright.Page) (playwright.Locator, error) {
	loc := page.Locator("button")
	n, err := loc.Count()
	if err != nil {
		return nil, err
	}
	if n == 1 {
		return loc, nil
	}
	if n == 0 {
		return nil, fmt.Errorf("error finding consent button")
	}
	buttons, err := loc.All()
	if err != nil {
		return nil, err
	}
	for _, btn := range buttons {
		label, err := btn.GetAttribute("aria-label")
		log.Debugf("button label=%q err=%v", label, err)
		if err == nil && strings.Contains(strings.ToLower(label), "accept") {
			return btn, nil
		}
	}
	return buttons[0], nil
}

func getContent(page playwright.Page, opt Options) (content, title string, err error) {
	for {
		n, _ := page.Locator("meta[http-equiv='refresh']").Count()
		if n == 0 {
			break
		}
		log.Debugf("%s: got meta refresh - waiting", page.URL())
		page.WaitForTimeout(100)
	}
	for n := 0; n < maxRetries; n++ {
		if opt.WaitFor > 0 {
			page.WaitForTimeout(float64(opt.WaitFor.Milliseconds()))
		}
		content, err = page.Content()
		if err == nil || opt.WaitFor == 0 {
			break
		}
		log.Warn(err)
	}
	if err != nil {
		return
	}
	title, err = page.Title()
	return
}

var reStrip = regexp.MustCompile(`(?m)\s*<!--THE END-->\n*`)

func toMarkdown(r Response) (Response, error) {
	// filter tags
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(r.RawHTML))
	if err != nil {
		return r, err
	}
	for _, tag := range tagsToRemove {
		doc.Find(tag).Each(func(n int, s *goquery.Selection) {
			node := s.Nodes[0]
			if tag[0] != '.' && tag[0] != '#' || !slices.Contains(tagsToKeep, node.Data) {
				log.Tracef("remove %s %s", node.Data, tag)
				s.Remove()
			}
		})
	}
	// remove image links
	doc.Find("a").Each(func(n int, s *goquery.Selection) {
		node := s.Nodes[0]
		for _, attr := range node.Attr {
			if attr.Key == "href" {
				if attr.Val == "#" {
					log.Trace("skip link: #")
					s.Remove()
				} else {
					ext := strings.TrimPrefix(path.Ext(attr.Val), ".")
					if ext != "" && slices.Contains(mediaExtensions, ext) {
						log.Tracef("skip link: %q", attr.Val)
						s.Remove()
					}
				}
			}
		}
	})

	r.MainHTML, err = doc.Html()
	if err != nil {
		return r, err
	}
	r.MainHTML = strings.ReplaceAll(r.MainHTML, "\u00a0", " ") // &nbsp; elements
	// convert to markdown
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
		),
	)
	r.Markdown, err = conv.ConvertString(r.MainHTML, converter.WithDomain(r.URL))
	r.Markdown = reStrip.ReplaceAllLiteralString(r.Markdown, "")
	return r, err
}

func getHost(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		log.Error(err)
		return ""
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}

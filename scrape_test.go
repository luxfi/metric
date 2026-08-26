// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A scraper reads three things and nothing else: the status, the content type,
// and the body. These pin all three for every path a scrape can take, so the
// net/http handler and any other transport driving [Scrape] answer alike — the
// handler is a writer over Scrape, and this is what says the two agree.

type stubGatherer struct {
	families []*MetricFamily
	err      error
}

func (s stubGatherer) Gather() ([]*MetricFamily, error) { return s.families, s.err }

func oneFamily() []*MetricFamily {
	return []*MetricFamily{{
		Name:    "hits_total",
		Help:    "how many",
		Type:    MetricTypeCounter,
		Metrics: []Metric{{Value: MetricValue{Value: 3}}},
	}}
}

// answer drives the net/http handler and reports what a scraper would read.
func answer(t *testing.T, g Gatherer, opts HandlerOpts, header map[string]string) (int, string, string, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	HandlerForWithOpts(g, opts).ServeHTTP(rec, req)
	res := rec.Result()
	return res.StatusCode,
		res.Header.Get("Content-Type"),
		res.Header.Get("X-Content-Type-Options"),
		rec.Body.String()
}

func TestScrapeAndTheHandlerAnswerAlike(t *testing.T) {
	for _, tc := range []struct {
		name   string
		g      Gatherer
		opts   HandlerOpts
		header map[string]string
	}{
		{"a plain scrape", stubGatherer{families: oneFamily()}, HandlerOpts{}, nil},
		{"a scrape with no families", stubGatherer{}, HandlerOpts{}, nil},
		{"the scraper names a deadline", stubGatherer{families: oneFamily()}, HandlerOpts{},
			map[string]string{"X-Scrape-Timeout-Seconds": "5"}},
		{"the prometheus spelling of that deadline", stubGatherer{families: oneFamily()}, HandlerOpts{},
			map[string]string{"X-Prometheus-Scrape-Timeout-Seconds": "5"}},
		{"a deadline that is not a number is no deadline", stubGatherer{families: oneFamily()}, HandlerOpts{},
			map[string]string{"X-Scrape-Timeout-Seconds": "soon"}},
		{"gathering fails", stubGatherer{err: errors.New("boom")}, HandlerOpts{}, nil},
		{"gathering fails and the caller said continue", stubGatherer{err: errors.New("boom")},
			HandlerOpts{ErrorHandling: HandlerErrorHandlingContinue}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, ctype, nosniff, body := answer(t, tc.g, tc.opts, tc.header)

			get := func(k string) string {
				if tc.header == nil {
					return ""
				}
				return tc.header[k]
			}
			e := Scrape(context.Background(), tc.g, tc.opts, ScrapeTimeout(get))

			if e.Status != status {
				t.Errorf("Scrape status = %d, the handler answers %d", e.Status, status)
			}
			if got := e.Header["Content-Type"]; got != ctype {
				t.Errorf("Scrape Content-Type = %q, the handler answers %q", got, ctype)
			}
			if got := e.Header["X-Content-Type-Options"]; got != nosniff {
				t.Errorf("Scrape nosniff = %q, the handler answers %q", got, nosniff)
			}
			if got := string(e.Body); got != body {
				t.Errorf("Scrape body = %q, the handler answers %q", got, body)
			}
		})
	}
}

// The content type carries the exposition VERSION, which a scraper reads to
// choose a parser — so it is pinned as a literal rather than compared to itself.
func TestTheExpositionNamesItsVersion(t *testing.T) {
	e := Scrape(context.Background(), stubGatherer{families: oneFamily()}, HandlerOpts{}, 0)
	if got, want := e.Header["Content-Type"], "text/plain; version=0.0.4; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if e.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200", e.Status)
	}
	if len(e.Body) == 0 {
		t.Fatal("body is empty; a gathered family should render")
	}
}

// A failed scrape is a refusal a browser must not sniff, and the body ends in a
// newline — http.Error's shape, which callers on every transport reproduce.
func TestAFailedScrapeIsARefusal(t *testing.T) {
	e := Scrape(context.Background(), stubGatherer{err: errors.New("boom")}, HandlerOpts{}, 0)
	if e.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", e.Status)
	}
	if got, want := e.Header["X-Content-Type-Options"], "nosniff"; got != want {
		t.Fatalf("nosniff = %q, want %q", got, want)
	}
	if got, want := string(e.Body), "metrics gather error\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestScrapeTimeoutReadsBothSpellings(t *testing.T) {
	for _, tc := range []struct {
		name, key, val string
		want           time.Duration
	}{
		{"the short spelling", "X-Scrape-Timeout-Seconds", "2.5", 2500 * time.Millisecond},
		{"the prometheus spelling", "X-Prometheus-Scrape-Timeout-Seconds", "1", time.Second},
		{"not a number", "X-Scrape-Timeout-Seconds", "soon", 0},
		{"zero is no deadline", "X-Scrape-Timeout-Seconds", "0", 0},
		{"negative is no deadline", "X-Scrape-Timeout-Seconds", "-1", 0},
		{"absent", "", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ScrapeTimeout(func(k string) string {
				if k == tc.key {
					return tc.val
				}
				return ""
			})
			if got != tc.want {
				t.Fatalf("ScrapeTimeout = %v, want %v", got, tc.want)
			}
		})
	}
}

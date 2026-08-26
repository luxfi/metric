// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"bytes"
	"context"
	"strconv"
	"time"
)

// Exposition is one rendered scrape: the status to answer with, the headers a
// scraper reads, and the body. Rendering it is separate from writing it, so a
// caller writes it the way its own transport writes.
//
// Every handler in this package is a writer over [Scrape]. A caller whose
// transport is not net/http — a router that hands out its own context and
// writes bytes itself — uses Scrape directly and writes the three fields,
// rather than reimplementing the timeout, the error policy and the content
// type a scraper expects.
type Exposition struct {
	Status int
	Header map[string]string
	Body   []byte
}

// Scrape renders the exposition for gatherer under opts.
//
// timeout is the deadline the scraper asked for; zero means none. A caller on
// net/http reads it with [ScrapeTimeout]; any other transport passes its own
// header lookup to the same function, so the two cannot disagree about which
// header carries it.
//
// A gather error is answered per opts.ErrorHandling: Continue logs and renders
// whatever families were gathered, anything else renders the 500 that
// http.Error writes — the same status, content type, nosniff and trailing
// newline — so a caller writing this value answers exactly as the handler did.
func Scrape(ctx context.Context, gatherer Gatherer, opts HandlerOpts, timeout time.Duration) Exposition {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	families, err := gatherWithContext(ctx, gatherer)
	if err != nil {
		if opts.ErrorHandling != HandlerErrorHandlingContinue {
			return refusal("metrics gather error")
		}
		if opts.ErrorLog != nil {
			opts.ErrorLog.Println("metrics gather error:", err)
		}
	}

	var b bytes.Buffer
	if err := EncodeText(&b, families); err != nil {
		if opts.ErrorHandling == HandlerErrorHandlingContinue && opts.ErrorLog != nil {
			opts.ErrorLog.Println("metrics encode error:", err)
			return Exposition{Status: 200, Header: expositionHeader(), Body: b.Bytes()}
		}
		return refusal("metrics encode error")
	}

	return Exposition{Status: 200, Header: expositionHeader(), Body: b.Bytes()}
}

// ScrapeTimeout reads the deadline a scraper asked for. get is the caller's own
// header lookup — http.Header.Get, or whatever the transport offers — so this
// is the one place that knows which headers carry it.
func ScrapeTimeout(get func(string) string) time.Duration {
	v := get("X-Scrape-Timeout-Seconds")
	if v == "" {
		v = get("X-Prometheus-Scrape-Timeout-Seconds")
	}
	if v == "" {
		return 0
	}
	seconds, err := strconv.ParseFloat(v, 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

// expositionHeader is the content type a Prometheus scraper parses. The version
// is part of the media type, not decoration: a scraper reads it to choose a
// parser.
func expositionHeader() map[string]string {
	return map[string]string{"Content-Type": "text/plain; version=0.0.4; charset=utf-8"}
}

// refusal renders what http.Error writes, so a caller on any transport answers
// a failed scrape identically: the plain-text type, the nosniff that stops a
// browser sniffing the message as markup, and the trailing newline.
func refusal(msg string) Exposition {
	return Exposition{
		Status: 500,
		Header: map[string]string{
			"Content-Type":           "text/plain; charset=utf-8",
			"X-Content-Type-Options": "nosniff",
		},
		Body: []byte(msg + "\n"),
	}
}

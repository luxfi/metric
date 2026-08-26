// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// [Handler] is the zero-configuration exposition: a caller records through the
// package-level constructors and serves it, naming a registry at neither end.
// That only works while the two ends name the SAME registry, and nothing about a
// handler's own answer reveals which one it holds — an empty scrape is a
// well-formed 200. So this records a value and reads it back off the bytes.
func TestHandlerExposesWhatThePackageConstructorsRecord(t *testing.T) {
	const name = "handler_default_probe"

	g := NewGaugeVec(GaugeOpts{Name: name, Help: "a value recorded through the package helpers"}, []string{"lane"})
	g.WithLabelValues("one").Set(7)

	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if body == "" {
		t.Fatal("the exposition is empty; Handler is reading a registry nothing records into")
	}
	if !strings.Contains(body, name) {
		t.Fatalf("the exposition does not carry %q; Handler and NewGaugeVec disagree about the registry.\n%s", name, body)
	}
	if !strings.Contains(body, name+`{lane="one"} 7`) {
		t.Fatalf("%s is named but does not carry the value that was set.\n%s", name, body)
	}
}

// The two names for the default registry are one registry. Holding them apart
// would let the constructors write where the exposition does not read, which is
// the whole failure above with nothing on the wire to show for it.
func TestTheDefaultGathererIsTheDefaultRegistry(t *testing.T) {
	if DefaultGatherer != Gatherer(DefaultRegistry) {
		t.Fatal("DefaultGatherer does not read DefaultRegistry")
	}
	if DefaultRegisterer != Registerer(DefaultRegistry) {
		t.Fatal("DefaultRegisterer does not write DefaultRegistry")
	}
}

package metric

import "testing"

func TestAppendNamespace(t *testing.T) {
	for _, tc := range []struct {
		namespace, name, want string
	}{
		{"coredns", "cache", "coredns_cache"},
		{"coredns", "", "coredns"}, // a namespace with no subsystem
		{"", "cache", "cache"},
		{"", "", ""},
	} {
		if got := AppendNamespace(tc.namespace, tc.name); got != tc.want {
			t.Errorf("AppendNamespace(%q, %q) = %q, want %q", tc.namespace, tc.name, got, tc.want)
		}
	}
}

// TestVecNameWithoutSubsystem pins the name a metric is registered under when
// it declares a namespace and no subsystem. It used to gain a second separator
// from the empty subsystem — coredns__build_info — so a scrape for the name the
// caller asked for found nothing.
func TestVecNameWithoutSubsystem(t *testing.T) {
	for _, tc := range []struct {
		opts CounterOpts
		want string
	}{
		{CounterOpts{Namespace: "coredns", Subsystem: "cache", Name: "hits_total"}, "coredns_cache_hits_total"},
		{CounterOpts{Namespace: "coredns", Name: "build_info"}, "coredns_build_info"},
		{CounterOpts{Name: "bare"}, "bare"},
	} {
		got := prefixedName(AppendNamespace(tc.opts.Namespace, tc.opts.Subsystem), tc.opts.Name)
		if got != tc.want {
			t.Errorf("name for %+v = %q, want %q", tc.opts, got, tc.want)
		}
	}
}

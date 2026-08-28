//go:build metrics_noop

// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

func init() {
	r := NewNoOpRegistry()
	DefaultRegistry = r
	DefaultRegisterer = r
	DefaultGatherer = r
	defaultFactory = NewNoOpFactory()
}

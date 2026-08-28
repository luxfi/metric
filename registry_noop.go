//go:build metrics_noop

// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

// NewRegistry returns a no-op registry when metrics are disabled.
func NewRegistry() Registry {
	return NewNoOpRegistry()
}

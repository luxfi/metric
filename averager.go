// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"errors"
	"fmt"
)

var ErrFailedRegistering = errors.New("failed registering metric")

type Averager interface {
	Observe(float64)
}

type averager struct {
	count Counter
	sum   Gauge
}

func NewAverager(name, desc string, reg Registerer) (Averager, error) {
	errs := &Errs{}
	a := NewAveragerWithErrs(name, desc, reg, errs)
	return a, errs.Err
}

func NewAveragerWithErrs(name, desc string, reg Registerer, errs *Errs) Averager {
	a := averager{
		count: NewCounter(CounterOpts{
			Name: AppendNamespace(name, "count"),
			Help: "Total # of observations of " + desc,
		}),
		sum: NewGauge(GaugeOpts{
			Name: AppendNamespace(name, "sum"),
			Help: "Sum of " + desc,
		}),
	}

	if err := reg.Register(AsCollector(a.count)); err != nil {
		errs.Add(fmt.Errorf("%w: %w", ErrFailedRegistering, err))
	}
	if err := reg.Register(AsCollector(a.sum)); err != nil {
		errs.Add(fmt.Errorf("%w: %w", ErrFailedRegistering, err))
	}
	return &a
}

func (a *averager) Observe(v float64) {
	a.count.Inc()
	a.sum.Add(v)
}

type noAverager struct{}

func NewNoAverager() Averager {
	return noAverager{}
}

func (noAverager) Observe(float64) {}

// AppendNamespace joins a namespace and a name with the "_" prometheus uses,
// skipping the separator when either side is absent.
//
// Both sides are optional. Every New*Vec below calls this as
// AppendNamespace(opts.Namespace, opts.Subsystem) to build a prefix, and a
// metric with a namespace and no subsystem is ordinary — coredns_build_info is
// one. Returning namespace+"_" for that case left a trailing separator, and the
// caller then adds its own, so the metric registered as coredns__build_info and
// nothing scraping for its real name could find it.
func AppendNamespace(namespace, name string) string {
	switch {
	case namespace == "":
		return name
	case name == "":
		return namespace
	default:
		return namespace + "_" + name
	}
}

// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// MultiGatherer extends the Gatherer interface by allowing additional gatherers
// to be registered and deregistered.
type MultiGatherer interface {
	prometheus.Gatherer

	// Register adds the outputs of [gatherer] to the results of future calls to
	// Gather with the provided [namespace] added to the metrics.
	Register(namespace string, gatherer prometheus.Gatherer) error

	// Deregister removes the outputs of a gatherer with [namespace] from the results
	// of future calls to Gather. Returns true if a gatherer with [namespace] was
	// found.
	Deregister(namespace string) bool
}

// NewMultiGatherer returns a new MultiGatherer that merges metrics by namespace
func NewMultiGatherer() MultiGatherer {
	return &multiGatherer{
		gatherers: make(map[string]prometheus.Gatherer),
	}
}

type multiGatherer struct {
	lock      sync.RWMutex
	gatherers map[string]prometheus.Gatherer
}

func (g *multiGatherer) Gather() ([]*dto.MetricFamily, error) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	var result []*dto.MetricFamily
	for _, gatherer := range g.gatherers {
		metrics, err := gatherer.Gather()
		if err != nil {
			return nil, err
		}
		result = append(result, metrics...)
	}
	return result, nil
}

func (g *multiGatherer) Register(namespace string, gatherer prometheus.Gatherer) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if _, exists := g.gatherers[namespace]; exists {
		return fmt.Errorf("gatherer already registered for namespace: %s", namespace)
	}
	g.gatherers[namespace] = gatherer
	return nil
}

func (g *multiGatherer) Deregister(namespace string) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	_, exists := g.gatherers[namespace]
	delete(g.gatherers, namespace)
	return exists
}

// MakeAndRegister creates a new registry and registers it with the gatherer
// Returns our Registry alias which is just *prometheus.Registry
func MakeAndRegister(gatherer MultiGatherer, namespace string) (Registry, error) {
	reg := prometheus.NewRegistry()
	if err := gatherer.Register(namespace, reg); err != nil {
		return nil, fmt.Errorf("couldn't register %q metrics: %w", namespace, err)
	}
	return reg, nil
}

// NewPrefixGatherer returns a new MultiGatherer that adds a prefix to all metrics
func NewPrefixGatherer() MultiGatherer {
	return &prefixGatherer{
		multiGatherer: multiGatherer{
			gatherers: make(map[string]prometheus.Gatherer),
		},
	}
}

type prefixGatherer struct {
	multiGatherer
}

func (g *prefixGatherer) Gather() ([]*dto.MetricFamily, error) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	var result []*dto.MetricFamily
	for namespace, gatherer := range g.gatherers {
		metrics, err := gatherer.Gather()
		if err != nil {
			return nil, err
		}
		// Add namespace prefix to each metric
		for _, mf := range metrics {
			prefixedName := namespace + "_" + *mf.Name
			mf.Name = &prefixedName
		}
		result = append(result, metrics...)
	}
	return result, nil
}

// NewLabelGatherer returns a new MultiGatherer that adds a label to all metrics
func NewLabelGatherer(labelName string) MultiGatherer {
	return &labelGatherer{
		labelName: labelName,
		multiGatherer: multiGatherer{
			gatherers: make(map[string]prometheus.Gatherer),
		},
	}
}

type labelGatherer struct {
	labelName string
	multiGatherer
}

func (g *labelGatherer) Gather() ([]*dto.MetricFamily, error) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	var result []*dto.MetricFamily
	for labelValue, gatherer := range g.gatherers {
		metrics, err := gatherer.Gather()
		if err != nil {
			return nil, err
		}
		// Add label to each metric
		for _, mf := range metrics {
			for _, m := range mf.Metric {
				// Add the label
				labelPair := &dto.LabelPair{
					Name:  &g.labelName,
					Value: &labelValue,
				}
				m.Label = append(m.Label, labelPair)
			}
		}
		result = append(result, metrics...)
	}
	return result, nil
}

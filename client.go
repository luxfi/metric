// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// Client for requesting metrics from a remote Lux Node instance
type Client struct {
	uri string
}

// NewClient returns a new Metrics API Client
func NewClient(uri string) *Client {
	return &Client{
		uri: uri + "/ext/metrics",
	}
}

// GetMetrics returns the metrics from the connected node. The metrics are
// returned as a map of metric family name to the metric family.
func (c *Client) GetMetrics(ctx context.Context) (map[string]*dto.MetricFamily, error) {
	uri, err := url.Parse(c.uri)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		uri.String(),
		bytes.NewReader(nil),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to issue request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	parser := expfmt.NewTextParser(model.UTF8Validation)
	metrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}

	return metrics, nil
}
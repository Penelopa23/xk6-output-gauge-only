package penelopa

import (
	"github.com/castai/promwrite"
	"go.k6.io/k6/metrics"
	"time"
)

// seriesWithMeasure represents a time series with accumulated measurements
type seriesWithMeasure struct {
	TimeSeries metrics.TimeSeries
	Latest     time.Time
	testid     string
	pod        string
	TotalValue float64
}

// newSeriesWithMeasure creates a new seriesWithMeasure instance
func newSeriesWithMeasure(ts metrics.TimeSeries, testid, pod string) *seriesWithMeasure {
	return &seriesWithMeasure{
		TimeSeries: ts,
		Latest:     time.Time{},
		testid:     testid,
		pod:        pod,
		TotalValue: 0,
	}
}

// AddSample adds a sample to the series
func (swm *seriesWithMeasure) AddSample(sample metrics.Sample) {
	if isGauge(swm.TimeSeries.Metric.Name) {
		swm.TotalValue = sample.Value // просто переписать
	} else {
		swm.TotalValue += sample.Value // накапливать
	}
	if sample.Time.After(swm.Latest) {
		swm.Latest = sample.Time
	}
}

// MapPrompb converts the series to Prometheus protobuf format
func (swm *seriesWithMeasure) MapPrompb(renameMap map[string]string, testid string, pod string) []promwrite.TimeSeries {
	origName := swm.TimeSeries.Metric.Name
	mappedName, ok := renameMap[origName]
	if !ok {
		return nil
	}

	tags := swm.TimeSeries.Tags.Map()
	tags["testid"] = testid
	tags["pod"] = pod

	return []promwrite.TimeSeries{{
		Labels: MapStaticLabels(mappedName, tags),
		Sample: promwrite.Sample{
			Time:  swm.Latest,
			Value: swm.TotalValue,
		},
	}}
}

// isGauge determines if a metric should be treated as a gauge
// Based on k6 source code: https://github.com/grafana/k6/blob/master/metrics/builtin.go
// Gauge metrics represent current values and should be overwritten
// Counter/Trend/Rate metrics represent cumulative values and should be accumulated
func isGauge(metric string) bool {
	switch metric {
	case "vus", "vus_max":
		return true
	default:
		return false
	}
}

// isPreservedMetric determines if a metric should be preserved in memory
// even if not updated frequently (like counters, trends, rates)
func isPreservedMetric(metric string) bool {
	switch metric {
	case "http_reqs", "iterations", "checks", "data_sent", "data_received",
		"http_req_duration", "http_req_waiting", "http_req_connecting",
		"http_req_tls_handshaking", "http_req_blocked", "http_req_receiving",
		"http_req_sending", "iteration_duration", "group_duration",
		"ws_sessions", "ws_msgs_sent", "ws_msgs_received", "ws_ping",
		"ws_session_duration", "ws_connecting", "grpc_req_duration",
		"dropped_iterations", "http_req_failed":
		return true
	default:
		return false
	}
}

// MapStaticLabels creates Prometheus labels from a name and tags
func MapStaticLabels(name string, tags map[string]string) []promwrite.Label {
	labels := []promwrite.Label{
		{Name: "__name__", Value: name},
	}
	for k, v := range tags {
		labels = append(labels, promwrite.Label{Name: k, Value: v})
	}
	return labels
}

// extractName extracts the metric name from labels
func extractName(labels []promwrite.Label) string {
	for _, label := range labels {
		if label.Name == "__name__" {
			return label.Value
		}
	}
	return "unknown"
}

// extractTags extracts tags from various types
func extractTags(ts interface{}) map[string]string {
	switch tagSet := ts.(type) {
	case map[string]string:
		return tagSet
	case *metrics.TagSet:
		return tagSet.Map()
	default:
		return map[string]string{}
	}
}

// buildLabels creates Prometheus labels from name and tags
func buildLabels(name string, tags map[string]string) []promwrite.Label {
	labels := make([]promwrite.Label, 0, len(tags)+1)
	labels = append(labels, promwrite.Label{Name: "__name__", Value: name})
	for k, v := range tags {
		labels = append(labels, promwrite.Label{Name: k, Value: v})
	}
	return labels
} 
package penelopa

import (
	"context"
	"github.com/castai/promwrite"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/metrics"
	"go.k6.io/k6/output"
	"runtime"
	"sync"
	"time"
)

// Output represents the penelopa output module
type Output struct {
	output.SampleBuffer
	logger        logrus.FieldLogger
	now           func() time.Time
	ctx           context.Context
	cancel        context.CancelFunc
	client        *promwrite.Client
	flushInterval time.Duration
	seriesMap     map[string]*promwrite.TimeSeries
	tsdb          map[metrics.TimeSeries]*seriesWithMeasure
	renameMap     map[string]string
	metricsBuffer sync.Pool
	testid        string
	pod           string
	interval      string
	batchSize     int64
	// Cleanup mechanism
	lastCleanup     time.Time
	cleanupInterval time.Duration
	maxSeriesAge    time.Duration
}

// New creates a new penelopa output instance
func New(params output.Params) (output.Output, error) {
	ctx, cancel := context.WithCancel(context.Background())

	config, err := GetConsolidatedConfig(params.JSONConfig, params.Environment, params.ConfigArgument)
	if err != nil {
		return nil, err
	}

	flushInterval := config.PushInterval.TimeDuration()
	testid := config.TestId.String
	pod := config.Pod.String
	batchSize := config.BatchSize.Int64

	client := promwrite.NewClient(config.ServerURL.String)

	renaming := map[string]string{
		"vus":                      "k6_vus",
		"vus_max":                  "k6_vus_max",
		"iterations":               "k6_iterations_total",
		"http_reqs":                "k6_http_reqs_total",
		"http_req_duration":        "k6_http_req_duration",
		"http_req_waiting":         "k6_http_req_waiting",
		"http_req_failed":          "k6_http_req_failed",
		"http_req_blocked":         "k6_http_req_blocked",
		"data_sent":                "k6_data_sent",
		"data_received":            "k6_data_received",
		"iteration_duration":       "k6_iteration_duration",
		"dropped_duration":         "k6_dropped_duration",
		"checks":                   "k6_checks",
		"http_req_sending":         "k6_http_req_sending",
		"http_req_receiving":       "k6_http_req_receiving",
		"http_req_tls_handshaking": "k6_http_req_tls_handshaking",
	}

	o := &Output{
		now:             time.Now,
		logger:          params.Logger,
		client:          client,
		ctx:             ctx,
		cancel:          cancel,
		flushInterval:   flushInterval,
		renameMap:       renaming,
		seriesMap:       map[string]*promwrite.TimeSeries{},
		tsdb:            make(map[metrics.TimeSeries]*seriesWithMeasure),
		testid:          testid,
		pod:             pod,
		batchSize:       batchSize,
		lastCleanup:     time.Now(),
		cleanupInterval: 1 * time.Minute, // Cleanup every minute
		maxSeriesAge:    10 * time.Minute, // Remove series older than 10 minutes
		metricsBuffer: sync.Pool{
			New: func() interface{} {
				return make([]promwrite.TimeSeries, 0, 1000)
			},
		},
	}
	return o, nil
}

// Description returns the output description
func (o *Output) Description() string {
	return "Penelopa Prometheus remote write output"
}

// Start starts the output module
func (o *Output) Start() error {
	go o.periodicFlush()
	return nil
}

// Stop stops the output module
func (o *Output) Stop() error {
	o.cancel()
	o.flush()
	return nil
}

// periodicFlush runs the periodic flush loop
func (o *Output) periodicFlush() {
	ticker := time.NewTicker(o.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			o.flush()
		case <-o.ctx.Done():
			return
		}
	}
}

// flush processes and sends metrics
func (o *Output) flush() {
	var (
		start = time.Now()
		nts   int
	)

	defer func() {
		d := time.Since(start)
		msg := "[PENELOPA] Successfully flushed time series"
		if d > o.flushInterval {
			o.logger.WithField("nts", nts).Warnf("%s, but it took %s (interval is %s)", msg, d, o.flushInterval)
		} else {
			o.logger.WithField("nts", nts).Debugf(msg)
		}
	}()

	// Check if cleanup is needed
	if time.Since(o.lastCleanup) > o.cleanupInterval {
		o.cleanupOldSeries()
		o.lastCleanup = time.Now()
	}

	samplesContainers := o.GetBufferedSamples()
	seen := o.tsdb

	for _, container := range samplesContainers {
		for _, sample := range container.GetSamples() {
			name := sample.TimeSeries.Metric.Name
			_, ok := o.renameMap[name]
			if !ok {
				continue
			}
			ts := sample.TimeSeries
			swm, exists := seen[ts]
			if !exists {
				swm = newSeriesWithMeasure(ts, o.testid, o.pod)
				seen[ts] = swm
			}
			swm.AddSample(sample)
		}
	}

	var series []promwrite.TimeSeries
	for _, swm := range seen {
		series = append(series, swm.MapPrompb(o.renameMap, o.testid, o.pod)...)
	}

	// Add memory metrics
	memSeries := o.getMemoryMetrics()
	series = append(series, memSeries...)

	err, _ := o.client.Write(o.ctx, &promwrite.WriteRequest{TimeSeries: series})
	if err != nil {
		o.logger.Debugf("[PENELOPA] Failed to push metrics: %v", err)
	} else {
		o.logger.Debugf("[PENELOPA] Successfully pushed %d timeseries", len(series))
	}
}

// cleanupOldSeries removes series that haven't been updated recently
// Preserved metrics (like counters, trends, rates) are kept even if old
func (o *Output) cleanupOldSeries() {
	now := time.Now()
	cutoff := now.Add(-o.maxSeriesAge)
	removed := 0

	for ts, swm := range o.tsdb {
		// Don't remove preserved metrics even if they're old
		if isPreservedMetric(ts.Metric.Name) {
			continue
		}
		
		if swm.Latest.Before(cutoff) {
			delete(o.tsdb, ts)
			removed++
		}
	}

	if removed > 0 {
		o.logger.Debugf("[PENELOPA] Cleaned up %d old series", removed)
	}
}

// getMemoryMetrics returns memory-related metrics
func (o *Output) getMemoryMetrics() []promwrite.TimeSeries {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memTags := map[string]string{"source": "k6", "testid": o.testid, "pod": o.pod}
	now := time.Now()
	
	return []promwrite.TimeSeries{
		{Labels: MapStaticLabels("k6_mem_alloc_mb", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.Alloc) / 1024.0 / 1024.0}},
		{Labels: MapStaticLabels("k6_mem_heapalloc_mb", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.HeapAlloc) / 1024.0 / 1024.0}},
		{Labels: MapStaticLabels("k6_mem_heap_sys_mb", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.HeapSys) / 1024.0 / 1024.0}},
		{Labels: MapStaticLabels("k6_mem_heap_idle_mb", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.HeapIdle) / 1024.0 / 1024.0}},
		{Labels: MapStaticLabels("k6_mem_heap_inuse_mb", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.HeapInuse) / 1024.0 / 1024.0}},
		{Labels: MapStaticLabels("k6_mem_stack_inuse_mb", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.StackInuse) / 1024.0 / 1024.0}},
		{Labels: MapStaticLabels("k6_mem_stack_sys_mb", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.StackSys) / 1024.0 / 1024.0}},
		{Labels: MapStaticLabels("k6_mem_gc_cpu_fraction", memTags), Sample: promwrite.Sample{Time: now, Value: m.GCCPUFraction}},
		{Labels: MapStaticLabels("k6_mem_gc_pause_ns", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.PauseTotalNs)}},
		{Labels: MapStaticLabels("k6_mem_gc_count", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.NumGC)}},
		{Labels: MapStaticLabels("k6_mem_objects", memTags), Sample: promwrite.Sample{Time: now, Value: float64(m.Mallocs - m.Frees)}},
	}
} 
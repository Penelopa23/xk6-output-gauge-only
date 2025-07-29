package penelopa

import (
	"log"
	"net/http"
	"strings"
)

// LoggingRoundTripper is a custom HTTP transport that logs requests
type LoggingRoundTripper struct {
	rt http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface with logging
func (l *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := l.rt.RoundTrip(req)
	if resp != nil {
		log.Printf("[PENELOPA] HTTP status: %d", resp.StatusCode)
	}
	return resp, err
}

// makeSeriesKey creates a unique key for a series based on name and tags
func (o *Output) makeSeriesKey(name string, tags map[string]string) string {
	b := strings.Builder{}
	b.WriteString(name)
	for k, v := range tags {
		b.WriteString("|")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(v)
	}
	return b.String()
} 
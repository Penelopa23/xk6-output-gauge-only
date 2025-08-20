package penelopa

import (
	"log"
	stdhttp "net/http"
)

// LoggingRoundTripper is a custom HTTP transport that logs requests
type LoggingRoundTripper struct {
	rt stdhttp.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface with logging
func (l *LoggingRoundTripper) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	resp, err := l.rt.RoundTrip(req)
	if resp != nil {
		log.Printf("[PENELOPA] HTTP status: %d", resp.StatusCode)
	}
	return resp, err
}

 
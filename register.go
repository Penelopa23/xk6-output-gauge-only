// Package penelopa registers the xk6-output-penelopa extension
package penelopa

import (
	"xk6-output-penelopa/pkg/penelopa"
	"go.k6.io/k6/output"
)

func init() {
	output.RegisterExtension("penelopa", func(p output.Params) (output.Output, error) {
		return penelopa.New(p)
	})
} 
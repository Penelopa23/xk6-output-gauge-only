package penelopa

import (
	"go.k6.io/k6/output"
)

func init() {
	output.RegisterExtension("penelopa", func(p output.Params) (output.Output, error) {
		return New(p)
	})
}

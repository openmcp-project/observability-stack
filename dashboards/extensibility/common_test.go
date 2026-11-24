package extensibility

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func Test_prefixesToRegex(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		prefix []string
		want   string
	}{
		{
			name:   "no prefix",
			prefix: []string{},
			want:   "",
		},
		{
			name:   "single prefix",
			prefix: []string{"sp-"},
			want:   "sp-.*",
		},
		{
			name:   "multiple prefixes",
			prefix: []string{"sp-", "cp-", "ps-"},
			want:   "sp-.*|cp-.*|ps-.*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prefixesToRegex(tt.prefix...)
			assert.Equal(t, tt.want, got)
		})
	}
}

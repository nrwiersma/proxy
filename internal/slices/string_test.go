package slices_test

import (
	"testing"

	"github.com/nrwiersma/proxy/internal/slices"
	"github.com/stretchr/testify/assert"
)

func TestStringContains(t *testing.T) {
	tests := []struct {
		name   string
		string string
		slice  []string
		want   bool
	}{
		{
			name:   "Found",
			string: "foo",
			slice:  []string{"foo", "bar"},
			want:   true,
		},
		{
			name:   "Not Found",
			string: "test",
			slice:  []string{"foo", "bar"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.StringContains(tt.string, tt.slice)

			assert.Equal(t, tt.want, got)
		})
	}
}

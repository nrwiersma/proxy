package http_test

import (
	"bytes"
	"testing"

	"github.com/nrwiersma/proxy/http"
	"github.com/stretchr/testify/assert"
)

func TestHeader_Get(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		key    string
		want   string
	}{
		{
			name:   "Gets Single Header",
			header: http.Header{"Host": []string{"test-host"}},
			key:    "Host",
			want:   "test-host",
		},
		{
			name:   "Gets First In Multi-Header",
			header: http.Header{"Host": []string{"test-host", "other"}},
			key:    "Host",
			want:   "test-host",
		},
		{
			name:   "Gets Canonicalised Heafer",
			header: http.Header{"Host": []string{"test-host", "other"}},
			key:    "host",
			want:   "test-host",
		},
		{
			name:   "Returns Empty String If Doesnt Exist",
			header: http.Header{"Host": []string{"test-host", "other"}},
			key:    "Something",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.header.Get(tt.key)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHeader_Set(t *testing.T) {
	h := http.Header{}

	h.Set("foo", "bar")

	assert.Equal(t, http.Header{"Foo": []string{"bar"}}, h)
}

func TestHeader_Del(t *testing.T) {
	h := http.Header{"Foo": []string{"bar"}}

	h.Del("Foo")
	h.Del("Test")

	assert.Equal(t, http.Header{}, h)
}

func TestHeader_Write(t *testing.T) {
	h := http.Header{
		"Host":       []string{"something"},
		"Other":      []string{"foo", "bar"},
		"Connection": []string{"close"},
	}

	buf := bytes.NewBuffer(nil)
	err := h.Write(buf)

	if assert.NoError(t, err) {
		want := "Connection: close\r\nHost: something\r\nOther: foo\r\nOther: bar\r\n"
		assert.Equal(t, want, buf.String())
	}
}

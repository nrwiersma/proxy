package proxy_test

import (
	"context"
	"testing"

	"github.com/nrwiersma/proxy/http"
	"github.com/nrwiersma/proxy/http/proxy"
	"github.com/stretchr/testify/mock"
)

func TestRRLoadBalancer_ServeHTTP(t *testing.T) {
	h1 := new(MockHandler)
	h1.On("ServeHTTP", mock.Anything, mock.Anything).Twice().Return(nil)
	h2 := new(MockHandler)
	h2.On("ServeHTTP", mock.Anything, mock.Anything).Once().Return(nil)

	bal := proxy.NewRRLoadBalancer([]http.Handler{h1, h2})

	for i := 0; i < 3; i++ {
		bal.ServeHTTP(context.Background(), nil)
	}

	h1.AssertExpectations(t)
	h2.AssertExpectations(t)
}

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) ServeHTTP(ctx context.Context, r *http.Request) *http.Response {
	args := m.Called(ctx, r)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*http.Response)
}
